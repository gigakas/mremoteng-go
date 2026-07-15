#!/usr/bin/env bash
# Full verification: formatting, static analysis and unit tests.
# Both agents run this before considering a change done.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
command -v go >/dev/null 2>&1 || export PATH="$HOME/.local/go/bin:$PATH"

unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
    echo "gofmt needed on:" >&2
    echo "$unformatted" >&2
    echo "run: gofmt -w ." >&2
    exit 1
fi

go vet ./...
go test ./...
echo "check OK: gofmt + go vet + go test"
