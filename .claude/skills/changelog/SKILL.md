---
name: changelog
description: Record a change in the shared multi-agent changelog. Use after any modification to code, docs or configuration, before ending the turn.
---

# Shared changelog recording

Every change is recorded as a new fragment in `changelog.d/` — `CHANGELOG.md`
is never edited by hand and existing fragments are never modified (so agents
working in parallel cannot collide).

## Steps

1. With the changes already in the working tree, run:

   ```bash
   go run ./cmd/changelog new -agent claude-code -summary "<one-line summary, imperative mood>"
   ```

   - Affected files are detected automatically from git.
   - To record only a subset: `-files path/a.go,path/b.go`.
   - The command also recompiles `CHANGELOG.md` automatically.

2. Verify `CHANGELOG.md` reflects the entry (do not edit it by hand).

## Rules

- One fragment per logical change, not per file.
- The summary must not contain the `---` sequence.
- Summaries are written in English.
- When working on a blueprint phase, mention it in the summary
  (e.g. "Phase 1: add XML v26 parser").
