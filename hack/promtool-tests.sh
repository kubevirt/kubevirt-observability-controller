#!/usr/bin/env bash
# Runs localized promtool fixtures against the controller's generated rules.
set -euo pipefail

BASE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
PROMTOOL="${PROMTOOL:-${BASE_DIR}/bin/promtool}"
TEST_DIR="${BASE_DIR}/pkg/monitoring/rules"

if [[ ! -x "${PROMTOOL}" ]]; then
    echo "ERROR: promtool not found at ${PROMTOOL}. Run 'make promtool'." >&2
    exit 1
fi

# Fixtures expect the default runbook URLs, independent of the caller's env.
unset RUNBOOK_URL_TEMPLATE
STAGE=$(mktemp -d "${TMPDIR:-/tmp}/promtool-tests.XXXXXX")
trap 'rm -rf "${STAGE}"' EXIT

(cd "${BASE_DIR}" && go run ./hack/gen-prometheus-rules) > "${STAGE}/rules.yaml"
"${PROMTOOL}" check rules --lint-fatal "${STAGE}/rules.yaml"

shopt -s globstar nullglob
test_files=("${TEST_DIR}"/**/*_test.promtool.yml)
if [[ ${#test_files[@]} -eq 0 ]]; then
    echo "ERROR: no *_test.promtool.yml files found under ${TEST_DIR}" >&2
    exit 1
fi

failed=0
for file in "${test_files[@]}"; do
    rel="${file#"${TEST_DIR}/"}"
    staged="${STAGE}/${rel}"
    staged_dir="$(dirname "${staged}")"
    mkdir -p "${staged_dir}"
    cp "${file}" "${staged}"
    # Each fixture refers to rules.yaml relative to its own directory.
    if [[ "${staged_dir}" != "${STAGE}" ]]; then
        ln -sf "${STAGE}/rules.yaml" "${staged_dir}/rules.yaml"
    fi
    echo "Running ${rel}..."
    if ! "${PROMTOOL}" test rules "${staged}"; then
        echo "FAILED: ${rel}" >&2
        failed=1
    fi
done

if [[ "${failed}" -ne 0 ]]; then
    echo "ERROR: promtool rule tests failed" >&2
    exit 1
fi
