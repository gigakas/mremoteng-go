# Audit — Phase 2, Stage 2.2

- **Date (UTC)**: 2026-07-23
- **Agent**: claude-code
- **Audited stage**: SSH, Telnet, rlogin, raw socket, serial
- **Commits covered**: uncommitted at audit time; see the stage 2.2
  implementation and closing-audit changelog fragments created alongside
  this file

## 1. Code quality

- Shared helpers added to `internal/protocol` (not backend-specific, so
  placed in the package that owns the interface): `Lifecycle`
  (`lifecycle.go`) implements the OnError/OnClose callback bookkeeping
  every backend needs, and `WatchedStream` wraps an `io.ReadWriteCloser`
  so I/O errors automatically drive it. This was extracted *before*
  writing five near-identical backends, not after — the duplication was
  obvious from the interface design in stage 2.1's audit already
  anticipating five terminal backends.
- Found and fixed a real bug while extracting `WatchedStream`: the first
  version left `Close()` as the promoted `net.Conn.Close`, which errors on
  a second call — every backend's `Disconnect` would have been
  non-idempotent, violating the documented `Protocol.Disconnect` contract.
  Caught by `TestSession_ConnectReadWriteDisconnect_EchoesData` in
  `internal/protocol/raw` (double-`Disconnect` at the end of the test)
  before it reached the other four backends. Fixed once in
  `WatchedStream.Close` (a `sync.Once`) rather than five times.
- `internal/protocol/telnet/telnet.go`: the one genuinely non-trivial
  backend — a stateful IAC filter (`Read`/`handleCommand`/
  `respondNegotiation`/`skipSubnegotiation`). Kept deliberately minimal:
  refuses every option (falls back to plain NVT mode) rather than
  implementing ECHO/SGA/NAWS/terminal-type, which is the "thin custom
  implementation" the blueprint asks for. Tests exercise both directions:
  negotiation stripped from Read, and 0xFF escaped on Write.
- `internal/protocol/ssh/ssh.go`: password auth only, host key
  unconditionally accepted (`ssh.InsecureIgnoreHostKey`) — both called out
  in the `Session` doc comment and repeated below as pending actions, not
  hidden. SSH-1 is registered but its constructor always returns an error
  ("deprecated and insecure, not implemented") instead of leaving it
  silently unregistered, so a user picking SSH-1 gets a precise message
  instead of a generic "no backend registered for SSH1".
- `internal/protocol/rlogin`, `raw`: straightforward, no notable findings.
- `internal/protocol/serial`: only construction/validation is tested (see
  Evidence) — no false claim of I/O coverage.
- No duplication across the five backends beyond the unavoidable
  boilerplate of "dial → wrap in WatchedStream → expose Read/Write" that
  differs in the handshake step for each protocol.
- No function over ~50 lines; no discarded errors.

## 2. Performance

Not a hot path: one `Session` per opened tab, I/O bound by the remote end
in every case. The Telnet `Read` loop processes one buffered byte at a
time via `bufio.Reader.ReadByte`, which is a function call per byte but
not a syscall per byte (bufio does the batching) — fine at interactive
terminal data rates; would be worth revisiting only if a use case needed
bulk-transferring megabytes over Telnet, which is not this protocol's use
case.

## 3. Architecture

- `TerminalProtocol` (`internal/protocol/terminal.go`) is a new exported
  interface, `Protocol` + `io.ReadWriter`, composed rather than folded
  into `Protocol` itself — window-embedded backends (RDP/VNC/AnyDesk,
  stages 2.5/2.4/2.7) have no byte-stream concept on the caller's side, so
  forcing Read/Write on them would be meaningless. This is the concrete
  resolution of the gap flagged as a pending item implicitly present since
  stage 2.1 (the interface had no way to expose a byte stream at all).
- Amended `Protocol.Resize`'s doc comment (`protocol.go`): the unit is now
  documented as backend-defined (pixels for window-embedded backends,
  character cells for terminal backends) instead of unconditionally
  "pixels" as stage 2.1 originally wrote it, once stage 2.2 needed
  character-cell semantics for PTY/window-change sizing. This is a
  same-day correction of the prior stage's own docs, not a silent
  reinterpretation.
