---
description: Record a change in the shared multi-agent changelog and recompile CHANGELOG.md
---

Record the current change in the shared changelog. The changes must already
be in the working tree.

Run:

```bash
go run ./cmd/changelog new \
  -agent opencode \
  -summary "<one-line title, imperative mood>" \
  -description "<what changed, why, how it was validated>"
```

`$ARGUMENTS` is the raw request behind this change — use it to write the
`-summary`/`-description` above, don't pass it through as-is.

Rules:

- Never edit `CHANGELOG.md` by hand (it is generated) and never modify
  existing fragments in `changelog.d/` — every change is a new fragment.
- One fragment per logical change, not per file.
- `-summary` is a short title, one line, imperative mood, without the `---`
  sequence.
- `-description` is where the bitácora explanation goes — **required**
  unless the change is genuinely trivial (e.g. a typo fix); the tool warns
  if it's omitted. Cover what changed, why, how it was validated, and any
  deviation/trade-off worth recording.
- Affected files are detected from git automatically; use `-files a.go,b.go`
  to record only a subset.
- When working on a blueprint phase, mention it in the summary.
