#!/bin/bash

# Copyright 2019 The Kubeflow Authors.
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

# This shell is used to auto generate some useful tools for k8s, such as lister,
# informer, deepcopy, defaulter and so on.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
echo ">> Script root ${SCRIPT_ROOT}"
ROOT_PKG=github.com/jazzsir/common

# Grab code-generator version from go.mod
CODEGEN_VERSION=$(grep 'k8s.io/code-generator' go.mod | awk '{print $2}' | head -1)
CODEGEN_PKG=$(echo `go env GOPATH`"/pkg/mod/k8s.io/code-generator@${CODEGEN_VERSION}")

# Grab kube-openapi version from go.mod
OPENAPI_VERSION=$(grep 'k8s.io/kube-openapi' go.mod | awk '{print $2}' | head -1)
# remove /go.mod if it happens to match the version
if [[ $OPENAPI_VERSION == */go.mod ]]; then
    OPENAPI_VERSION=${OPENAPI_VERSION%/*}
fi

OPENAPI_PKG=$(echo `go env GOPATH`"/pkg/mod/k8s.io/kube-openapi@${OPENAPI_VERSION}")

if [[ ! -d ${CODEGEN_PKG} || ! -d ${OPENAPI_PKG} ]]; then
    echo "${CODEGEN_PKG} or ${OPENAPI_PKG} is missing. Running 'go mod download'."
    go mod download
fi

echo ">> Using ${CODEGEN_PKG}"
echo ">> Using ${OPENAPI_PKG}"
# Ensure we can execute shell scripts.
chmod +x ${CODEGEN_PKG}/generate-groups.sh

# code-generator does work with go.mod but makes assumptions about
# the project living in `$GOPATH/src`. To work around this and support
# any location; create a temporary directory, use this as an output
# base, and copy everything back once generated.
TEMP_DIR=$(mktemp -d)
cleanup() {
    echo ">> Removing ${TEMP_DIR}"
    rm -rf ${TEMP_DIR}
}
trap "cleanup" EXIT SIGINT

echo ">> Temporary output directory ${TEMP_DIR}"

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
cd ${SCRIPT_ROOT}
${CODEGEN_PKG}/generate-groups.sh "deepcopy" \
 github.com/jazzsir/common/pkg/client github.com/jazzsir/common/pkg/apis \
 common:v1 \
 --output-base "${TEMP_DIR}" \
 --go-header-file hack/boilerplate/boilerplate.go.txt

${CODEGEN_PKG}/generate-groups.sh "all" \
 github.com/jazzsir/common/test_job/client github.com/jazzsir/common/test_job/apis \
 test_job:v1 \
 --output-base "${TEMP_DIR}" \
 --go-header-file hack/boilerplate/boilerplate.go.txt

# Notice: The code in code-generator does not generate defaulter by default.
# We need to build binary from vendor cmd folder.
#echo "Building defaulter-gen"
#go get k8s.io/code-generator/cmd/defaulter-gen@v0.19.9
#go build -o ${GOPATH}/bin/defaulter-gen ${CODEGEN_PKG}/cmd/defaulter-gen

echo "Generating defaulters for common/v1"
${GOPATH}/bin/defaulter-gen --input-dirs github.com/jazzsir/common/pkg/apis/common/v1 \
-O zz_generated.defaults \
--output-package github.com/jazzsir/common/pkg/apis/common/v1 \
--go-header-file hack/boilerplate/boilerplate.go.txt "$@" \
--output-base "${TEMP_DIR}" 

echo "Generating defaulters for test_job/v1"
${GOPATH}/bin/defaulter-gen --input-dirs github.com/jazzsir/common/test_job/apis/test_job/v1 \
-O zz_generated.defaults \
--output-package github.com/jazzsir/common/test_job/apis/test_job/v1 \
--go-header-file hack/boilerplate/boilerplate.go.txt "$@" \
--output-base "${TEMP_DIR}" 

echo "Building openapi-gen"
GOFLAGS=-mod=mod go build -o ${GOPATH}/bin/openapi-gen ${OPENAPI_PKG}/cmd/openapi-gen

echo "Generating OpenAPI specification for common/v1"
${GOPATH}/bin/openapi-gen --input-dirs github.com/jazzsir/common/pkg/apis/common/v1 \
--output-package github.com/jazzsir/common/pkg/apis/common/v1 \
--go-header-file hack/boilerplate/boilerplate.go.txt "$@" \
--output-base "${TEMP_DIR}" 

echo "Generating OpenAPI specification for test_job/v1"
${GOPATH}/bin/openapi-gen --input-dirs github.com/jazzsir/common/test_job/apis/test_job/v1 \
--output-package github.com/jazzsir/common/test_job/apis/test_job/v1 \
--go-header-file hack/boilerplate/boilerplate.go.txt "$@" \
--output-base "${TEMP_DIR}" 

## Copy everything back.
cp -a "${TEMP_DIR}/${ROOT_PKG}/." "${SCRIPT_ROOT}/"
