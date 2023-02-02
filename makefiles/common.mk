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

# Binary basename
BIN_BASENAME     := $(BIN)$(BIN_EXTENSION)
# Binary basename with version and target
BIN_VERBOSE_BASE := $(BIN)-$(VERSION)-$(OS)-$(ARCH)$(BIN_EXTENSION)
# If the user set FULL_NAME, we will use the basename with version and target.
# e.g. kube-trigger-v0.0.1-linux-amd64
BIN_FULLNAME     := $(if $(FULL_NAME),$(BIN_VERBOSE_BASE),$(BIN_BASENAME))
PKG_FULLNAME     := $(if $(FULL_NAME),$(BIN_VERBOSE_BASE),$(BIN_BASENAME)).tar.gz
ifeq ($(OS), windows)
    PKG_FULLNAME := $(subst .exe,,$(if $(FULL_NAME),$(BIN_VERBOSE_BASE),$(BIN_BASENAME))).zip
endif

BIN_VERBOSE_DIR  := bin/$(BIN)-$(VERSION)
# Full output relative path
OUTPUT           := $(if $(FULL_NAME),$(BIN_VERBOSE_DIR)/$(BIN_FULLNAME),bin/$(BIN_FULLNAME))


all: build

build-%:
	$(MAKE) -f $(firstword $(MAKEFILE_LIST)) \
	    build                                \
	    --no-print-directory                 \
	    GOOS=$(firstword $(subst _, ,$*))    \
	    GOARCH=$(lastword $(subst _, ,$*))   \
	    FULL_NAME=1

package-%:
	$(MAKE) -f $(firstword $(MAKEFILE_LIST)) \
	    package                              \
	    --no-print-directory                 \
	    GOOS=$(firstword $(subst _, ,$*))    \
	    GOARCH=$(lastword $(subst _, ,$*))   \
	    FULL_NAME=1

all-build: # @HELP build binaries for all platforms
all-build: $(addprefix build-, $(subst /,_, $(BIN_PLATFORMS)))

all-package: # @HELP build and package binaries for all platforms
all-package: $(addprefix package-, $(subst /,_, $(BIN_PLATFORMS)))
# overwrite previous checksums
	cd "$(BIN_VERBOSE_DIR)" && sha256sum *{.tar.gz,.zip} > "$(BIN)-$(VERSION)-checksums.txt"

# Build cache of the build container
BUILDCACHE ?= $$(pwd)/bin/buildcache

build: # @HELP build binary for current platform
build: gen-dockerignore
	mkdir -p bin "$(BUILDCACHE)/gocache" "$(BUILDCACHE)/gomodcache"
	docker run                                  \
	    -i                                      \
	    --rm                                    \
	    -u $$(id -u):$$(id -g)                  \
	    -v $$(pwd):/src                         \
	    -w /src                                 \
	    -v $(BUILDCACHE)/gocache:/gocache       \
	    -v $(BUILDCACHE)/gomodcache:/gomodcache \
	    --env GOCACHE="/gocache"                \
	    --env GOMODCACHE="/gomodcache"          \
	    --env ARCH="$(ARCH)"                    \
	    --env OS="$(OS)"                        \
	    --env VERSION="$(VERSION)"              \
	    --env DBG_BUILD="$(DBG_BUILD)"          \
	    --env OUTPUT="$(OUTPUT)"                \
	    --env GOFLAGS="$(GOFLAGS)"              \
	    --env GOPROXY="$(GOPROXY)"              \
	    --env HTTP_PROXY="$(HTTP_PROXY)"        \
	    --env HTTPS_PROXY="$(HTTPS_PROXY)"      \
	    $(BUILD_IMAGE)                          \
	    ./build/build.sh $(ENTRY)