- Five new subpackages — `internal/protocol/{raw,rlogin,telnet,ssh,serial}`
  — one per protocol, each self-registering via `init()`, matching stage
  2.1's factory design and the blueprint's package-ownership rule; none of
  them import each other or get imported by `internal/protocol`.
- `cmd/mremoteng/main.go` blank-imports all five, resolving the pending
  action noted in stage 2.1's audit (no backend was wired into the binary
  yet).
- New dependency: `go.bug.st/serial` (for `internal/protocol/serial`) —
  the standard library has no cross-platform serial port support; this is
  the most widely used, actively maintained, pure-Go (no cgo)
  implementation. `go mod tidy` bumped `go.mod`'s `go` directive from
  `1.23` to `1.25.0` as a side effect of this dependency's own
  requirement; the installed toolchain (1.26.5) satisfies it, but this is
  worth knowing if a CI pin exists elsewhere expecting `1.23`.
- SSH1/SSH2 use the already-present `golang.org/x/crypto/ssh` dependency
  from Phase 1 (no new module).
- No impact on already-closed Phase 1 packages, or on stage 2.1's public
  API beyond the additive `TerminalProtocol` interface and the `Resize`
  doc-comment amendment (no signature changes).

## 4. Evidence

- `./scripts/check.sh`: **green** — `gofmt` clean, `go vet` clean, every
  package `ok`, including the five new subpackages.
- `./scripts/smoke.sh`: **green** — binaries build (with all five
  backends now blank-imported into `cmd/mremoteng`), `mremoteng` starts,
  changelog reproducible.
- `go test -race ./internal/protocol/...`: **could not run** —
  `CGO_ENABLED=1` is required for the race detector and no C compiler is
  configured in this environment. Not run for the rest of the module
  either historically, so this is a pre-existing environment gap, not one
  introduced here — noted as a pending action since this stage adds the
  first genuinely concurrent code (goroutines for the SSH handshake and,
  under the hood, `WatchedStream`'s error paths running from whichever
  goroutine happens to be reading/writing).
- New tests in this stage:
  - `internal/protocol/raw`: 3 tests — connect/echo/idempotent-disconnect,
    missing hostname, refused connection.
  - `internal/protocol/rlogin`: 3 tests — missing username, full handshake
    + echo against a fake `rlogind`, bad acknowledgement byte.
  - `internal/protocol/telnet`: 3 tests — missing hostname, negotiation
    stripped from Read *and* correctly refused (asserted against actual
    bytes a fake telnetd receives), 0xFF escaped on Write.
  - `internal/protocol/ssh`: 4 tests — missing username, SSH-1 rejected,
    full connect/PTY/shell/echo/resize against an in-process SSH-2 server
    (ephemeral ed25519 host key), wrong password rejected.
  - `internal/protocol/serial`: 3 tests, construction/validation only
    (missing port name, valid construction, zero-baud default) — **no
    I/O test**, honestly: exercising `Connect`/`Read`/`Write` needs real
    hardware or a virtual COM port pair (com0com on Windows, socat/pty on
    Linux), neither available here.

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- **SSH host key verification**: currently `ssh.InsecureIgnoreHostKey()`
  — a real MITM exposure. Needs a `known_hosts`-equivalent store plus a
  Phase 3 UI flow to ask the user to confirm a new/changed host key
  (TOFU). Owner: whoever picks up SSH host-key trust, likely bundled with
  Phase 3's credential/security UI work.
- **SSH auth is password-only**: no public-key or keyboard-interactive
  auth; the connection model has no key-file field. Owner: needs a
  connection-model decision (Phase 1 package, so out of this stage's
  scope) before it can be implemented.
- **Serial I/O is untested against real hardware/virtual ports** in this
  environment. Owner: whoever has access to real/virtual serial hardware
  to validate before shipping; construction/validation is covered now.
- **Race detector not run** (`CGO_ENABLED=1` unavailable here) against the
  module's first concurrent code. Owner: run `go test -race ./...` in an
  environment with a C compiler before this ships.
- Commit the working tree — git operations are not taken without explicit
  request (see this session's earlier stage 2.1 audit for the same note).
