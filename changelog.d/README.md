# changelog.d/ — Shared changelog fragments

One file per change, created with `go run ./cmd/changelog new ...` — do not
create by hand and do not edit existing fragments. `CHANGELOG.md` is
generated from here with `make changelog`.

Fragment format:

```markdown
---
timestamp: 2026-07-15T20:45:00Z
agent: claude-code
files:
  - internal/connection/info.go
---

One-line change summary.
```

The file name (`YYYYMMDDTHHMMSSZ-<agent>-<slug>.md`) is unique per timestamp
and agent, which makes collisions between agents working in parallel
impossible. This README is the only file in the directory that is not a
fragment.
