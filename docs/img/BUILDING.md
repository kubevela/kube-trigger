# How to build?

This document describes how to build this project and briefly introduces the build system.

## Basic commands

This is a list of basic build commands that can get you started.

- `make`, `make build`: build `trigger` and `manager` binary into `bin`. The exact binary name can be seen in the build output.
- `make container`: build `trigger` and `manager` containers. Tags can be seen in the build output.
- `make clean`: clean built binaries
- `make test`: run tests
- `make reviewable`: check possible issues before committing code, make your code ready to review

## Dependencies

Chances are all the dependencies are already met on a dev machine.

- `GNU Make`: apparently, we need to use make.
- `git`: to set binary version according to git tag. If not present, version will be UNKNOWN.
- `docker`: to build Docker containers, or if you want to use the containerized build environment.
- `go`: only if you want use your local Go SDK. If you use the containerized build environment to build, you don't need Go.
- `tar/zip`: to package binaries
- `sha256sum`: to calculate checksum

> This has only been tested on (mostly) Linux and macOS. Windows support is very unlikely to come.

## Make targets

This list only includes part of the full make targets, to see to full list of targets, run `make all-help`.

### General targets

General targets can be executed by `make <target>`, e.g., `make help`.

- `help`: show general help message, including general targets
- `all-help`: show help messages for all subjects
- `reviewable`: check possible issues before committing code, make your code ready to review
- `test`: run tests
- ...omitted

### Subproject targets

Subproject targets can be executed by `make <subproject>-<target>`. This project have two subprojects: `trigger` and `manager`. So you can run `make trigger-help` to see help messages for `trigger` subproject.

Running `make <subproject>` without any target will execute the default target `build`, which is equivalent to `make <subproject>-build`. For example, running `make trigger` will build `trigger` build binary for current platform.

This is a list of common subproject targets. Some subprojects may have specific targets.


```
build                    (default) build binary for current platform
all-build                build binaries for all platforms
package                  build and package binary for current platform
all-package              build and package binaries for all platforms
container                build container image for current platform
container-push           push built container image to all repos
all-container-push       build and push container images for all platforms
shell                    launch a shell in the build container
clean                    clean built binaries
all-clean                clean built binaries, build cache, and helper tools
version                  output the version string
imageversion             output the container image version
binaryname               output current artifact binary name
variables                print makefile variables
help                     print this message
```

### Common targets

Common targets is a list of common subproject targets, which you have seen above, to run on all subprojects.

For example, to build containers for all subprojects (`trigger` and `manager`), instead of calling `make trigger-container` and `make manager-container`, you can just call `make container`.

This is a list of common targets that can be used in this way.

```
build                    (default) build binary for current platform
all-build                build binaries for all platforms
package                  build and package binary for current platform
all-package              build and package binaries for all platforms
container                build container image for current platform
container-push           push built container image to all repos
all-container-push       build and push container images for all platforms
clean                    clean built binaries
all-clean                clean built binaries, build cache, and helper tools
version                  output the version string
imageversion             output the container image version
binaryname               output current artifact binary name
variables                print makefile variables
```

## Advanced

#### Build environment

By default, if you have Go in your PATH, it will use you local Go SDK. If you don't have Go, it will use a containerized build environment, specifically, a `golang:1.xx-alpine` Docker image. The actual `1.xx` go version will be determined by your `go.mod`. To manually specify the image of the containerized build environment, set `BUILD_IMAGE` to the Docker image you want, e.g. `golang:1.20`.

To make sure the build environment is the same across your teammates and avoid problems, it is recommended to use the containerized build environment. To forcibly use the containerized build environment, set `USE_BUILD_CONTAINER` to `1`. For example, `USE_BUILD_CONTAINER=1 make all-build`.

#### Debug build

Set `DEBUG` to `1` to build binary for debugging (disable optimizations and inlining), otherwise it will build for release (trim paths, disable symbols and DWARF).

#### Environment variables

These environment variables will be passed to the containerized build environment: `GOFLAGS`, `GOPROXY`, `HTTP_PROXY`, `HTTPS_PROXY`.

Setting `GOOS` and `GOARCH` will do cross-compiling, even when using the containerized build environment, just like you would expect.

#### Container base image

To build your container (I'm referring to the container artifact of your application, not the build container), you will need a base image. By default, this will be `gcr.io/distroless/static:nonroot`.

To customize it, set `BASE_IMAGE` to your desired image. For example, `BASE_IMAGE=scratch make container`.

#### Versioning

By default, `pkg/version.Version` will be determined by `git describe --tags --always --dirty`. For example, if you are on a git commit tagged as `v0.0.1`, then `pkg/version.Version` will be `v0.0.1`. The tag of your container will be `v0.0.1` as well. If you are not on that exact tagged commit, then the version will be something like `v0.0.1-1-b5f5feb`. The tag of your container will be `latest`.

To set the version manually, set `VERSION` to something you want. This will affect `pkg/version.Version`. Set `IMAGE_TAG` to the Docker image tag you want too.

#### Binary names

After building, two binaries will be generated. For example, one is `bin/foo`, another one is `bin/foo-v0.0.1/foo-v0.0.1-linux-amd64`. The long one is the compiler output. The short one is provided so you can use it more easily, and is hard-linked to the long one.

#### Makefile debugging

To show debug messages for makefile, set `DBG_MAKEFILE` to `1`.
