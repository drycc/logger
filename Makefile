SHELL = /bin/bash
GOFMT = gofmt -l -w -s
GOLINT = lint
GOVET = go vet
GO_FILES = $(wildcard *.go)
GO_PACKAGES = storage log weblog
GO_PACKAGES_REPO_PATH = $(addprefix $(REPO_PATH)/,$(GO_PACKAGES))

# the filepath to this repository, relative to $GOPATH/src
REPO_PATH = github.com/drycc/logger

PLATFORM ?= linux/amd64,linux/arm64

# The following variables describe the containerized development environment
# and other build options
DEV_ENV_IMAGE := ${DEV_REGISTRY}/drycc/go-dev
DEV_ENV_WORK_DIR := /opt/drycc/go/src/${REPO_PATH}
DEV_ENV_OPTS := --rm -v ${CURDIR}:${DEV_ENV_WORK_DIR} -w ${DEV_ENV_WORK_DIR}
DEV_ENV_CMD := podman run ${DEV_ENV_OPTS} ${DEV_ENV_IMAGE}
DEV_ENV_CMD_INT := podman run -it ${DEV_ENV_OPTS} ${DEV_ENV_IMAGE}
LDFLAGS := "-s -X main.version=${VERSION}"

BINARY_DEST_DIR = rootfs/opt/logger/sbin
BUILD_TAG ?= git-$(shell git rev-parse --short HEAD)
SHORT_NAME ?= logger
DRYCC_REGISTRY ?= ${DEV_REGISTRY}
IMAGE_PREFIX ?= drycc

include versioning.mk

SHELL_SCRIPTS = $(wildcard _scripts/util/*)

check-podman:
	@if [ -z $$(which podman) ]; then \
	  echo "Missing podman client which is required for development"; \
	  exit 2; \
	fi

# Allow developers to step into the containerized development environment
dev: check-podman
	${DEV_ENV_CMD_INT} bash

# Containerized dependency resolution
bootstrap: check-podman
	${DEV_ENV_CMD} go mod vendor

# This is so you can build the binary without using podman
build-binary:
	CGO_ENABLED=0 go build -ldflags ${LDFLAGS} -o $(BINARY_DEST_DIR)/logger .

build: podman-build
build-without-container: build-binary build-image
push: podman-push

# Containerized build of the binary
build-with-container: check-podman
	mkdir -p ${BINARY_DEST_DIR}
	${DEV_ENV_CMD} make build-binary

podman-build: check-podman
	podman build --build-arg CODENAME=${CODENAME} -t ${IMAGE} --build-arg LDFLAGS=${LDFLAGS} .
	podman tag ${IMAGE} ${MUTABLE_IMAGE}

clean: check-podman
	podman rmi $(IMAGE)

test: test-style test-unit

test-cover:
	_scripts/tests.sh test-cover "${DEV_ENV_OPTS}" "${DEV_ENV_IMAGE}"

test-style: check-podman
	${DEV_ENV_CMD} make style-check

style-check:
# display output, then check
	$(GOFMT) $(GO_PACKAGES) $(GO_FILES)
	@$(GOFMT) $(GO_PACKAGES) $(GO_FILES)
	$(GOVET) $(REPO_PATH) $(GO_PACKAGES_REPO_PATH)
	$(GOLINT)
	shellcheck $(SHELL_SCRIPTS)

test-unit:
	_scripts/tests.sh test-unit "${DEV_ENV_OPTS}" "${DEV_ENV_IMAGE}"
