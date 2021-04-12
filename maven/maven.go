package maven

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"regexp"
	"sif/models"
	"strings"
	"time"
)

var (
	regexProjectDetails    = regexp.MustCompile("\\[INFO\\] Building ([^\\s]+) ([^\\s]+)")
	regexErrNonreadablepom = regexp.MustCompile(".* Non-readable POM.*")
	dependencyRegex        = regexp.MustCompile("(\\|\\s\\s)*(\\+-|\\\\-) (.+)")
)

type Maven struct {
	RootCtx      models.RootCtx
	PomFile      string
	Scope        string
	MavenCommand string
	MavenRepo    string
}

func (m *Maven) describeError(errMsg string) {
	log.Tracef("Error message: %s", errMsg)

	if regexErrNonreadablepom.MatchString(errMsg) {
		log.Fatalf("POM was not found at %s", m.PomFile)
	} else {
		log.Fatalf("Unknown error encountered: %s", errMsg)
	}
}

func (m *Maven) determineFileSize(dep *models.Dependency) models.Dependency {
	groupPath := strings.ReplaceAll(dep.GroupId, ".", "/")
	file, _ := homedir.Expand(fmt.Sprintf("%s/%s/%s/%s/%s-%s.%s",
		m.MavenRepo,
		groupPath,
		dep.ArtifactId,
		dep.Version,
		dep.ArtifactId,
		dep.Version,
		dep.Extension))
	stats, err := os.Stat(file)
	if err != nil {
		log.Fatalf("Failed to read file at %s", file)
	}
	dep.Size = uint64(stats.Size())
	return *dep
}

func (m *Maven) parseMavenCoordinates(entry string) models.Dependency {
	r := dependencyRegex.FindStringSubmatch(entry)
	depString := r[3]
	split := strings.Split(depString, ":")
	return m.determineFileSize(&models.Dependency{
		GroupId:    split[0],
		ArtifactId: split[1],
		Extension:  split[2],
		Version:    split[3],
		Size:       0,
	})
}

func (m *Maven) parseProjectDetails(output string) (string, string) {
	var r = regexProjectDetails.FindStringSubmatch(output)
	return r[1], r[2]
}

func (m *Maven) parseOutputTree(output string) []models.Dependency {
	// Remove everything except the tree output. The first line
	// that matches this regex is where we should begin.
	var lines = strings.Split(output, "\n")
	var startLine = 0
	var endLine = 0
	var startRegex = regexp.MustCompile("\\[INFO\\] --- maven-dependency-plugin:.+:tree .*")
	var endRegex = regexp.MustCompile("\\[INFO\\] BUILD SUCCESS.*")
	for lineNum, line := range lines {
		if startRegex.MatchString(line) {
			startLine = lineNum
		}
		if endRegex.MatchString(line) {
			endLine = lineNum - 1
		}
	}

	// Extract the tree from the output, removing the [INFO] bit
	var depTreeEntries []string
	for i := startLine; i < endLine; i++ {
		l := strings.Split(lines[i], "[INFO] ")
		depTreeEntries = append(depTreeEntries, l[1])
	}

	// The dependency tree output is in ordered form, so extracting is easy. We
	// keep track of each top-level dep and for each child entry, we just find
	// the last entry in the toplevel and walk down its children, using the
	// last entry in each one until we reach the depth indicated.
	var dependencies []models.Dependency
	for _, entry := range depTreeEntries {
		// Determine depth of the current line. dependency:tree has a specific
		// format it uses to indicate parent-child relationships:
		//
		//	- +-	: 	Indicates a top-level dependency
		//	- \-	: 	Indicates the last top-level dependency
		//	- |  +- : 	Indicates a child dependency. The depth is equal to the number of | chars
		//	- |  \-	:	Indicates the last entry at the current level. The depth is equal to the number of | chars
		if dependencyRegex.MatchString(entry) {
			// Count the pipes
			depth := strings.Count(entry, "|  ")
			if depth > 0 {
				// We are mutating array contents, so we can't use normal assignment or Go will
				// transparently copy the data
				curr := &dependencies
				for i := 1; i < depth-1; i++ {
					curr = &(*curr)[len(*curr)-1].Children
				}

				var parent = &(*curr)[len(*curr)-1]
				parent.Children = append(parent.Children, m.parseMavenCoordinates(entry))
			} else {
				dependencies = append(dependencies, m.parseMavenCoordinates(entry))
			}
		}

	}

	return dependencies
}

func (m *Maven) Analyze() models.Project {
	log.Infof("Running Maven command (%s)", m.MavenCommand)
	if m.RootCtx.LogLevel == "DEBUG" {
		log.Debug("Logging Maven command output")
	}

	// Run dependency:tree tool
	var out bytes.Buffer
	cmd := exec.Command("mvn", "dependency:tree", "-f", m.PomFile)
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
	for running {
		time.Sleep(25 * time.Millisecond)
	}
	var output = out.String()
	if err != nil {
		m.describeError(output)
		log.Fatal(err)
	}

	// Parse output
	log.Infof("Parsing output")
	deps := m.parseOutputTree(output)
	name, version := m.parseProjectDetails(output)
	return models.Project{
		Name:         name,
		Version:      version,
		Dependencies: deps,
	}
}
