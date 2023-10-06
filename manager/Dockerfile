# Copyright Contributors to the Open Cluster Management project

# Stage 1: build the target binaries
FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:v1.19.10-202307171905.el8.gba7a391 AS builder

WORKDIR /workspace

COPY go.mod go.sum ./
COPY ./manager/ ./manager/
COPY ./pkg/ ./pkg/

RUN CGO_ENABLED=1 GOEXPERIMENT=strictfipsruntime GOFLAGS="-p=4" go build -a -v -i -o bin/manager ./manager/cmd/manager/main.go

# Stage 2: Copy the binaries from the image builder to the base image
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
ENV USER_UID=1001
ENV USER_NAME=manager

# install operator binary
COPY --from=builder /workspace/bin/manager /usr/local/bin/manager

COPY ./manager/scripts/user_setup /usr/local/scripts/user_setup
RUN  /usr/local/scripts/user_setup


USER ${USER_UID}
ENTRYPOINT ["/usr/local/bin/manager"]
