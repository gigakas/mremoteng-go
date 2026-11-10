---
timestamp: 2026-07-23T04:57:37Z
agent: claude-code
files:
  - auditory/phase2-stage2-20260723-claude-code.md
  - blueprint/phase-2-protocols.md
  - cmd/mremoteng/main.go
  - go.mod
  - go.sum
  - internal/protocol/lifecycle.go
  - internal/protocol/protocol.go
  - internal/protocol/raw/raw.go
  - internal/protocol/raw/raw_test.go
  - internal/protocol/rlogin/rlogin.go
  - internal/protocol/rlogin/rlogin_test.go
  - internal/protocol/serial/serial.go
  - internal/protocol/serial/serial_test.go
  - internal/protocol/ssh/ssh.go
  - internal/protocol/ssh/ssh_test.go
  - internal/protocol/telnet/telnet.go
  - internal/protocol/telnet/telnet_test.go
  - internal/protocol/terminal.go
---

Phase 2 stage 2.2: add SSH, Telnet, rlogin, raw socket and serial backends

Implemented the five terminal-family protocols. Added protocol.TerminalProtocol (Protocol + io.ReadWriter) since stage 2.1's Protocol interface had no way to expose a byte stream for these -- window-embedded backends (RDP/VNC/AnyDesk, later stages) don't need one, so it's a separate composed interface rather than added to Protocol itself. Added shared internal/protocol helpers (Lifecycle for OnError/OnClose bookkeeping, WatchedStream wrapping an io.ReadWriteCloser so I/O errors auto-fire those callbacks) before writing five near-identical backends, not after -- and while writing WatchedStream, found and fixed a real idempotency bug (its Close() was the promoted net.Conn.Close, which errors on a second call, breaking every backend's Disconnect contract) caught by the raw backend's test before it could spread to the other four. Amended Protocol.Resize's doc comment: unit is now backend-defined (pixels for window-embedded, character cells for terminal) instead of unconditionally pixels, since SSH's PTY resize needed cell semantics. internal/protocol/raw: plain TCP passthrough. internal/protocol/rlogin: RFC 1282 handshake (four NUL-delimited fields + one ack byte) then transparent passthrough. internal/protocol/telnet: a minimal RFC 854 NVT client that strips IAC negotiation from Read and replies WONT/DONT to every DO/WILL (refuses every option rather than implementing ECHO/SGA/NAWS/terminal-type), escapes literal 0xFF on Write. internal/protocol/ssh: SSH-2 via golang.org/x/crypto/ssh, password auth, PTY + shell, WindowChange-backed Resize; SSH-1 is registered but its constructor always returns a clear 'not supported' error instead of a generic 'no backend registered' one. internal/protocol/serial: go.bug.st/serial (new dependency -- stdlib has no cross-platform serial support; this is the standard actively-maintained pure-Go option), reusing the connection model's Hostname/Port fields as port-name/baud-rate exactly like the original C# app's PuttyBase does for its -serial mode (confirmed by reading ../mRemoteNG's Connection/Protocol/Serial/Connection.Protocol.Serial.cs and PuttyBase.cs -- no connection-model changes needed). go mod tidy bumped go.mod's go directive 1.23 -> 1.25.0 as a side effect of the new dependency. Wired all five into cmd/mremoteng via blank imports, resolving stage 2.1's pending action. Tests: local in-process servers per the blueprint's own instruction -- raw/rlogin/telnet against fake TCP servers, ssh against a real in-process SSH-2 server (ephemeral ed25519 host key, actual handshake/PTY/shell/echo/resize); serial only has construction/validation tests, honestly, since real or virtual COM hardware isn't available here. check.sh and smoke.sh green; go test -race could not run (no C compiler configured in this environment for CGO_ENABLED=1) -- flagged in the audit as a pending action given this is the module's first genuinely concurrent code.
