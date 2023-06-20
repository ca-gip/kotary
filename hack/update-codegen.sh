#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
bash "${CODEGEN_PKG}"/generate-groups.sh all \
  github.com/ca-gip/kotary/pkg/generated github.com/ca-gip/kotary/pkg/apis \
  cagip:v1  --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate.go.txt

# # Dirty fix fake clientset
# # GroupName needs to be updated manually to ca-gip.cagip.com
# # Issue open https://github.com/kubernetes/code-generator/issues/98
# BASE_PATH=$(pwd "${SCRIPT_ROOT}")
# FILE_TO_FIX="${BASE_PATH}/pkg/generated/clientset/versioned/typed/ca-gip/v1/fake/fake_resourcequotaclaim.go"
# SED_EXP='s/cagip.github.com/ca-gip.github.com/g'

# echo "Fixing group name in fake clienset"

# if [ "${OSTYPE//[0-9.]/}" == "darwin" ]; then sed -i '' ${SED_EXP} ${FILE_TO_FIX}
# else sed -i ${SED_EXP} ${FILE_TO_FIX}
# fi
