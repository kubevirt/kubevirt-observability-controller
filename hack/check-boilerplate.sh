#!/usr/bin/env bash

set -euo pipefail

BOILERPLATE="hack/boilerplate.go.txt"
BOILERPLATE_LINES=$(wc -l < "$BOILERPLATE")

failed=0

while IFS= read -r file; do
    if ! head -n "$BOILERPLATE_LINES" "$file" | diff -q "$BOILERPLATE" - > /dev/null 2>&1; then
        echo "ERROR: Missing or incorrect boilerplate header: $file"
        failed=1
    fi
done < <(find . -name '*.go' -not -path './vendor/*' -not -path './bin/*')

if [ "$failed" -eq 1 ]; then
    echo ""
    echo "Run 'make add-boilerplate' to add the boilerplate header to all files."
    exit 1
fi
