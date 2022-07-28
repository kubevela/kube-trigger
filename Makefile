# Copyright 2022 The KubeVela Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

DBG_MAKEFILE ?=
ifeq ($(DBG_MAKEFILE),1)
    $(warning ***** starting Makefile for goal(s) "$(MAKECMDGOALS)")
    $(warning ***** $(shell date))
else
    # If we're not debugging the Makefile, don't echo recipes.
    MAKEFLAGS += -s
endif

MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --always-make

ALL_PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

OS := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
ARCH := $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))
# Use git tags to set the version string
VERSION ?= $(shell git describe --tags --always --dirty)

BIN_EXTENSION :=
ifeq ($(OS), windows)
  BIN_EXTENSION := .exe
endif

BIN := kube-trigger
_OUT := $(BIN)$(BIN_EXTENSION)
_VER_OUT := $(BIN)-$(VERSION)-$(OS)-$(ARCH)$(BIN_EXTENSION)
# If the user set FULL_NAME, we will output the binary with detailed info.
# e.g. kube-trigger-v0.0.1-linux-amd64
BIN_FULLNAME := $(if $(FULL_NAME),$(_VER_OUT),$(_OUT))
OUTPUT := bin/$(BIN_FULLNAME)
ENTRY := cmd/kubetrigger/main.go

REGISTRY ?= ghcr.io/oam-dev
IMG ?= $(REGISTRY)/$(BIN):$(VERSION)

GOFLAGS ?=
GOPROXY ?=

# Use bash explicitly
SHELL := /usr/bin/env bash -o errexit -o pipefail -o nounset

all: build

build: # @HELP clean-build binary locally
build:
	ARCH=$(ARCH)                     \
	    OS=$(OS)                     \
	    OUTPUT=$(OUTPUT)             \
	    VERSION=$(VERSION)           \
	    GOFLAGS=$(GOFLAGS)           \
	    bash build/build.sh $(ENTRY)

all-build: # @HELP builds binaries for all platforms, binary names will use full name
all-build: $(addprefix build-, $(subst /,_, $(ALL_PLATFORMS)))

build-%:
	$(MAKE) build                          \
	    --no-print-directory               \
	    GOOS=$(firstword $(subst _, ,$*))  \
	    GOARCH=$(lastword $(subst _, ,$*)) \
	    FULL_NAME=1

dirty-build: # @HELP same as build, but using build cache is allowed
dirty-build:
	ARCH=$(ARCH)                     \
	    OS=$(OS)                     \
	    OUTPUT=$(OUTPUT)             \
	    VERSION=$(VERSION)           \
	    GOFLAGS=$(GOFLAGS)           \
	    DIRTY_BUILD=1                \
	    bash build/build.sh $(ENTRY)

docker-build: # @HELP build docker image
docker-build:
	docker build                         \
	    --build-arg "ARCH=$(ARCH)"       \
	    --build-arg "OS=$(OS)"           \
        --build-arg "VERSION=$(VERSION)" \
	    --build-arg "GOFLAGS=$(GOFLAGS)" \
	    --build-arg "GOPROXY=$(GOPROXY)" \
	    -t $(IMG) .

docker-push: # @HELP pushes the container to the defined registry
docker-push: docker-build
	docker push $(IMG)

clean: # @HELP clean build artifacts
clean:
	rm -rf bin

version: # @HELP outputs the version string
version:
	echo $(VERSION)

binary-name: # @HELP outputs the output binary name
binary-name:
	echo $(BIN_FULLNAME)

help: # @HELP prints this message
help:
	echo "VARIABLES:"
	echo "  OUTPUT           $(OUTPUT)"
	echo "  OS               $(OS)"
	echo "  ARCH             $(ARCH)"
	echo "  VERSION          $(VERSION)"
	echo "  REGISTRY         $(REGISTRY)"
	echo "  IMG              $(IMG)"
	echo "  GOFLAGS          $(GOFLAGS)"
	echo "  GOPROXY          $(GOPROXY)"
	echo
	echo "TARGETS:"
	grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST)     \
	    | awk '                                   \
	        BEGIN {FS = ": *# *@HELP"};           \
	        { printf "  %-15s %s\n", $$1, $$2 };  \
	    '
	echo
	echo "NOTES:"
	echo "  set \$$FULL_NAME to output binary with detailed name"
