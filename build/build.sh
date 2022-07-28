#!/bin/sh

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

set -o errexit
set -o nounset
# This will not work in POSIX sh
# set -o pipefail

if [ -z "${OS:-}" ]; then
  echo "OS must be set"
  exit 1
fi

if [ -z "${ARCH:-}" ]; then
  echo "ARCH must be set"
  exit 1
fi

if [ -z "${VERSION:-}" ]; then
  echo "VERSION must be set"
  exit 1
fi

if [ -z "${OUTPUT:-}" ]; then
  echo "OUTPUT must be set"
  exit 1
fi

export CGO_ENABLED=0
export GOARCH="${ARCH}"
export GOOS="${OS}"
export GO111MODULE=on
export GOFLAGS="${GOFLAGS:-} -mod=mod "

echo -e "# target: ${OS}/${ARCH}\tversion: ${VERSION}\toutput: ${OUTPUT}"

LDFLAGS_EXTRA="${LDFLAGS_EXTRA:-}"

if [ -z "${DIRTY_BUILD:-}" ]; then
  # If user don't want dirty build, remove all unnecessary info from binary.
  LDFLAGS_EXTRA="${LDFLAGS_EXTRA:-} -s -w"
  # No cache.
  export GOFLAGS="${GOFLAGS:-} -a "
  echo -n "# Clean "
else
  echo -n "# Dirty "
fi

# Set some version info.
GO_LDFLAGS="${LDFLAGS_EXTRA} -X $(go list -m)/pkg/version.Version=${VERSION}"

echo "building... "

go build                   \
  -ldflags "${GO_LDFLAGS}" \
  -o "${OUTPUT}"           \
  "$@"
