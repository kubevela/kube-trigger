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
	echo -e "# Building and pushing images for platforms $(IMG_PLATFORMS)"
	echo -e "# target: $(OS)/$(ARCH)\tversion: $(VERSION)\ttags: $(IMGTAGS)"
	TMPFILE=$$(mktemp) && \
	    sed 's/$${BIN}/$(BIN)/g' Dockerfile.in > $${TMPFILE} && \
	    docker buildx build --push       \
	    -f $${TMPFILE}                   \
	    --platform "$(BUILDX_PLATFORMS)" \
	    --build-arg "VERSION=$(VERSION)" \
	    --build-arg "GOFLAGS=$(GOFLAGS)" \
	    --build-arg "GOPROXY=$(GOPROXY)" \
	    --build-arg "ENTRY=$(ENTRY)"     \
	    $(addprefix -t ,$(IMGTAGS)) .


docker-build: # @HELP build docker image
docker-build:
	echo -e "# target: $(OS)/$(ARCH)\tversion: $(VERSION)\ttags: $(IMGTAGS)"
	TMPFILE=$$(mktemp) && \
	    sed 's/$${BIN}/$(BIN)/g' Dockerfile.in > $${TMPFILE} && \
	    docker build                     \
	    -f $${TMPFILE}                   \
	    --build-arg "ARCH=$(ARCH)"       \
	    --build-arg "OS=$(OS)"           \
	    --build-arg "VERSION=$(VERSION)" \
	    --build-arg "GOFLAGS=$(GOFLAGS)" \
	    --build-arg "GOPROXY=$(GOPROXY)" \
	    --build-arg "ENTRY=$(ENTRY)"     \
	    $(addprefix -t ,$(IMGTAGS)) .

docker-push-%:
	echo "# Pushing $(subst =,:,$(subst _,/,$*))"
	docker push $(subst =,:,$(subst _,/,$*))

docker-push: # @HELP push images
docker-push: $(addprefix docker-push-, $(subst :,=, $(subst /,_, $(IMGTAGS))))

lint: # @HELP run linter
lint: generate
	bash build/lint.sh

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
	grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST)    \
	    | sed --expression='s_.*.mk:__g'         \
	    | awk '                                  \
	        BEGIN {FS = ": *# *@HELP"};          \
	        { printf "  %-25s %s\n", $$1, $$2 }; \
	    '
	echo
	echo "NOTES:"
	echo "  set \$$FULL_NAME to include target string in binary name"
