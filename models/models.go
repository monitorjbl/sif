package models

type RootCtx struct {
	LogLevel                      string
	LargeDependencyThreshold      string
	LargeDependencyThresholdBytes uint64
}

type Dependency struct {
	GroupId    string
	ArtifactId string
	Version    string
	Extension  string
	Size       uint64
	Children   []Dependency
}

type Project struct {
	Name         string
	Version      string
	Dependencies []Dependency
}
