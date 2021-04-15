package gradle

import (
	"bufio"
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sif/models"
	"strings"
)

var (
	regexProjectName    = regexp.MustCompile("name: (.+)")
	regexProjectVersion = regexp.MustCompile("version: (.+)")
	dependencyTreeRegex = regexp.MustCompile("(\\|\\s\\s\\s)*(\\+---|\\\\---) (.+)")
	dependencyRegex     = regexp.MustCompile("([^:\\s]+):([^:\\s]+):([^:\\s]+)$")
)

type Gradle struct {
	RootCtx         models.RootCtx
	BuildGradleFile string
	Configuration   string
	GradleCommand   string
	ChildModule     string
}

func (g *Gradle) describeError(errMsg string) {
	log.Tracef("Error message: %s", errMsg)
	log.Debugf("Unknown error:\n%s", errMsg)
}

func (g *Gradle) parseProjectDetails() (string, string) {
	cmd := exec.Command(g.GradleCommand,
		"-p",
		g.BuildGradleFile,
		"properties")
	out, err := cmd.CombinedOutput()
	output := string(out)
	if err != nil {
		log.Error(err)
		g.describeError(output)
	}

	log.Debug(output)
	nameResult := regexProjectName.FindStringSubmatch(output)
	versionResult := regexProjectVersion.FindStringSubmatch(output)
	return nameResult[1], versionResult[1]
}

// Search for gradle executable to use. We will first look to see if there
// is a gradlew/gradlew.bat file in the directory specified. If none is
// found, use the "gradle" command
func (g *Gradle) findGradleExecutable() string {
	log.Debugf("Searching for gradle executable")
	f, err := os.Stat(g.BuildGradleFile)
	if err != nil {
		log.Fatalf("Unable to read Gradle file or directory: %s", err)
	}

	// Figure out the project directory path. If we were given a file, assume it
	// was a build.gradle file and the project is in the parent directory. If it
	// is a directory, assume it is the project directory.
	var projectDir string
	if f.IsDir() {
		projectDir = g.BuildGradleFile
	} else {
		projectDir = filepath.Dir(g.BuildGradleFile)
	}

	files, err := ioutil.ReadDir(projectDir)
	if err != nil {
		log.Fatalf("Failed to read files in directory: %s", err)
	}

	// List files in project directory and search for gradlew or gradlew.bat
	for _, f := range files {
		if (f.Name() == "gradlew.bat" && runtime.GOOS == "windows") || (f.Name() == "gradlew" && runtime.GOOS != "windows") {
			bin := path.Join(projectDir, f.Name())
			log.Debugf("Found %s to run build", bin)
			return bin
		}
	}

	// If we find nothing, just assume it's "gradle"
	log.Debugf("No executable found, assuming that gradle is available on the PATH")
	return "gradle"
}

func (g *Gradle) parseDependency(output string) *models.Dependency {
	dep := dependencyTreeRegex.FindStringSubmatch(output)[3]

	// Gradle dependencies have a variable format, which is a bit complex to parse. Also,
	// the tree is not limited to just the transitives that the build will actually use
	// (a la Maven), it shows *all* dependencies. Below are the formats that can be seen:
	//
	//	* Just a plain dependency		 :		<groupId>:<artifactId>:<version>
	//	* Version forced-changed		 :		<groupId>:<artifactId>:<version> -> <newVersion>
	//	* Omitted due to previous listing:		<groupId>:<artifactId>:<version> [-> <newVersion] (*)
	//  * Dependency constrained		 :		<groupId>:<artifactId>:<version> [-> <newVersion] (c)
	//
	// We want to only show unique dependencies that the build will actually use. Because
	// of this, we can safely ignore all but the plain dependencies.
	res := dependencyRegex.FindStringSubmatch(dep)
	if res == nil {
		return nil
	}

	return &models.Dependency{
		GroupId:    res[1],
		ArtifactId: res[2],
		Version:    res[3],
		Size:       1,
	}
}

func (g *Gradle) parseOutputTree(output string) []models.Dependency {
	// Remove everything except the tree output. The first line
	// that matches this regex is where we should begin.
	var lines = strings.Split(output, "\n")
	var startLine = 0
	var endLine = 0
	var startRegex = regexp.MustCompile(fmt.Sprintf("%s - Runtime classpath of .+", g.Configuration))
	var endRegex = regexp.MustCompile("\\(c\\) - dependency constraint")
	for lineNum, line := range lines {
		if startRegex.MatchString(line) {
			startLine = lineNum
		}
		if endRegex.MatchString(line) {
			endLine = lineNum - 1
		}
	}

	// The dependency tree output is in ordered form, so extracting is easy. We
	// keep track of each top-level dep and for each child entry, we just find
	// the last entry in the toplevel and walk down its children, using the
	// last entry in each one until we reach the depth indicated.
	depTreeEntries := lines[startLine:endLine]
	var dependencies []models.Dependency
	for _, entry := range depTreeEntries {
		// Determine depth of the current line. dependencies has a specific
		// format it uses to indicate parent-child relationships:
		//
		//	- +---		: 	Indicates a top-level dependency
		//	- \---		: 	Indicates the last top-level dependency
		//	- |  +--- 	: 	Indicates a child dependency. The depth is equal to the number of | chars
		//	- |  \---	:	Indicates the last entry at the current level. The depth is equal to the number of | chars
		if dependencyTreeRegex.MatchString(entry) {
			// Count the pipes
			depth := strings.Count(entry, "|    ")
			if depth > 0 {
				// We are mutating array contents, so we can't use normal assignment or Go will
				// transparently copy the data
				curr := &dependencies
				for i := 1; i < depth-1; i++ {
					curr = &(*curr)[len(*curr)-1].Children
				}

				var parent = &(*curr)[len(*curr)-1]

				dep := g.parseDependency(entry)
				if dep != nil {
					parent.Children = append(parent.Children, *dep)
				}
			} else {
				dep := g.parseDependency(entry)
				if dep != nil {
					dependencies = append(dependencies, *dep)
				}
			}
		}
	}

	return dependencies
}

func (g *Gradle) Analyze() models.Project {
	if g.RootCtx.LogLevel == "DEBUG" {
		log.Debug("Logging Gradle command output")
	}

	if g.GradleCommand == "" {
		g.GradleCommand = g.findGradleExecutable()
	}

	log.Infof("Running Gradle command (%s)", g.GradleCommand)

	// Run dependency:tree tool
	var out bytes.Buffer
	cmd := exec.Command(g.GradleCommand,
		"-p",
		g.BuildGradleFile,
		"-q",
		fmt.Sprintf("%s:dependencies", g.ChildModule),
		"--configuration",
		g.Configuration)
	stdout, err := cmd.StdoutPipe()
	scanner := bufio.NewScanner(stdout)
	running := true

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			log.Debug(line)
			out.WriteString(line)
			out.Write([]byte("\n"))
		}
		if err := scanner.Err(); err != nil {
			log.Error(os.Stderr, "reading standard input:", err)
		}
		running = false
	}()

	err = cmd.Run()
	output := out.String()
	if err != nil {
		log.Error(err)
		g.describeError(output)
	}

	deps := g.parseOutputTree(output)
	name, version := g.parseProjectDetails()
	return models.Project{
		Name:         name,
		Version:      version,
		Dependencies: deps,
	}
}
