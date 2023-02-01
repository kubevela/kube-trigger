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

include makefiles/consts.mk

# CLI entry file
ENTRY         := cmd/kubetrigger/main.go

# Binary targets that we support.
# When doing all-build, these targets will be built.
BIN_PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
IMG_PLATFORMS := linux/amd64 linux/arm64

# Binary basename, without extension
BIN           := kube-trigger

# Docker image tag
IMGTAGS  ?= $(addsuffix /$(BIN):$(IMG_VERSION),$(REGISTRY))

include makefiles/common.mk

generate: # @HELP run go generate
generate:
	go generate ./...
