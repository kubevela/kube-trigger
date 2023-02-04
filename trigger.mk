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

# Setup make
include makefiles/common.mk

# Settings for this subproject
# Entry file, containing func main
ENTRY           := cmd/kubetrigger/main.go
# All supported platforms for binary distribution
BIN_PLATFORMS   := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
# All supported platforms for container image distribution
IMAGE_PLATFORMS := linux/amd64 linux/arm64
# Binary basename (.exe will be automatically added when building for Windows)
BIN             := kube-trigger
# Container image name, without repo or tags
IMAGE_NAME      := $(BIN)
# Container image repositories to push to (supports multiple repos)
IMAGE_REPOS     := docker.io/oamdev ghcr.io/kubevela

# Setup make variables
include makefiles/consts.mk

# Specific targets to this subproject
generate: # @HELP run go generate
generate:
	go generate ./...

# Setup common targets
include makefiles/targets.mk
