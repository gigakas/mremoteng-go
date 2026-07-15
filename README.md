# mremoteng-go

Migration of [mRemoteNG](https://github.com/mRemoteNG/mRemoteNG)
(C#/WinForms, Windows-only) to Go, targeting native Linux support while
keeping Windows portability.

**Status**: **Phase 0 (embedding spike) closed — GO decision.** The
project's core premise is validated on both platforms: external FreeRDP
clients embedded in a Fyne window (X11/XWayland reparenting on Linux,
`SetParent` on Windows). Evidence in [`docs/spike-result.md`](docs/spike-result.md)
and the per-stage findings (`docs/spike-x11.md`, `docs/spike-win32.md`,
`docs/spike-wayland.md`). Phase 1 (data core) is underway: encryption
(`internal/security`, AES-256-GCM + PBKDF2 + legacy Rijndael read) is done.
No end-user functionality yet. See [`docs/MIGRATION_PLAN.md`](docs/MIGRATION_PLAN.md)
for the master plan and [`blueprint/`](blueprint/) for the per-stage detail.

## Layout

- `cmd/mremoteng` — application entry point.
- `cmd/changelog` — shared changelog tool (see below).
- `internal/connection` — connection model and container tree.
- `internal/serialize/xml`, `internal/serialize/csv` — connection file
  serializers (compatible with the original project's formats).
- `internal/security` — encryption and key derivation.
- `internal/protocol` — protocol implementations (SSH, RDP, VNC, ...).
- `internal/ui` — graphical interface (Fyne).
- `internal/changelog` — shared changelog engine.
- `blueprint/` — per-phase operational plan and agent coordination rules.
- `auditory/` — stage closing audits (quality, performance, architecture).
- `changelog.d/` — changelog fragments (source of the generated `CHANGELOG.md`).

## Development

```bash
./scripts/check.sh    # gofmt + go vet + go test
./scripts/smoke.sh    # build all binaries and verify they start
make changelog        # regenerate CHANGELOG.md from changelog.d/
```

Requires Go 1.23+.

## Multi-agent workflow

This repo is worked on by multiple coding agents (Claude Code, OpenCode) and
humans. Shared instructions live in [`AGENTS.md`](AGENTS.md) (OpenCode reads
it natively; `CLAUDE.md` imports it). Key rules:

- Every change is recorded in the shared changelog via
  `go run ./cmd/changelog new` — one fragment per change in `changelog.d/`,
  compiled chronologically into `CHANGELOG.md` (never edited by hand).
- Work is claimed per blueprint stage; each phase owns specific packages to
  avoid agents colliding in the same code.
- Every finished stage gets an audit in `auditory/` and, when it completes a
  phase, this README is updated.
