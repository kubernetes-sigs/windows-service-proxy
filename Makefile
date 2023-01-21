# Copyright 2023 The Kubernetes Authors.
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

SHELL:=/usr/bin/env bash

GOOS ?= windows
GO_VERSION ?= 1.19.4

BIN_DIR := bin
TEST_DIR := test
TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(abspath $(TOOLS_DIR)/$(BIN_DIR))

GOLANGCI_LINT_BIN := golangci-lint
GOLANGCI_LINT := $(abspath $(TOOLS_BIN_DIR)/$(GOLANGCI_LINT_BIN))

VERSION ?= $(shell git describe --tags --always `git rev-parse HEAD`)
STAGING_REGISTRY ?= gcr.io/k8s-staging-win-svc-proxy
REGISRY ?= 

all: build

help:  # Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[0-9A-Za-z_-]+:.*?##/ { printf "  \033[36m%-45s\033[0m %s\n", $$1, $$2 } /^\$$\([0-9A-Za-z_-]+\):.*?##/ { gsub("_","-", $$1); printf "  \033[36m%-45s\033[0m %s\n", tolower(substr($$1, 3, length($$1)-7)), $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## --------------------------------------
## Binaries
## --------------------------------------

##@ build:

.PHONY: build
build:  ## Build Windows kernelspace as kpng store
	GOOS="$(GOOS)" go build -o output/kpng.exe ./cmd/...


## --------------------------------------
## Lint / Verify
## --------------------------------------

##@ lint and verify:

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Lint the codebase
	$(GOLANGCI_LINT) run -v $(GOLANGCI_LINT_EXTRA_ARGS)

.PHONY: lint-dockerfiles
lint-dockerfiles:
	./scripts/ci-lint-dockerfiles.sh $(HADOLINT_VER) $(HADOLINT_FAILURE_THRESHOLD)

.PHONY: lint-fix
lint-fix: $(GOLANGCI_LINT) ## Lint the codebase and run auto-fixers if supported by the linter
	GOLANGCI_LINT_EXTRA_ARGS=--fix $(MAKE) lint

## --------------------------------------
## Hack / Tools
## --------------------------------------

##@ hack/tools:

$(GOLANGCI_LINT):
	hack/ensure-golangci-lint.sh \
		-b $(TOOLS_BIN_DIR) \
		$(shell cat .github/workflows/golangci-lint.yml | grep [[:space:]]version | sed 's/.*version: //')

# Contianer image building targets

.PHONY: image
image: 
	REPOSITORY=$(REGISTRY) ./hack/build_windows_container.sh

.PHONY: image-push
image-push:
	REPOSITORY=$(REGISTRY) VERSION=$(VERSION) ./hack/build_windows_container.sh --push

.PHONY: release-staging
release-staging:
	REPOSITORY=$(STAGING_REGISTRY) VERSION=$(VERSION) ./hack/build_windows_container.sh --push
