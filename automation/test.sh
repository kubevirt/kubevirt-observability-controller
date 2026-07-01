#!/bin/bash
#
# This file is part of the KubeVirt project

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Copyright The KubeVirt Authors.

set -exo pipefail

CONTAINER_TOOL="${CONTAINER_TOOL:-docker}"

_base_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
_kubevirt_sh="${_base_dir}/hack/kubevirt.sh"

# Bring up the KubeVirt cluster
"${_kubevirt_sh}" up

# Export kubeconfig so kubectl and test clients can reach the cluster
export KUBECONFIG=$("${_kubevirt_sh}" kubeconfig)

# Build the controller image and push it to the local registry
REGISTRY=$("${_kubevirt_sh}" registry)
PUSH_IMG="${REGISTRY}/virt-observability-controller:latest"

${CONTAINER_TOOL} build -t "${PUSH_IMG}" "${_base_dir}"
${CONTAINER_TOOL} push --tls-verify=false "${PUSH_IMG}"

# Inside the cluster, the registry is reachable at registry:5000
CLUSTER_IMG="registry:5000/virt-observability-controller:latest"

# Run the e2e tests
make test-e2e IMG="${CLUSTER_IMG}"