package: # @HELP package binary using gzip or zip
package: build
	echo "# Compressing $(BIN_FULLNAME) to $(PKG_FULLNAME)"
	mkdir -p "$(BIN_VERBOSE_DIR)"
	cp LICENSE "$(BIN_VERBOSE_DIR)/LICENSE"
	cp "$(OUTPUT)" "$(BIN_VERBOSE_DIR)/$(BIN_BASENAME)"
	cd $(BIN_VERBOSE_DIR) &&              \
	    if [ "$(OS)" == "windows" ]; then \
	        zip "$(PKG_FULLNAME)" "$(BIN_BASENAME)" LICENSE;     \
	    else                                                     \
	        tar czf "$(PKG_FULLNAME)" "$(BIN_BASENAME)" LICENSE; \
	    fi;                                                      \
	    sha256sum "$(PKG_FULLNAME)" >> "$(BIN)-$(VERSION)-checksums.txt"; \
	    rm -f LICENSE "$(BIN_BASENAME)"

docker-build-%:
	$(MAKE) -f $(firstword $(MAKEFILE_LIST)) \
	    docker-build                         \
	    --no-print-directory                 \
	    GOOS=$(firstword $(subst _, ,$*))    \
	    GOARCH=$(lastword $(subst _, ,$*))

BUILDX_PLATFORMS := $(shell echo "$(IMG_PLATFORMS)" | sed 's/ /,/g')

all-docker-build-push: # @HELP build and push docker images for all platforms
all-docker-build-push: $(addprefix build-, $(subst /,_, $(IMG_PLATFORMS)))
	echo -e "# Building and pushing images for platforms $(IMG_PLATFORMS)"
	echo -e "# target: $(OS)/$(ARCH)\tversion: $(VERSION)\ttags: $(IMGTAGS)"
	TMPFILE=Dockerfile && \
	    sed 's/$${BIN}/$(BIN)/g' Dockerfile.in > $${TMPFILE} && \
	    docker buildx build --push             \
	    -f $${TMPFILE}                         \
	    --platform "$(BUILDX_PLATFORMS)"       \
	    --build-arg "VERSION=$(VERSION)"       \
	    --build-arg "BASE_IMAGE=$(BASE_IMAGE)" \
	    $(addprefix -t ,$(IMGTAGS)) .


docker-build: # @HELP build docker image for current platform
docker-build: build-$(OS)_$(ARCH)
	echo -e "# target: $(OS)/$(ARCH)\tversion: $(VERSION)\ttags: $(IMGTAGS)"
	TMPFILE=Dockerfile && \
	    sed 's/$${BIN}/$(BIN)/g' Dockerfile.in > $${TMPFILE} && \
	    DOCKER_BUILDKIT=1                      \
	    docker build                           \
	    -f $${TMPFILE}                         \
	    --build-arg "ARCH=$(ARCH)"             \
	    --build-arg "OS=$(OS)"                 \
	    --build-arg "VERSION=$(VERSION)"       \
	    --build-arg "BASE_IMAGE=$(BASE_IMAGE)" \
	    $(addprefix -t ,$(IMGTAGS)) .

docker-push-%:
	echo "# Pushing $(subst =,:,$(subst _,/,$*))"
	docker push $(subst =,:,$(subst _,/,$*))

docker-push: # @HELP push images
docker-push: $(addprefix docker-push-, $(subst :,=, $(subst /,_, $(IMGTAGS))))

gen-dockerignore:
	echo -e "*\n!$(BIN_VERBOSE_DIR)" > .dockerignore

version: # @HELP output the version string
version:
	echo $(VERSION)

imageversion: # @HELP output the docker image version
imageversion:
	echo $(IMG_VERSION)

binary-name: # @HELP output current artifact binary name
binary-name:
	echo $(BIN_FULLNAME)

variables: # @HELP print makefile variables
variables:
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

help: # @HELP print this message
help: variables
	echo
	echo "TARGETS:"
	grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST)    \
	    | sed --expression='s_.*.mk:__g'         \
	    | awk '                                  \
	        BEGIN {FS = ": *# *@HELP"};          \
	        { printf "  %-25s %s\n", $$1, $$2 }; \
	    '
	echo
	echo "NOTES:"
	echo "  set \$$FULL_NAME to include target string in binary name"
