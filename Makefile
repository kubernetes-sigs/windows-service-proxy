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

BIN_DIR := bin
TEST_DIR := test
TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(abspath $(TOOLS_DIR)/$(BIN_DIR))

GOLANGCI_LINT_BIN := golangci-lint
GOLANGCI_LINT := $(abspath $(TOOLS_BIN_DIR)/$(GOLANGCI_LINT_BIN))

COMMIT ?= $(shell git describe --tags --always `git rev-parse HEAD`)
VERSION ?= $(COMMIT)
STAGING_REGISTRY ?= gcr.io/k8s-staging-win-svc-proxy
REGISTRY ?= docker.io/jsturtevant

help:  # Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[0-9A-Za-z_-]+:.*?##/ { printf "  \033[36m%-45s\033[0m %s\n", $$1, $$2 } /^\$$\([0-9A-Za-z_-]+\):.*?##/ { gsub("_","-", $$1); printf "  \033[36m%-45s\033[0m %s\n", tolower(substr($$1, 3, length($$1)-7)), $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## --------------------------------------
## Lint / Verify
## --------------------------------------

##@ lint and verify:

.PHONY: lint-helm
lint-helm:
	helm lint charts

.PHONY: lint
lint: lint-helm 

# Contianer image building targets

.PHONY: image
image: 
	REPOSITORY=$(REGISTRY) VERSION=$(VERSION) COMMIT=$(COMMIT) ./hack/build_windows_container.sh

.PHONY: image-push
image-push:
	REPOSITORY=$(REGISTRY) VERSION=$(VERSION) COMMIT=$(COMMIT) ./hack/build_windows_container.sh --push

.PHONY: release-staging
release-staging:
	REPOSITORY=$(STAGING_REGISTRY) VERSION=$(VERSION) COMMIT=$(COMMIT) ./hack/build_windows_container.sh --push
