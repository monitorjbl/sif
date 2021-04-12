package main

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"path/filepath"
	"sif/maven"
	"sif/models"
	"strings"
)

var (
	rootCmd = &cobra.Command{
		Use:   "sif",
		Short: "A dependency analyzer for software projects",
	}
	rootCtx  = models.RootCtx{}
	mavenCtx = maven.Maven{}
)

type AnalyzedDependency struct {
	Dependency *models.Dependency
	Parent     *AnalyzedDependency
	Children   *[]AnalyzedDependency
	Depth      int
	TotalSize  uint64
}
type DependencyStack []*AnalyzedDependency

func (s DependencyStack) Push(v *AnalyzedDependency) DependencyStack {
	return append(s, v)
}

func (s DependencyStack) Pop() (DependencyStack, *AnalyzedDependency) {
	l := len(s)
	return s[:l-1], s[l-1]
}

func resolvePath(path string) string {
	resolved, err := homedir.Expand(path)
	if err != nil {
		log.Fatalf("Failed to resolve path with home dir: %s: %s", path, err)
	}
	absolute, err := filepath.Abs(resolved)
	if err != nil {
		log.Fatalf("Failed to resolve absolute path: %s: %s", absolute, err)
	}
	return absolute
}

func processRootConfig() models.RootCtx {
	b, err := humanize.ParseBytes(rootCtx.LargeDependencyThreshold)
	if err != nil {
		log.Fatalf("Unable to parse threshold %s as a size", rootCtx.LargeDependencyThreshold)
	}
	rootCtx.LargeDependencyThresholdBytes = b

	switch strings.ToUpper(rootCtx.LogLevel) {
	case "TRACE":
		log.SetLevel(log.TraceLevel)
		rootCtx.LogLevel = "TRACE"
		break
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
		rootCtx.LogLevel = "DEBUG"
		break
	case "INFO":
		log.SetLevel(log.InfoLevel)
		rootCtx.LogLevel = "INFO"
	default:
		log.Fatalf("Unknown log level: %s", rootCtx.LogLevel)
	}
	return rootCtx
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&rootCtx.LogLevel,
		"logging",
		"",
		"INFO",
		"The level of logging to use")
	rootCmd.PersistentFlags().StringVarP(&rootCtx.LargeDependencyThreshold,
		"large-threshold",
		"",
		"3MB",
		"The location of the Maven repository to use")
	rootCmd.PersistentFlags().BoolVarP(&rootCtx.LargeDependenciesOnly,
		"large-deps-only",
		"",
		false,
		"Only show dependency trees that exceed the threshold")

	var mavenCmd = cobra.Command{
		Use:   "maven [options] path/to/pom.xml",
		Short: "Analyzes a Maven project's dependencies",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 || args[0] == "help" {
				cmd.Help()
			} else {
				mavenCtx.PomFile = resolvePath(args[0])
				mavenCtx.MavenRepo = resolvePath(mavenCtx.MavenRepo)
				mavenCtx.RootCtx = processRootConfig()
				printResult(mavenCtx.Analyze())
			}
		},
	}
	mavenCmd.PersistentFlags().StringVarP(&mavenCtx.MavenCommand,
		"cmd",
		"",
		"mvn",
		"Path to Maven command")
	mavenCmd.PersistentFlags().StringVarP(&mavenCtx.Scope,
		"scope",
		"",
		"compile",
		"The project scope to use")
	mavenCmd.PersistentFlags().StringVarP(&mavenCtx.MavenRepo,
		"repo",
		"",
		"~/.m2/repository",
		"The location of the Maven repository to use")
	mavenCmd.PersistentFlags().StringVarP(&mavenCtx.ChildModule,
		"child",
		"",
		"",
		"Specifies a child module in a multi-module project")
	rootCmd.AddCommand(&mavenCmd)
}

func initConfig() {
}

