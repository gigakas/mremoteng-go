#!/usr/bin/env bash
# Smoke test: builds every binary and verifies the main one starts and
# responds. Fast by design; real coverage lives in the unit tests.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
command -v go >/dev/null 2>&1 || export PATH="$HOME/.local/go/bin:$PATH"

out_dir=$(mktemp -d)
trap 'rm -rf "$out_dir"' EXIT

go build -o "$out_dir/" ./...

output=$("$out_dir/mremoteng")
if ! echo "$output" | grep -q "mremoteng-go"; then
    echo "smoke FAILED: unexpected output from main binary: $output" >&2
    exit 1
fi

"$out_dir/changelog" compile >/dev/null
if ! git diff --quiet -- CHANGELOG.md; then
    echo "smoke FAILED: 'changelog compile' is not reproducible (CHANGELOG.md changed)" >&2
    exit 1
fi

echo "smoke OK: binaries build, mremoteng starts, changelog is reproducible"
