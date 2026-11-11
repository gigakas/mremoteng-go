---
timestamp: 2026-11-11T22:02:12Z
agent: mimocode-cleanup
files:
  - internal/ui/ansi.go
  - internal/ui/ansi_test.go
  - internal/ui/framebuffer.go
  - internal/ui/framebuffer_test.go
  - internal/ui/folders.go
  - internal/protocol
  - go.mod
  - go.sum
  - README.md
  - auditory/linux-deadcode-20261111-mimocode.md
---

Delete code the shipped binary cannot execute

Remove every package the mremoteng binary could never reach, proven with go list -deps ./cmd/mremoteng, and record the deletion in a new audit.

Deleted in import-graph order:
1. internal/ui/ansi.go (345 non-test LOC) + ansi_test.go (175 LOC, 12 tests). Superseded by internal/terminal (vt10x), which internal/ui/terminal.go drives; newANSIState had no caller outside its own file. The ui_test.go call to min() now resolves to the Go builtin, no test edit needed.
2. internal/ui/framebuffer.go (190 non-test LOC) + framebuffer_test.go (164 LOC, 4 tests). The live VNC panel renders through internal/ui/vnc.go; Attach was called only from its own test and fakeFramebuffer was the only protocol.FramebufferProtocol implementation.
3. internal/protocol/{raw,rlogin,serial,anydesk} (570 non-test LOC, 431 test LOC, 12 tests). Only their own tests imported them; RAW and RLOGIN are served by internal/protocol/telnet modes via internal/ui/stream.go, and internal/profiles has no SERIAL or AnyDesk protocol constant. go mod tidy then dropped go.bug.st/serial v1.8.0, the only external dependency lost.
4. internal/protocol root package (333 non-test LOC, 185 test LOC, 7 tests): protocol.go, factory.go, lifecycle.go, terminal.go, window.go, framebuffer.go. No production importer remained: Register was called only from init() funcs in the packages above, Create only from tests, and cmd/mremoteng blank-imports no backend, so the shipped registry was always empty.
5. internal/ui/folders.go: requestMoveConnection (12 LOC) had no call site; moves are served by the row context menu Move to submenu, drag-and-drop and the File menu paths that call moveConnection directly.

Totals: 1451 non-test LOC and 955 test LOC (2406 lines), 35 test functions, one external dependency and five packages.

README.md:25 advertised a File > Move connection to folder menu item that does not exist; it now points at the row right-click Move to submenu, so the capability stays documented and only the false path is gone.

New audit auditory/linux-deadcode-20261111-mimocode.md follows auditory/TEMPLATE.md and explains why the nine audits that record these packages as delivered stages stay untouched (committed audits are immutable), what deliberately stayed (L9 credential providers, the connection test-only API), and the pending repo-wide gate.

Validation was scoped to avoid a concurrent agent's uncommitted work in internal/serialize/xml and internal/connection: go build -tags x11 ./internal/ui/ ./cmd/..., go build ./internal/protocol/..., go vet -tags x11 ./internal/ui/ ./cmd/..., go vet ./integration/, go test ./internal/ui/ -count=1 (ok, 0.849s), and go mod tidy with git diff confirming only go.bug.st/serial left go.mod/go.sum. ./scripts/check.sh and ./scripts/smoke.sh are left to the lead.
