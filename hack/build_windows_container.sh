#!/bin/bash
set -e

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

export DOCKER_CLI_EXPERIMENTAL=enabled

args=$(getopt -o p -l push -- "$@")
eval set -- "$args"

while [ $# -ge 1 ]; do
  case "$1" in
    --)
      shift
      break
      ;;
    -p|--push)
    push="1"
    shift
    ;;
  esac
  shift
done

output="type=docker,dest=./output/export.tar"

if [[ "$push" == "1" ]]; then
  output="type=registry"
else
  # ensure output directory exists
  mkdir -p ./output
fi

: "${REPOSITORY?"Need to set REPOSITORY"}"
VERSION=${VERSION:-"latest"}

set -x

docker buildx create --name img-builder --use --platform windows/amd64
trap 'docker buildx rm img-builder' EXIT

tags=" -t ${REPOSITORY}/kube-proxy:${VERSION}"
if [[ "$VERSION" != "latest" ]]; then
  tags+=" -t ${REPOSITORY}/kube-proxy:latest"
fi

docker buildx build --platform windows/amd64 --output=$output --build-arg=KUBEPROXY_VERSION=$VERSION -f Dockerfile $tags . 
docker buildx build --platform windows/amd64 --output=$output -f Dockerfile.sourcevip -t ${REPOSITORY}/kube-proxy:sourcevip-$VERSION .
