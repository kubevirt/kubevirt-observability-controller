#!/usr/bin/env bash
#
# Runs promtool rule unit tests against the rules defined in pkg/monitoring/rules.
#

set -euo pipefail

BASE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
PROMTOOL="${PROMTOOL:-${BASE_DIR}/bin/promtool}"
TEST_DIR="${BASE_DIR}/pkg/monitoring/rules"
TEST_GLOB='*_test.promtool.yml'

if [[ ! -x "${PROMTOOL}" ]]; then
    echo "ERROR: promtool not found at ${PROMTOOL}. Run 'make promtool'." >&2
    exit 1
fi

unset RUNBOOK_URL_TEMPLATE

STAGE=$(mktemp -d "${TMPDIR:-/tmp}/promtool-tests.XXXXXX")
trap 'rm -rf "${STAGE}"' EXIT

echo "Generating rules from pkg/monitoring/rules..."
(cd "${BASE_DIR}" && go run ./hack/gen-prometheus-rules) > "${STAGE}/rules.yaml"

echo "Checking rule syntax..."
"${PROMTOOL}" check rules --lint-fatal "${STAGE}/rules.yaml"

test_files=()
while IFS= read -r file; do
    test_files+=("${file}")
done < <(find "${TEST_DIR}" -name "${TEST_GLOB}" | sort)

if [[ ${#test_files[@]} -eq 0 ]]; then
    echo "ERROR: no ${TEST_GLOB} files found under ${TEST_DIR}" >&2
    exit 1
fi

failed=0
for file in "${test_files[@]}"; do
    rel="${file#"${BASE_DIR}/"}"
    staged="${STAGE}/$(printf '%s' "${rel#pkg/monitoring/rules/}" | tr '/' '_')"
    cp "${file}" "${staged}"

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

echo "All promtool rule tests passed."
