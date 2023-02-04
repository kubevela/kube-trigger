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
    # If we're not debugging the Makefile, don't print directories or recipies.
    MAKEFLAGS += -s --no-print-directory
endif

# No, we don't want builtin rules.
MAKEFLAGS += --no-builtin-rules
# Get rid of .PHONY everywhere.
MAKEFLAGS += --always-make

# Use bash explicitly
SHELL := /usr/bin/env bash -o errexit -o pipefail -o nounset

# All subprojects, e.g. trigger and manager
BIN = $(patsubst %.mk,%,$(wildcard *.mk))

# ===== Misc Targets ======

generate: $(addprefix mk-generate_,$(BIN))

lint:
	build/lint.sh

checklicense:
	hack/verify-boilerplate.sh

svgformat:
	hack/format-svg-image.sh

clean:
	rm -rf bin

reviewable: generate checklicense lint

checkdiff: generate
	git --no-pager diff
	if ! git diff --quiet; then                                     \
	    echo "Please run 'make reviewable' to include all changes"; \
	    false;                                                      \
	fi

# ===== Specific Targets ======

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.24.1
ENVTEST            ?= bin/setup-envtest
# Location to install dependencies to
bin:
	mkdir -p bin

envtest: bin
	[ -f $(ENVTEST) ] || GOBIN=$(PWD)/bin go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

test: envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
	    go test -coverprofile=cover.out ./...

# ===== Common Targets for both kubetrigger and manager ======

# Run `make TARGET' to run TARGET for both kubetrigger and manager
# Run `make TARGET-SUBPROJ' to run TARGET for SUBPROJ

all: # @HELP same as build
all: $(addprefix mk-all_,$(BIN))
all-%: mk-all_%;

build: # @HELP build binary for current platform
build: $(addprefix mk-build_,$(BIN))
build-%: mk-build_%;

all-build: # @HELP build binaries for all platforms
all-build: $(addprefix mk-all-build_,$(BIN))
all-build-%: mk-all-build_%;

package: # @HELP build and package binary for current platform
package: $(addprefix mk-package_,$(BIN))
package-%: mk-package_%;

all-package: # @HELP build and package binaries for all platforms with checksum
all-package: $(addprefix mk-all-package_,$(BIN))
all-package-%: mk-all-package_%;

all-docker-build-push: # @HELP build and push docker images for all platforms to all registries
all-docker-build-push: $(addprefix mk-all-docker-build-push_,$(BIN))
all-docker-build-push-%: mk-all-docker-build-push_%;

docker-build: # @HELP build docker image for current platform
docker-build: $(addprefix mk-docker-build_,$(BIN))
docker-build-%: mk-docker-build_%;

docker-push: # @HELP push alredy built images to all registries
docker-push: $(addprefix mk-docker-push_,$(BIN))
docker-push-%: mk-docker-push_%;

version: # @HELP output the version string
version: $(addprefix mk-version_,$(BIN))
version-%: mk-version_%;

imageversion: # @HELP output the docker image version
imageversion: $(addprefix mk-imageversion_,$(BIN))
imageversion-%: mk-imageversion_%;

binary-name: # @HELP output current artifact binary name
binary-name: $(addprefix mk-binary-name_,$(BIN))
binary-name-%: mk-binary-name_%;

variables: # @HELP print makefile variables
variables: $(addprefix mk-variables_,$(BIN))
variables-%: mk-variables_%;

help: # @HELP print this message
help: $(addprefix mk-help_,$(BIN))
help-%: mk-help_%;

mk-%:
	$(MAKE) -f $(lastword $(subst _, ,$*)).mk $(firstword $(subst _, ,$*))

