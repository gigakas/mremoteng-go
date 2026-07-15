#!/usr/bin/env bash
# Stop hook (Claude Code): prevents ending the turn with working-tree changes
# that have no corresponding changelog fragment.
set -uo pipefail
input=$(cat)

# Avoid an infinite loop: if the turn is already continuing because of this
# hook, let it pass.
if printf '%s' "$input" | grep -q '"stop_hook_active"[[:space:]]*:[[:space:]]*true'; then
    exit 0
fi

cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || exit 0

# Real changes, excluding the changelog system itself.
changed=$(git status --porcelain | grep -vE ' (changelog\.d/|CHANGELOG\.md)' || true)
[ -z "$changed" ] && exit 0

# Is there any new (uncommitted) fragment recording these changes?
fragments=$(git status --porcelain -- changelog.d/ | grep -E '^(\?\?|A )' || true)
[ -n "$fragments" ] && exit 0

echo "There are changes not recorded in the shared changelog. Run: go run ./cmd/changelog new -agent claude-code -summary \"<change summary>\" (it detects affected files and recompiles CHANGELOG.md automatically)." >&2
exit 2
