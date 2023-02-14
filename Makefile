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

# ===== Common Targets for subprojects (trigger and manager) ======

SUBPROJS := $(patsubst %.mk,%,$(wildcard *.mk))

# Run `make TARGET' to run TARGET for both kube-trigger and manager.
#   For example, `make build' will build both kube-trigger and manager binaries.

# Run `make SUBPROJ-TARGET' to run TARGET for SUBPROJ.
#   For example, `make trigger-build' will only build kube-trigger binary.

# Run `make help' to see all available targets for subprojects. Similarly,
# `make trigger-help' will show help for kube-trigger.

# Targets to run on a specific subproject (<subproj>-<target>)
$(foreach p,$(SUBPROJS),$(eval \
    $(p)-%: mk-%.$(p);         \
))

# Common targets for subprojects, will be executed on all subprojects
TARGETS := build             \
    all-build                \
    package                  \
    all-package              \
    container-build          \
    container-push           \
    all-container-build-push \
    clean                    \
    all-clean                \
    version                  \
    imageversion             \
    binaryname               \
    variables                \
    help

# Run common targets on all subprojects
$(foreach t,$(TARGETS),$(eval                \
    $(t): $(addprefix mk-$(t).,$(SUBPROJS)); \
))

# `shell' only needs to be executed once, not on every subproject
shell: $(addprefix mk-shell.,$(word 1,$(SUBPROJS)));

mk-%:
	$(MAKE) -f $(lastword $(subst ., ,$*)).mk $(firstword $(subst ., ,$*))

# ===== Misc Targets ======

# Go packages to lint or test
GOCODEDIR := ./api/... ./cmd/... ./controllers/... ./pkg/...

# Call `make generate' on all subprojects
generate: $(addprefix mk-generate.,$(SUBPROJS))

# Lint code
lint:
	build/lint.sh $(GOCODEDIR)

# Check file header
checklicense:
	hack/verify-boilerplate.sh

# Format svg images
svgformat:
	hack/format-svg-image.sh

# Check possible issues before committing code
reviewable: generate checklicense lint

# Run tests
test: envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
	    go test -coverprofile=cover.out $(GOCODEDIR)

checkdiff: generate
	git --no-pager diff
	if ! git diff --quiet; then                                     \
	    echo "Please run 'make reviewable' to include all changes"; \
	    false;                                                      \
	fi

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION := 1.24.1
ENVTEST             ?= bin/setup-envtest

envtest:
	mkdir -p bin
	[ -f $(ENVTEST) ] || GOBIN=$(PWD)/bin \
	    go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