func calculateTotalSizes(project models.Project) []AnalyzedDependency {
	// Convert top-level deps into analyzed form
	var deps []AnalyzedDependency
	for _, e := range project.Dependencies {
		var dep = e
		deps = append(deps, AnalyzedDependency{
			Dependency: &dep,
			Parent:     nil,
			Depth:      0,
			TotalSize:  0,
		})
	}

	// Push all top-level deps
	var stack DependencyStack
	for i := len(deps) - 1; i >= 0; i-- {
		stack = stack.Push(&deps[i])
	}

	for len(stack) > 0 {
		var entry *AnalyzedDependency
		stack, entry = stack.Pop()

		var childDeps []AnalyzedDependency
		for i := 0; i < len(entry.Dependency.Children); i++ {
			e := entry.Dependency.Children[i]
			childDeps = append(childDeps, AnalyzedDependency{
				Dependency: &e,
				Parent:     entry,
				Depth:      entry.Depth + 1,
				TotalSize:  0,
			})
		}
		for i := len(childDeps) - 1; i >= 0; i-- {
			stack = stack.Push(&childDeps[i])
		}
		entry.Children = &childDeps

		dep := entry.Dependency
		ptr := entry
		for ptr != nil {
			ptr.TotalSize += dep.Size
			ptr = ptr.Parent
		}
	}
	return deps
}

func printResult(project models.Project) {
	log.Infof("Project: %s (%s)", project.Name, project.Version)

	if len(project.Dependencies) > 0 {
		// Depth-first stack walk, printing as we go
		analyzedDeps := calculateTotalSizes(project)
		var stack DependencyStack

		// Insert these in reverse because stacks operate on the last inserted record
		for i := len(analyzedDeps) - 1; i >= 0; i-- {
			stack = stack.Push(&analyzedDeps[i])
		}

		topLevelCount := 0
		var totalDeps uint64 = 0
		var totalSize uint64 = 0
		var currTopLevel *AnalyzedDependency
		for len(stack) > 0 {
			var entry *AnalyzedDependency
			stack, entry = stack.Pop()
			dep := entry.Dependency
			totalDeps++
			totalSize += dep.Size

			// The prefix is dependent on the next item in the stack. If the next
			// item is at the same depth, then we need to include pipes to extend
			// the tree downwards. If the next item is not at the same depth, then
			// we need to use the angle character.
			prefix := "├── "
			if entry.Depth > 0 {
				if len(stack) > 0 && stack[len(stack)-1].Depth == entry.Depth {
					prefix = fmt.Sprintf("%s├── ", strings.Repeat("|    ", entry.Depth))
				} else {
					prefix = fmt.Sprintf("%s└── ", strings.Repeat("|    ", entry.Depth))
				}
			} else {
				topLevelCount++
				currTopLevel = entry
				if len(stack) == 0 && len(*entry.Children) == 0 {
					prefix = "└── "
				}
			}

			// Highlight any file that is greater than than the large file threshold
			fileColor := color.New(color.Reset)
			totalColor := color.New(color.Reset)
			if dep.Size > rootCtx.LargeDependencyThresholdBytes {
				fileColor = color.New(color.BgRed)
			}
			if entry.TotalSize > rootCtx.LargeDependencyThresholdBytes {
				totalColor = color.New(color.BgRed)
			}

			if !rootCtx.LargeDependenciesOnly || currTopLevel.TotalSize > rootCtx.LargeDependencyThresholdBytes {
				log.Infof("%s%s:%s:%s Size[%s, %s]",
					prefix,
					dep.GroupId,
					dep.ArtifactId,
					dep.Version,
					fileColor.Sprintf("File: %s", humanize.Bytes(dep.Size)),
					totalColor.Sprintf("Total: %s", humanize.Bytes(entry.TotalSize)))
			}

			// Push all child dependencies in reverse order since stacks operate on the
			// last inputted value
			for i := len(*entry.Children) - 1; i >= 0; i-- {
				stack = stack.Push(&(*(*entry).Children)[i])
			}
		}

		log.Infof("%s in %d dependencies", humanize.Bytes(totalSize), totalDeps)
	} else {
		log.Infof("0MB in 0 dependencies")
	}
}

type LogFormatter struct {
}

func (*LogFormatter) Format(entry *log.Entry) ([]byte, error) {
	if entry.Level >= log.DebugLevel {
		return []byte(color.New(color.FgWhite).Sprintf("%s\n", entry.Message)), nil
	} else {
		return []byte(color.New(color.Reset).Sprintf("%s\n", entry.Message)), nil
	}
}

func main() {
	log.SetFormatter(&LogFormatter{})
	rootCmd.Execute()
}
