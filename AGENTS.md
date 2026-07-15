# AGENTS.md — Shared instructions (Claude Code, OpenCode, humans)

mremoteng-go is the migration of [mRemoteNG](https://github.com/mRemoteNG/mRemoteNG)
(C#/WinForms) to Go, cross-platform Linux/Windows. The master plan lives in
`docs/MIGRATION_PLAN.md`; the operational per-phase detail in `blueprint/`.

All code comments, documentation, changelog entries and audits are written
in **English**.

## Commands

```bash
./scripts/check.sh    # gofmt + go vet + go test (mandatory before finishing)
./scripts/smoke.sh    # builds binaries and verifies they start
make changelog        # regenerates CHANGELOG.md from changelog.d/
go test ./internal/changelog/           # one package
go test ./... -run 'TestName'           # one test
```

The Go toolchain is in `~/.local/go/bin` (added to PATH in `~/.bashrc`);
if `go` is not found: `export PATH="$HOME/.local/go/bin:$PATH"`.

## Shared changelog protocol (collision-free)

- `CHANGELOG.md` is **generated** — never edit it by hand.
- Every logical change = one **new** fragment in `changelog.d/` created with:
  `go run ./cmd/changelog new -agent <claude-code|opencode> -summary "<summary>"`
  (detects affected files from git, recompiles CHANGELOG.md by itself).
- Never modify or delete existing fragments: since each change is a new
  file, two agents working in parallel cannot collide in the changelog.
- Each record captures UTC date/time, agent, summary and affected files.

## Multi-agent collision avoidance

- Before working, check `blueprint/README.md`: each phase declares which
  packages it owns. An agent works **one stage** at a time and only touches
  that phase's packages.
- If you must touch another phase's package, record the reason in the
  changelog fragment and do it as a separate, minimal change.
- Never two agents on the same stage simultaneously; stages are claimed by
  marking them `in progress (<agent>)` in the phase's status table.

## Stage closing (mandatory)

1. `./scripts/check.sh` and `./scripts/smoke.sh` green.
2. Audit in `auditory/` (code quality, performance, architecture) using
   `auditory/TEMPLATE.md` — see rules in `auditory/README.md`.
3. Changelog fragment for the audit.
4. If the stage completes a **phase**: update the top-level `README.md`
   (project status and available functionality).

## Code and refactoring conventions

- `gofmt` is mandatory (automatic hook in Claude Code; in OpenCode, run
  `./scripts/check.sh` before finishing).
- Small single-responsibility packages under `internal/`; binaries in
  `cmd/` are thin layers over `internal/`.
- No new external dependencies without a justification recorded in the
  stage audit; prefer the standard library.
- Refactor continuously and in commits separate from functional changes:
  if while touching a package you find duplication or >50-line functions
  with mixed responsibilities, extract first, then implement.
- Errors: wrap with context (`fmt.Errorf("...: %w", err)`); never discard
  with `_` unless justified in a comment.
- Tests: `TestFunction_Scenario_ExpectedResult`, table-driven cases,
  `t.TempDir()` for files. Every new package ships with unit tests.
- RDP/AnyDesk: external-process integration **only** (see
  `docs/MIGRATION_PLAN.md`, GPLv2 vs Apache-2.0 license restriction).
  Never cgo against libfreerdp.
