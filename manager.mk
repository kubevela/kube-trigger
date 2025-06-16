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
ENTRY           := cmd/manager/main.go
# All supported platforms for binary distribution
BIN_PLATFORMS   := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
# All supported platforms for container image distribution
IMAGE_PLATFORMS := linux/amd64 linux/arm64
# Binary basename (.exe will be automatically added when building for Windows)
BIN             := manager
# Container image name, without repo or tags
IMAGE_NAME      := kube-trigger-$(BIN)
# Container image repositories to push to (supports multiple repos)
IMAGE_REPOS     := docker.io/oamdev ghcr.io/kubevela

# Setup make variables
include makefiles/consts.mk

# Specific targets to this subproject
manifests: # @HELP Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects
manifests: controller-gen
	$(CONTROLLER_GEN) rbac:roleName=kube-trigger-manager-role crd webhook paths="{./api/...,./cmd/...,./controllers/...,./pkg/...}" output:crd:artifacts:config=config/crd output:rbac:dir=config/manager

generate: # @HELP Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate/boilerplate.go.txt" paths="{./api/...,./cmd/...,./controllers/...,./pkg/...}"
	go generate ./...

install: # @HELP Install CRDs into the K8s cluster specified in ~/.kube/config
install: manifests
	kubectl apply -f config/crd

uninstall: # @HELP Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion
uninstall: manifests
	kubectl delete --ignore-not-found -f config/crd

deploy: # @HELP Deploy controller to the K8s cluster specified in ~/.kube/config
deploy: manifests
	kubectl apply -f config/manager

undeploy: # @HELP Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion
undeploy:
	kubectl delete --ignore-not-found -f config/manager

# Location to install dependencies to
bin:
	mkdir -p bin

# Tool Binaries
CONTROLLER_GEN ?= bin/controller-gen

# Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.16.4

controller-gen: bin
	[ -f $(CONTROLLER_GEN) ] || GOBIN=$(PWD)/bin go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

# Setup common targets
include makefiles/targets.mk
