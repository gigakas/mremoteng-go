# Repository instructions

## Workflow sources

- Write code comments, documentation, changelog entries and audits in English.
- The active delivery track is `blueprint/linux-preview.md`; it overrides the
  historical phase ordering in `blueprint/README.md`, which still governs agent
  coordination, ownership and closing. OpenCode loads `blueprint/README.md` via
  `opencode.json`. The design rationale is `docs/MIGRATION_PLAN.md`; do not infer
  implementation status from its roadmap.
- Stage audits use `auditory/TEMPLATE.md` and `auditory/README.md`. Committed
  audits are immutable; create a new audit for a re-review.

## Commands and verification

Run from the repository root with Go 1.23+. If Go is not on PATH, use
`export PATH="$HOME/.local/go/bin:$PATH"`; both scripts below do this automatically.

```bash
./scripts/check.sh    # mandatory before finishing: formatting check -> vet -> all tests
./scripts/smoke.sh    # required for stage closure: build, skeleton startup, changelog check
make changelog       # regenerate CHANGELOG.md

go test ./internal/serialize/xml/                      # one package
go test ./internal/serialize/xml/ -run '^TestName$' -count=1
go test ./integration/ -run '^TestCSharpCorpus_' -count=1 -v
```

- `check.sh` does NOT format files: fix reported files with `gofmt -w <files>`,
  then rerun. Its `go test ./...` includes the compatibility integration suite.
- `smoke.sh` regenerates `CHANGELOG.md` and runs `git diff --quiet -- CHANGELOG.md`
  (working tree versus index). An intended unstaged changelog update also fails
  this check; regeneration alone does not resolve that. Do not discard the update
  or stage changes merely to hide the failure; report it when leaving work unstaged.

## Changelog and commits

- Every logical change, including docs and audits, needs a new fragment:
  `go run ./cmd/changelog new -agent opencode -summary "<imperative summary>"`.
  Use the assigned suffixed agent name for parallel OpenCode sessions;
  Claude Code uses `claude-code`.
- The command detects changed/untracked files and regenerates `CHANGELOG.md`.
  Use `-files path1,path2` to scope a fragment when unrelated changes exist.
- Never hand-edit `CHANGELOG.md` or edit/delete existing fragments in `changelog.d/`.
- Commits use only the configured Git user: no agent/model signatures,
  `Co-Authored-By` or `Generated with` trailers.

## Implementation and compatibility constraints

- After any UI change, rebuild and relaunch with `scripts/dev.sh`; the window
  title and About dialog carry version, commit and build time, so a stale window
  is visible at a glance. Do not claim a UI change is on screen without it.
- `cmd/mremoteng` is a Fyne desktop app (`internal/ui`) over the native
  `internal/profiles` store. It manages profiles only; SSH/RDP sessions are
  stage L2. Building it needs cgo plus the Fyne Linux headers (`libgl1-mesa-dev`,
  `xorg-dev`, `libxkbcommon-dev`). Build the application with `-tags x11`
  (`scripts/dev.sh` does): without it Fyne reports a Wayland context and the RDP
  session cannot be embedded. UI tests use Fyne's software driver and must
  stay headless: do not add tests that open a real window, call `Canvas().Capture()`
  in a long-lived test, or require `$DISPLAY`.
- The optional data core is `internal/connection` + `internal/security` +
  `internal/serialize/{xml,csv}`; serializers consume the shared connection model
  and remain regression coverage, not a prerequisite for Linux features.
- XML reads 2.6/2.7/2.8 and writes 2.8; missing attributes must retain defaults.
  Preserve C# field/inheritance parity and CSV column order when changing the model.
- Native profiles are stored as JSON in the user config directory, atomically and
  with `0600`. Never store passwords there, and never overwrite a store that
  failed to load.
- `integration/corpus_test.go` reads `testdata/corpus/manifest.json` and validates
  fixture provenance metadata, SHA-256 digests and round trips. New acceptance
  fixtures must come from the C# app, not the Go writer. Tests can pass with
  `complete: false`: this is not Phase 1 acceptance. Stage 1.7 requires at least
  20 fixtures covering 2.6/2.7/2.8 and blocks Phase 2 (see its phase table).
- RDP/AnyDesk integration is external-process-only; never link `libfreerdp` via
  cgo (license boundary documented in the migration plan).
- New external dependencies require justification in the stage audit.
- Keep refactoring separate from functional commits; extract duplication or
  mixed-responsibility functions over 50 lines before extending them.
- Test names follow `TestFunction_Scenario_ExpectedResult`. New packages need unit
  tests; discarded errors require a justification comment.
