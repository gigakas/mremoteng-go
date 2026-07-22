---
description: Record a change in the shared multi-agent changelog and recompile CHANGELOG.md
---

Record the current change in the shared changelog. The changes must already
be in the working tree.

Run:

```bash
go run ./cmd/changelog new -agent opencode -summary "$ARGUMENTS"
```

Rules:

- Never edit `CHANGELOG.md` by hand (it is generated) and never modify
  existing fragments in `changelog.d/` — every change is a new fragment.
- One fragment per logical change, not per file.
- The summary must explain the change, not just name it: what was done and,
  when not obvious, why — `CHANGELOG.md` is the project's bitácora, so a
  bare title is not acceptable.
- The summary is one line, imperative mood, in English, without the `---`
  sequence.
- Affected files are detected from git automatically; use `-files a.go,b.go`
  to record only a subset.
- When working on a blueprint phase, mention it in the summary.
