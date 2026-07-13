#!/bin/bash
#
# This file is part of the KubeVirt project
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
#
# Copyright the KubeVirt Authors.
#
#
set -exuo pipefail

source automation/github-source-tarball-signature.sh

function cleanup_gh_install() {
    [ -n "${gh_cli_dir}" ] && [ -d "${gh_cli_dir}" ] && rm -rf "${gh_cli_dir:?}/"
}

function ensure_gh_cli_installed() {
    if command -V gh; then
        return
    fi

    trap 'cleanup_gh_install' EXIT SIGINT SIGTERM

    # install gh cli for uploading release artifacts, with prompt disabled to enforce non-interactive mode
    gh_cli_dir=$(mktemp -d)
    (
        cd  "$gh_cli_dir/"
        curl -sSL "https://github.com/cli/cli/releases/download/v${GH_CLI_VERSION}/gh_${GH_CLI_VERSION}_linux_amd64.tar.gz" -o "gh_${GH_CLI_VERSION}_linux_amd64.tar.gz"
        tar xvf "gh_${GH_CLI_VERSION}_linux_amd64.tar.gz"
    )
    export PATH="$gh_cli_dir/gh_${GH_CLI_VERSION}_linux_amd64/bin:$PATH"
    if ! command -V gh; then
        echo "gh cli not installed successfully"
        exit 1
    fi
    gh config set prompt disabled
}

function build_release_artifacts() {
    make build-installer IMG="${DOCKER_PREFIX}/virt-observability-controller:${DOCKER_TAG}"
}

function update_github_release() {
    if ! gh release view --repo "$GITHUB_REPOSITORY" "$DOCKER_TAG" &>/dev/null; then
        gh release create --repo "$GITHUB_REPOSITORY" "$DOCKER_TAG" --title "$DOCKER_TAG" --generate-notes
    fi
    gh release upload --repo "$GITHUB_REPOSITORY" --clobber "$DOCKER_TAG" dist/install.yaml
}

function setup_github_release() {
    GIT_ASKPASS="$(pwd)/automation/git-askpass.sh"
    [ -f "$GIT_ASKPASS" ] || exit 1
    export GIT_ASKPASS

    ensure_gh_cli_installed

    gh auth login --with-token <"$GITHUB_TOKEN_PATH"
}

function create_github_release() {
    build_release_artifacts
    update_github_release
    # update_github_source_tarball_signature
}

function publish_image() {
    local img="${DOCKER_PREFIX}/virt-observability-controller"
    make docker-buildx IMG="${img}:${DOCKER_TAG}"

    if [ "${DOCKER_TAG}" = "latest" ]; then
        make docker-retag IMG="${img}:${DOCKER_TAG}" NEW_IMG="${img}:$(date +%s)"
    elif [[ "${DOCKER_TAG}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        make docker-retag IMG="${img}:${DOCKER_TAG}" NEW_IMG="${img}:stable"
    fi
}

function main() {
    if [ -z "${DOCKER_PREFIX:-}" ]; then
        echo "DOCKER_PREFIX is not set"
        exit 1
    fi
    if [ -z "${DOCKER_TAG:-}" ]; then
        echo "DOCKER_TAG is not set"
        exit 1
    fi

    if [ "${GITHUB_RELEASE:-false}" = "true" ]; then
        setup_github_release
    fi

    publish_image

    if [ "${GITHUB_RELEASE:-false}" = "true" ]; then
        create_github_release
    fi
}

main "$@"
