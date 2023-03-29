# How to build?

This document describes how to build this project and briefly introduces the build system.

## Basic commands



## Dependencies

Chances are all the dependencies are already met on a dev machine.

- `GNU Make`: apparently, we need to use make.
- `git`: to set binary version according to git tag. If not present, version will be UNKNOWN.
- `docker`: to build Docker containers, or if you want to use the containerized build environment.
- `go`: only if you want use your local Go SDK. If you use the containerized build environment to build, you don't need Go.
- `tar/zip`: to package binaries
- `sha256sum`: to calculate checksum

