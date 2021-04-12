# sif

![sif](docs/sif.png)

[image source](https://www.deviantart.com/birchwing/art/Crossover-Homebrew-Sif-Dark-Souls-792426892)

Protects you from ~~the abyss~~ dependency bloat by showing you where your largest dependencies are coming from. Large
dependencies are highlighted for you and transitive are included.

Sif supports multiple kinds of builds and languages:

* Maven
* Gradle (planned)
* NPM (planned)

Sif can also be run on multiple platforms 

* OSX
* Linux (not tested)
* Windows (not tested)

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
# Compile for your specific platform (in project root)
go mod download
go scripts

# Compile cross platform builds (in dist directory)
./build.sh

# Release (requires release-it tool)
release-it
```