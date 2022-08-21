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
# Get rid of .PHONY everywhere.
MAKEFLAGS += --always-make

# Use bash explicitly
SHELL := /usr/bin/env bash -o errexit -o pipefail -o nounset

reviewable: generate checklicense lint svgformat

generate:
	./make-kt generate
	./make-mgr manifests generate

lint:
	build/lint.sh

checklicense:
	hack/verify-boilerplate.sh

svgformat:
	hack/format-svg-image.sh

clean:
	rm -rf bin
