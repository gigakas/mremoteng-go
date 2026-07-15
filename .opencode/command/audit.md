---
description: Produce the closing audit for a blueprint stage (code quality, performance, architecture)
---

Produce the closing audit for the stage given in `$ARGUMENTS`
(expected format: `<phase> <stage>`, e.g. `1 2`).

1. Copy `auditory/TEMPLATE.md` to
   `auditory/phase<N>-stage<M>-YYYYMMDD-opencode.md` (today's UTC date).
2. Fill in **every** section with concrete findings (`file:line` where
   applicable): code quality, performance, architecture, verdict and
   pending actions.
3. Run `./scripts/check.sh` and `./scripts/smoke.sh` and record the results
   in the evidence section.
4. Record the audit in the changelog:
   `go run ./cmd/changelog new -agent opencode -summary "Phase <N> stage <M>: closing audit"`
5. If this stage completes the phase, update the top-level `README.md`.

A stage is not considered done without its audit in `auditory/`.
