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

# Set this to 1 to enable debugging output.
DBG_MAKEFILE ?=
ifeq ($(DBG_MAKEFILE),1)
    $(warning ***** starting Makefile for goal(s) "$(MAKECMDGOALS)")
    $(warning ***** $(shell date))
else
    # If we're not debugging the Makefile, don't echo recipes.
    MAKEFLAGS += -s
endif

# No, we don't want builtin rules.
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --warn-undefined-variables
# Get rid of .PHONY everywhere.
MAKEFLAGS += --always-make

# Binary targets that we support.
# When doing all-build, these targets will be built.
BIN_PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
IMG_PLATFORMS := linux/amd64 linux/arm64

# If user has not defined target, set some default value, same as host machine.
OS          := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
ARCH        := $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))
# Use git tags to set the version string
VERSION     ?= $(shell git describe --tags --always --dirty)
IMG_VERSION ?= $(shell bash -c "\
if [[ ! $(VERSION) =~ ^v[0-9]{1,2}\.[0-9]{1,2}\.[0-9]{1,2}(-(alpha|beta)\.[0-9]{1,2})?$$ ]]; then \
  echo latest;                                                                                    \
else                                                                                              \
  echo $(VERSION);                                                                                \
fi")

BIN_EXTENSION :=
ifeq ($(OS), windows)
    BIN_EXTENSION := .exe
endif

DIRTY_BUILD ?=
FULL_NAME   ?=
GOFLAGS     ?=
GOPROXY     ?=

# Binary basename, without extension
BIN          := kube-trigger
# Binary basename
_OUT         := $(BIN)$(BIN_EXTENSION)
# Binary basename with version and target
_VER_OUT     := $(BIN)-$(VERSION)-$(OS)-$(ARCH)$(BIN_EXTENSION)
# If the user set FULL_NAME, we will use the basename with version and target.
# e.g. kube-trigger-v0.0.1-linux-amd64
BIN_FULLNAME := $(if $(FULL_NAME),$(_VER_OUT),$(_OUT))
PKG_FULLNAME := $(if $(FULL_NAME),$(_VER_OUT),$(_OUT)).tar.gz
ifeq ($(OS), windows)
    PKG_FULLNAME := $(subst .exe,,$(if $(FULL_NAME),$(_VER_OUT),$(_OUT))).zip
endif
# Full output relative path
OUTPUT       := bin/$(BIN_FULLNAME)
# CLI entry file
ENTRY        := cmd/kubetrigger/main.go

# Registry to push to
REGISTRY := docker.io/oamdev ghcr.io/kubevela
# Docker image tag
IMGTAGS  ?= $(addsuffix /$(BIN):$(IMG_VERSION),$(REGISTRY))

# Use bash explicitly
SHELL := /usr/bin/env bash -o errexit -o pipefail -o nounset

all: build

build-%:
	$(MAKE) package                        \
	    --no-print-directory               \
	    GOOS=$(firstword $(subst _, ,$*))  \
	    GOARCH=$(lastword $(subst _, ,$*)) \
	    FULL_NAME=1

all-build: # @HELP build and package binaries for all platforms
all-build: $(addprefix build-, $(subst /,_, $(BIN_PLATFORMS)))
	cd bin && sha256sum *{.tar.gz,.zip} > "$(BIN)-$(VERSION)-checksums.txt"

build: # @HELP build binary locally
build:
	ARCH=$(ARCH)                     \
	    OS=$(OS)                     \
	    OUTPUT=$(OUTPUT)             \
	    VERSION=$(VERSION)           \
	    GOFLAGS=$(GOFLAGS)           \
	    DIRTY_BUILD=$(DIRTY_BUILD)   \
	    bash build/build.sh $(ENTRY)

package: build
	echo "# Compressing $(BIN_FULLNAME) to $(PKG_FULLNAME)"
	cp LICENSE bin/LICENSE
	cd bin && if [ "$(OS)" == "windows" ]; then              \
	    zip "$(PKG_FULLNAME)" "$(BIN_FULLNAME)" LICENSE;     \
	else                                                     \
	    tar czf "$(PKG_FULLNAME)" "$(BIN_FULLNAME)" LICENSE; \
	fi

