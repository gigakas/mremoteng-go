#!/usr/bin/env bash
# Smoke test: builds every binary, verifies the desktop binary reports its
# version (headless-safe) and that the changelog is reproducible. Real
# coverage lives in the unit tests; the GUI window needs a desktop session.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
command -v go >/dev/null 2>&1 || export PATH="$HOME/.local/go/bin:$PATH"

out_dir=$(mktemp -d)
trap 'rm -rf "$out_dir"' EXIT

# The x11 tag is required for the RDP window embedding (see scripts/dev.sh).
go build -tags x11 -o "$out_dir/" ./...

output=$("$out_dir/mremoteng" -version)
if ! echo "$output" | grep -q "mremoteng-go"; then
    echo "smoke FAILED: unexpected output from main binary: $output" >&2
    exit 1
fi

"$out_dir/changelog" compile >/dev/null
if ! git diff --quiet -- CHANGELOG.md; then
    echo "smoke FAILED: 'changelog compile' is not reproducible (CHANGELOG.md changed)" >&2
    exit 1
fi

echo "smoke OK: binaries build, mremoteng reports its version, changelog is reproducible"
