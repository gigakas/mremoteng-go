#!/usr/bin/env bash
# Smoke test: builds every binary and verifies the main one starts and
# responds. Fast by design; real coverage lives in the unit tests.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
command -v go >/dev/null 2>&1 || export PATH="$HOME/.local/go/bin:$PATH"

out_dir=$(mktemp -d)
trap 'rm -rf "$out_dir"' EXIT

go build -o "$out_dir/" ./...

# mremoteng is a real GUI app since Phase 3 stage 3.1 (ShowAndRun blocks
# until the window closes), so "starts and responds" means: launch it,
# give it a moment to initialize, confirm the process is still alive
# (didn't crash on startup), then terminate it. This replaced an earlier
# version that captured stdout output, which no longer applies now that
# there's nothing printed to a terminal.
"$out_dir/mremoteng" &
mremoteng_pid=$!
sleep 2
if ! kill -0 "$mremoteng_pid" 2>/dev/null; then
    wait "$mremoteng_pid" 2>/dev/null
    echo "smoke FAILED: mremoteng exited immediately (crashed on startup?)" >&2
    exit 1
fi
kill "$mremoteng_pid" 2>/dev/null
wait "$mremoteng_pid" 2>/dev/null || true

"$out_dir/changelog" compile >/dev/null
if ! git diff --quiet -- CHANGELOG.md; then
    echo "smoke FAILED: 'changelog compile' is not reproducible (CHANGELOG.md changed)" >&2
    exit 1
fi

echo "smoke OK: binaries build, mremoteng starts, changelog is reproducible"
