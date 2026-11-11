# Audit — Linux preview, dead-code removal (not a phase stage)

- **Date (UTC)**: 2026-11-11
- **Agent**: mimocode
- **Audited stage**: Removal of code the shipped binary cannot execute
  (`internal/ui/ansi.go`, `internal/ui/framebuffer.go`, the
  `internal/protocol` root package and four of its backends, plus one unwired
  UI function), and the documentation correction that removal forced.
- **Commits covered**: working tree only — the deletions described here are
  staged but uncommitted at audit time; the fragment created alongside this
  file records them. Historical references: `b58c1a1` ("Linux preview:
  standalone connection manager with SSH, RDP, VNC and Telnet") and
  `5dda3b0` ("Fix the broken merge: keep the tested application and drop
  duplicates").

This audit exists because AGENTS.md forbids editing committed audits, and nine
of them record the affected packages — eight as the delivered stage work that
built or used the removed code, one (phase 1.1) as the connection model those
backends were typed against. It records what was removed, why it was
unreachable from `cmd/mremoteng`, and why those older audits stay as they are.

## 1. Code quality

Removed, in dependency order (each step was verified with `grep` before
deleting; nothing turned out to be live):

1. **`internal/ui/ansi.go` (345 non-test LOC) + `internal/ui/ansi_test.go`
   (175 LOC, 12 tests).** Superseded by `internal/terminal` (vt10x), which is
   what the live widget uses: `internal/ui/terminal.go:14` imports
   `internal/terminal` and drives `terminal.Emulator`
   (`internal/ui/terminal.go:23`). `newANSIState` had no reference outside
   `ansi.go` (`grep -rn 'newANSIState\|ansiState' --include=*.go .` matched
   only `ansi.go` and its own test). `termStyle`, `max0`, `min`, `clamp` and
   `firstParam` were likewise package-local helpers with no other caller.
   Note on the one coupling that mattered: `internal/ui/ui_test.go:1908`
   calls `min(len(content), 40)`, which resolved to the shadowing
   `func min(a, b int)` at the old `ansi.go:284`. After the deletion
   `grep -rn '^func min(' internal/ui/` is empty, the call resolves to the
   Go 1.21+ builtin, and `go vet`/`go test` pass unchanged — no edit to
   `ui_test.go` was needed.
2. **`internal/ui/framebuffer.go` (190 non-test LOC) +
   `internal/ui/framebuffer_test.go` (164 LOC, 4 tests).** The live VNC panel
   renders through `internal/ui/vnc.go` (`vncSession`/`canvas.Image`,
   `internal/ui/vnc.go:20-27`). `FramebufferView.Attach` had no caller outside
   its own test (`grep -rn '\.Attach(\|NewFramebufferView'`), and `fakeFramebuffer`
   in `framebuffer_test.go:22` was the only implementation of
   `protocol.FramebufferProtocol`. `internal/ui/framebuffer.go:12` was also the
   sole production import of the `internal/protocol` root package from
   `internal/ui`.
3. **`internal/protocol/{raw,rlogin,serial,anydesk}` — 570 non-test LOC, 431
   test LOC, 12 tests.** raw.go 76/raw_test.go 157, rlogin.go 140/rlogin_test.go
   155, serial.go 92/serial_test.go 52, anydesk.go 125 + embed_linux.go 68 +
   embed_windows.go 69/anydesk_test.go 67. Only their own tests imported them
   (`*_test.go` blank imports at `raw_test.go:14`, `rlogin_test.go:13`,
   `serial_test.go:8`, `anydesk_test.go:9`); the only other mentions are doc
   comments (`anydesk.go:54`, `internal/ui/nativewindow_other.go:12`). The
   capabilities those packages claimed are still delivered elsewhere:
   RAW and RLOGIN run through `internal/protocol/telnet`'s modes
   (`internal/ui/stream.go:35-45` maps `profiles.RAW`/`profiles.RLOGIN` to
   `telnetclient.ModeRaw`/`ModeRlogin`), and `internal/profiles/profiles.go:21-31`
   defines no SERIAL or AnyDesk protocol constant at all, so the serial and
   AnyDesk backends could not be selected from any code path in the binary.
   `go.bug.st/serial` was referenced only by `internal/protocol/serial/serial.go:12`.
4. **`internal/protocol` root package — 333 non-test LOC, 185 test LOC, 7
   tests**: protocol.go 62, factory.go 68, lifecycle.go 119, terminal.go 20,
   window.go 18, framebuffer.go 46, protocol_test.go 185. After steps 2 and 3
   it had no production importer. The registry was dead by construction:
   `protocol.Register` was called only from `init()` in
   `raw.go:17`, `rlogin.go:19`, `serial.go:19`, `anydesk.go:31` and from
   `internal/protocol/protocol_test.go`, `protocol.Create` only from tests
   (`raw_test.go:79`, `rlogin_test.go:68`, `serial_test.go:25`,
   `anydesk_test.go:20`, `protocol_test.go:97`), and `cmd/mremoteng/main.go`
   imports only `internal/profiles` and `internal/ui`
   (`cmd/mremoteng/main.go:13-14`) — no blank import of any backend, so the
   shipped binary's registry was always empty and `Create` could only ever
   return `protocol: no backend registered`. `Lifecycle` had no user outside
   `internal/protocol/` either: `b58c1a1` rewrote rdp, ssh, vnc and telnet to
   own their callback bookkeeping (`git show b58c1a1:internal/protocol/ssh/ssh.go`
   and `.../telnet/telnet.go` contain neither the root import nor
   `protocol.Register`), and `5dda3b0` kept those rewrites and dropped the
   registry-based duplicates (`internal/protocol/web`, `.../winrm`).
5. **`internal/ui/folders.go:308-319` — `requestMoveConnection` (12 LOC + 1
   blank line).** No call site anywhere in the tree; moves are served by the
   row context menu's **Move to** submenu (`internal/ui/menu.go:44-45`,
   `internal/ui/menu.go:68-86`), drag-and-drop (`internal/ui/drop.go:43`) and
   `menu.go:71`/`menu.go:83`, all of which call `moveConnection`
   (`internal/ui/folders.go:309`) directly. `showTextDialog` keeps five other
   callers, so nothing else became dead.

Totals: **1,451 non-test LOC and 955 test LOC (2,406 lines)** removed, with
**35 top-level test functions** (12 + 4 + 12 + 7). Two dependency entries
disappeared: `go.mod` lost `go.bug.st/serial v1.8.0` and `go.sum` lost exactly
its two lines; nothing else in either file changed (`git diff go.mod go.sum`
after `go mod tidy`).

The one documentation claim that removal falsified was fixed rather than
inherited: `README.md:25` advertised **File > Move connection to folder**, a
menu item that does not exist — the File menu is `internal/ui/shell.go:250-265`
(Credentials, Import, Export XML/CSV, Quick connect, New folder, New
connection, Save connection, Delete connection, Quit). It now points at the row
right-click **Move to** submenu, keeping the capability documented.
`docs/linux-preview-usability.md:76` already described the submenu correctly
and needed no change.

No technical debt was introduced: this change only deletes, and the two
substitutions it relies on (`internal/terminal` for `ansiState`,
`internal/ui/vnc.go` for `FramebufferView`) were already shipping before it.

## 2. Performance

Not applicable in the measured sense: no code path changes behaviour, because
nothing removed was reachable from `cmd/mremoteng` through any call site. The
binary's import graph loses five packages (17 files; 21 files leave the tree
counting the four in `internal/ui`), which drops their `init` work and a
`sync.RWMutex`-guarded registry map that never held an entry. No
benchmark was run — there is no before/after runtime difference to measure, and
the task explicitly excluded the full gate.

