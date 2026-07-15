#!/usr/bin/env bash
# PostToolUse hook (Claude Code): formats .go files after every Edit/Write.
# Never fails the turn: exits silently when the toolchain is missing.
set -uo pipefail
cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0
command -v gofmt >/dev/null 2>&1 || export PATH="$HOME/.local/go/bin:$PATH"
command -v gofmt >/dev/null 2>&1 || exit 0
gofmt -w . 2>/dev/null
exit 0
