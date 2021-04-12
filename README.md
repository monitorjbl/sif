# sif

![sif](docs/sif.png)

Protects you from ~~the abyss~~ dependency bloat by showing you where your largest dependencies are coming from. Large
dependencies are highlighted for you and transitive are included.

Sif supports multiple kinds of builds and languages:

* Maven
* Gradle (planned)
* NPM (planned)

Sif can also be run on multiple platforms (not tested)

* OSX
* Linux
* Windows

## Maven

```
Usage:
  sif maven [options] path/to/pom.xml [flags]

Flags:
      --child string   Specifies a child module in a multi-module project
      --cmd string     Path to Maven command (default "mvn")
  -h, --help           help for maven
      --repo string    The location of the Maven repository to use (default "~/.m2/repository")
      --scope string   The project scope to use (default "compile")
```

## Gradle

Not supported yet

## NPM

Not supported yet

## Building sif

```shell
# Compile for your specific platform
go mod download
go build

# Cross platform builds
GOOS=darwin GOARCH=amd64 go build -o sif-darwin-x64
GOOS=linux GOARCH=amd64 go build -o sif-linux-x64
GOOS=windows GOARCH=amd64 go build -o sif-windows-x64.exe
```