dirty-build: # @HELP same as build, but using build cache is allowed
dirty-build:
	$(MAKE) package DIRTY_BUILD=1

docker-build-%:
	$(MAKE) docker-build                   \
	    --no-print-directory               \
	    GOOS=$(firstword $(subst _, ,$*))  \
	    GOARCH=$(lastword $(subst _, ,$*))

BUILDX_PLATFORMS := $(shell echo "$(IMG_PLATFORMS)" | sed -r 's/ /,/g')

all-docker-build-push: # @HELP build and push images for all platforms
all-docker-build-push:
	echo -e "# Building and pushing images for $(IMG_PLATFORMS)"
	docker buildx build --push           \
	    --platform "$(BUILDX_PLATFORMS)" \
	    --build-arg "VERSION=$(VERSION)" \
	    --build-arg "GOFLAGS=$(GOFLAGS)" \
	    --build-arg "GOPROXY=$(GOPROXY)" \
	    $(addprefix -t ,$(IMGTAGS)) .


docker-build: # @HELP build docker image
docker-build:
	echo -e "# target: $(OS)/$(ARCH)\tversion: $(VERSION)"
	docker build                         \
	    --build-arg "ARCH=$(ARCH)"       \
	    --build-arg "OS=$(OS)"           \
	    --build-arg "VERSION=$(VERSION)" \
	    --build-arg "GOFLAGS=$(GOFLAGS)" \
	    --build-arg "GOPROXY=$(GOPROXY)" \
	    $(addprefix -t ,$(IMGTAGS)) .

docker-push-%:
	echo "# Pushing $(subst =,:,$(subst _,/,$*))"
	docker push $(subst =,:,$(subst _,/,$*))

docker-push: # @HELP push images
docker-push: $(addprefix docker-push-, $(subst :,=, $(subst /,_, $(IMGTAGS))))

lint: # @HELP run linter
lint: generate
	bash build/lint.sh

generate: # @HELP run go generate
generate:
	go generate ./...

checklicense: # @HELP check license headers
checklicense:
	bash hack/verify-boilerplate.sh

reviewable: # @HELP do some checks before submitting code
reviewable: generate checklicense lint

clean: # @HELP remove build artifacts
clean:
	rm -rf bin

version: # @HELP output the version string
version:
	echo $(VERSION)

imageversion: # @HELP output the image version
imageversion:
	echo $(IMG_VERSION)

binary-name: # @HELP output the binary name
binary-name:
	echo $(BIN_FULLNAME)

variables: # @HELP print makefile variables
variables:
	echo "  OUTPUT            $(OUTPUT)"
	echo "  OS                $(OS)"
	echo "  ARCH              $(ARCH)"
	echo "  VERSION           $(VERSION)"
	echo "  IMG_VERSION       $(IMG_VERSION)"
	echo "  REGISTRY          $(REGISTRY)"
	echo "  IMG_TAGS          $(IMGTAGS)"
	echo "  BIN_PLATFORMS     $(BIN_PLATFORMS)"
	echo "  IMG_PLATFORMS     $(IMG_PLATFORMS)"
	echo "  GOPROXY           $(GOPROXY)"
	echo "  GOFLAGS           $(GOFLAGS)"

help: # @HELP print this message
help:
	echo "VARIABLES:"
	echo "  OUTPUT            $(OUTPUT)"
	echo "  OS                $(OS)"
	echo "  ARCH              $(ARCH)"
	echo "  VERSION           $(VERSION)"
	echo "  IMG_VERSION       $(IMG_VERSION)"
	echo "  REGISTRY          $(REGISTRY)"
	echo "  IMG_TAGS          $(IMGTAGS)"
	echo "  BIN_PLATFORMS     $(BIN_PLATFORMS)"
	echo "  IMG_PLATFORMS     $(IMG_PLATFORMS)"
	echo "  GOPROXY           $(GOPROXY)"
	echo "  GOFLAGS           $(GOFLAGS)"
	echo
	echo "TARGETS:"
	grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST)     \
	    | awk '                                   \
	        BEGIN {FS = ": *# *@HELP"};           \
	        { printf "  %-25s %s\n", $$1, $$2 };  \
	    '
	echo
	echo "NOTES:"
	echo "  set \$$FULL_NAME to include target string in binary name"
