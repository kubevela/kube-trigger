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

ARG BUILD_IMAGE=golang:1.17
ARG BASE_IMAGE=gcr.io/distroless/static:nonroot
ARG ARCH
ARG OS

FROM --platform=${OS}/${ARCH} ${BUILD_IMAGE} as builder

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum

ARG GOPROXY
ENV GOPROXY=${GOPROXY}
RUN go mod download

ENV ARCH=${ARCH:-amd64}
ENV OS=${OS:-linux}
ARG VERSION
ENV VERSION=${VERSION}
ARG GOFLAGS
ENV GOFLAGS=${GOFLAGS}

COPY build/ build/
COPY cmd/ cmd/
COPY pkg/ pkg/

RUN ARCH=${ARCH}                \
        OS=${OS}                \
        OUTPUT=kube-trigger     \
        VERSION=${VERSION}      \
        GOFLAGS=${GOFLAGS}      \
        /bin/sh build/build.sh  \
        cmd/kubetrigger/main.go

FROM ${BASE_IMAGE}
WORKDIR /
COPY --from=builder /workspace/kube-trigger .
USER 65532:65532

ENTRYPOINT ["/kube-trigger"]
