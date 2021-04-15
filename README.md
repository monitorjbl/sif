# sif

![sif](docs/sif.png)

[image source](https://www.deviantart.com/birchwing/art/Crossover-Homebrew-Sif-Dark-Souls-792426892)

Protects you from ~~the abyss~~ dependency bloat by showing you where your largest dependencies are coming from. Large
dependencies are highlighted for you and transitives are included.

Sif supports multiple kinds of builds and languages:

* Maven
* Gradle (planned)
* NPM (planned)

Sif can also be run on multiple platforms

* OSX
* Linux (not tested)
* Windows (not tested)

# Installing

You can download a release from the [Releases]() tabl, or you can run these shell commands to download the latest for
you:

**OSX**

```shell
mkdir -p ~/bin && \
curl -s https://api.github.com/repos/monitorjbl/sif/releases/latest \
| grep "browser_download_url.*darwin-x64" \
| cut -d : -f 2,3 \
| tr -d \" \
| xargs curl -L -o ~/bin/sif &&\
chmod +x ~/bin/sif
```

**Linux**

```shell
curl -s https://api.github.com/repos/monitorjbl/sif/releases/latest \
| grep "browser_download_url.*linux-x64" \
| cut -d : -f 2,3 \
| tr -d \" \
| xargs curl -L -o ./sif && chmod +x ./sif
```

**Windows**

```
“¯\_(ツ)_/¯“

(i dont know powershell very well)
```

# Running

sif can support any build system that it can call externally and parse the result from.
Just use the subcommand that corresponds to your project's build process.

Currently, only Maven is supported, but more will be coming soon!

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

# Building

```shell
# Compile for your specific platform (in project root)
go mod download
go scripts

# Compile cross platform builds (in dist directory)
./build.sh

# Release (requires release-it tool)
release-it
```

# TODO

* Support Gradle builds
* Support NPM builds
* Support multiple output formats