## 3. Architecture

- **Package boundary**: the removal moves the tree closer to what the import
  graph already said. `cmd/mremoteng` now depends on
  `internal/{profiles,connection,security,serialize/{xml,csv},transfer,vault,terminal,ui}`
  plus `internal/protocol/{rdp,ssh,telnet,vnc}` — and, notably, on **no**
  `internal/protocol` root package at all. The surviving backends are the ones
  the UI actually drives (`internal/ui/session.go:17-18`,
  `internal/ui/shell.go:20`, `internal/ui/stream.go:12`,
  `internal/ui/vnc.go:16`, `internal/ui/nativewindow_windows.go:12` for
  `winembed`).
- **Dependencies**: one external dependency removed, no new dependency added.
  `go.bug.st/serial` was the serial backend's only consumer; the blueprint's
  serial-port capability is unimplemented in the Linux preview and
  `internal/profiles` has no constant for it, so dropping the module is
  consistent with the delivered feature set rather than a capability
  regression (`README.md:107-110` already says "Serial ports
  are still pending").
- **Deviations**: this is not a blueprint stage and does not close one; it
  touches packages owned by phases 2 and 3 as *deletions only*. Where the
  historical phase documents still describe `internal/protocol/` as live
  architecture (`blueprint/phase-2-protocols.md:8`, `:24-29`) that is left
  untouched because it is the historical record of a closed track, matching the
  immutable-audit rule. `blueprint/linux-preview.md:41` ("L2 owns
  `internal/protocol/` and `internal/terminal/`") is on the active track and is
  now partly stale — recorded as a pending action instead of an edit here,
  because that file has uncommitted changes from a concurrent agent.
- **Deliberately kept**: the credential provider packages
  (`internal/credential`, `.../vault`, `.../delinea`, `.../passwordstate`,
  `.../onepassword`, `.../awsec2`) are scaffold for backlog stage L9, not debt;
  the connection tree/inheritance API used only by tests is a separate
  decision; `internal/terminal`, `internal/vault`, `internal/transfer`,
  `internal/serialize` and `cmd/changelog` are out of scope and untouched.
- **Impact on future phases**: `internal/credential/*` (L9) and any future
  protocol work now have no `internal/protocol` contract to implement against;
  a future backend plugs into the UI the way the surviving four do (direct
  client package + `internal/ui` wiring), which is what `b58c1a1` established.

## 4. Evidence

Import-graph evidence, captured before the deletions:

```
$ go list -deps ./cmd/mremoteng | grep 'internal/\(ui\|protocol\|terminal\)'
github.com/mRemoteNG/mremoteng-go/internal/protocol
github.com/mRemoteNG/mremoteng-go/internal/protocol/rdp
github.com/mRemoteNG/mremoteng-go/internal/protocol/ssh
github.com/mRemoteNG/mremoteng-go/internal/protocol/telnet
github.com/mRemoteNG/mremoteng-go/internal/protocol/vnc
github.com/mRemoteNG/mremoteng-go/internal/terminal
github.com/mRemoteNG/mremoteng-go/internal/ui
```

`internal/protocol/{raw,rlogin,serial,anydesk}` never appear — they were never
in the shipped binary. The root package did appear, but only through
`internal/ui/framebuffer.go:12`; a per-package check isolated it:

```
$ for p in $(go list ./internal/... ./cmd/...); do
    go list -f '{{join .Imports "\n"}}' $p | grep -q 'internal/protocol$' && echo "$p"; done
.../internal/protocol/anydesk
.../internal/protocol/raw
.../internal/protocol/rlogin
.../internal/protocol/serial
.../internal/ui
```

All five importers are gone after steps 2-4. The same command on the current
tree prints nothing, and:

```
$ go list -deps ./cmd/mremoteng | grep 'mremoteng-go/internal'
.../internal/profiles  .../internal/connection  .../internal/protocol/rdp
.../internal/protocol/ssh  .../internal/protocol/telnet  .../internal/protocol/vnc
.../internal/security  .../internal/serialize/xml  .../internal/terminal
.../internal/serialize/csv  .../internal/transfer  .../internal/vault  .../internal/ui
```

— no `internal/protocol` root entry remains.

- `./scripts/check.sh`: **not run, by design.** A concurrent agent has
  uncommitted work in `internal/serialize/xml` and `internal/connection`, so the
  repo-wide gate would report its half-finished state, not this change. The
  lead runs it after both land.
- `./scripts/smoke.sh`: **not run** for the same reason (it also regenerates
  `CHANGELOG.md`, which a concurrent fragment will rewrite). The lead runs it.
- Scoped validation actually run:
  - `go build -tags x11 ./internal/ui/ ./cmd/...` — clean.
  - `go build ./internal/protocol/...` — clean (surviving backends still build).
  - `go vet -tags x11 ./internal/ui/ ./cmd/...` — clean.
  - `go test ./internal/ui/ -count=1` — `ok ... 0.849s`.
  - `go vet ./integration/` — clean (`go build ./integration/` only reports that
    the package has no non-test Go files; `integration/corpus_test.go` was not
    executed — the lead runs the corpus suite).
  - `go mod tidy` — `git diff go.mod go.sum` restricted to `go.bug.st/serial
    v1.8.0` (go.mod) and its two `go.sum` lines, as required.
- New tests in this stage: **none** — a deletion cannot add behaviour; the
  removed tests' coverage is recorded above so the loss is auditable rather
  than silent. No surviving test was modified; `internal/ui/ui_test.go:1908`
  was left byte-identical.

Audits covering the removed code, **left untouched** (immutable per AGENTS.md;
each remains an accurate record of what was delivered, built and tested at its
own date, and this file is the re-review that supersedes their live-code
claims):

- `auditory/phase1-stage1-20260716-opencode.md` — connection model, incl. the
  `ProtocolSerial` constant that stays in `internal/connection`
  (`internal/profiles` never had one).
- `auditory/phase2-stage1-20260722-claude-code.md` — `internal/protocol/protocol.go`
  and `factory.go` registry.
- `auditory/phase2-stage2-20260723-claude-code.md` — telnet/ssh/rlogin/raw/serial
  backends and the shared `internal/protocol` helpers.
- `auditory/phase2-stage3-20260723-claude-code.md` — `internal/protocol/window.go`.
- `auditory/phase2-stage4-20260723-claude-code.md` — `internal/protocol/framebuffer.go`
  and the VNC backend.
- `auditory/phase2-stage7-20260723-claude-code.md` — the AnyDesk backend.
- `auditory/phase3-stage3-20260723-claude-code.md` — `internal/ui/ansi.go`,
  `internal/ui/terminal.go`, `internal/ui/framebuffer.go`.
- `auditory/phase3-stage6-20260724-claude-code.md` — theme note citing the ANSI
  palette in `internal/ui/ansi.go`.
- `auditory/phase3-stage7-20260724-claude-code.md` — notes `internal/credential`
  being disjoint from `internal/protocol`.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

The deletions themselves are verified in scope; what is pending is the
repo-wide gate, which cannot run while another agent's work is in flight.

## 6. Pending actions

1. **Lead**: run `./scripts/check.sh` and `./scripts/smoke.sh` once
   `internal/serialize/xml` and `internal/connection` are committed, then run
   `go test ./integration/ -run '^TestCSharpCorpus_' -count=1 -v`. Owner:
   lead. This file records no result for either script, deliberately.
2. **Lead**: regenerate `CHANGELOG.md` after the concurrent
   `xml-retain-mremoteng-c-defaults` fragment lands — the two fragments race
   and the last `cmd/changelog new` wins the rendered file.
3. **Blueprint owner**: `blueprint/linux-preview.md:41` still says L2 owns
   `internal/protocol/`; it now owns the four surviving backend packages only.
   Left untouched here because that file has another agent's uncommitted
   changes. Owner: active-track lead. `blueprint/phase-2-protocols.md` is
   historical and should not be updated.
4. **Backlog**: if serial-port or AnyDesk support is genuinely wanted on the
   Linux preview, it needs a new implementation against
   `internal/profiles` (no protocol constant exists) — the deleted packages are
   recoverable from history up to this change's parent commit.
