---
name: changelog
description: Record a change in the shared multi-agent changelog with a detailed description of every change made. Use after any modification to code, docs or configuration, before ending the turn.
---

# Shared changelog recording (bitácora)

Every change is recorded as a new fragment in `changelog.d/` — `CHANGELOG.md`
is never edited by hand and existing fragments are never modified (so agents
working in parallel cannot collide).

**Each fragment must explain what changed and why.** The changelog is a
bitácora (logbook): a future reader should understand the change from the
fragment alone, without inspecting the diff.

## Steps

1. With the changes already in the working tree, run:

   ```bash
   go run ./cmd/changelog new \
     -agent opencode \
     -summary "<one-line summary, imperative mood>" \
     -description "<detailed multi-line explanation>"
   ```

   - `-summary` is the title (one line, imperative mood).
   - `-description` is **required** — explain every change made: what was
     added/modified/removed, why, and any side effects or trade-offs.
   - Affected files are detected automatically from git.
   - To record only a subset: `-files path/a.go,path/b.go`.
   - The command also recompiles `CHANGELOG.md` automatically.

2. Verify `CHANGELOG.md` reflects the entry (do not edit it by hand).

## What goes in the description

Write 2–10 sentences (or bullet points) covering:

- **What** changed (files, functions, behaviour).
- **Why** it changed (bug, feature, refactor, decision).
- **How** it was validated (tests, manual checks, cross-references).
- Any **deviation** from the blueprint or a **trade-off** introduced.

If the change is trivial (e.g. a typo fix), a single sentence suffices — but
the field must never be empty.

## Example

```bash
go run ./cmd/changelog new \
  -agent opencode \
  -summary "Add -description flag to the changelog tool" \
  -description "The Entry struct now carries a Description field parsed from the fragment body (first paragraph = summary, rest = description). Render outputs the description as an indented paragraph between the summary and the file list. The CLI warns when -description is omitted so the bitácora stays complete. Added four unit tests covering parsing with/without description and both render layouts."
```

## Rules

- One fragment per logical change, not per file.
- The summary must not contain the `---` sequence.
- Summaries and descriptions are written in English.
- When working on a blueprint phase, mention it in the summary
  (e.g. "Phase 1: add XML v26 parser").
- **Never** omit `-description` — if you do, the tool prints a warning.
