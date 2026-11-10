# Audit — Phase 2, Stage 2.4

- **Date (UTC)**: 2026-07-23
- **Agent**: claude-code
- **Audited stage**: VNC
- **Commits covered**: uncommitted at audit time; see the stage 2.4
  implementation and closing-audit changelog fragments created alongside
  this file

## 1. Code quality

- `internal/protocol/framebuffer.go`: new `FramebufferProtocol` interface
  (`Protocol` + `Frames()`/`SendKey`/`SendPointer`), composed rather than
  folded into `Protocol`, mirroring the `TerminalProtocol` precedent from
  stage 2.2 for the same reason — a pixel-pushing session and a
  byte-stream session are different shapes, and window-embedded backends
  (RDP/AnyDesk, later stages) are neither. `PointerButtons` is our own
  bitmask type, not a re-export of the underlying library's `ButtonMask` —
  Phase 3 UI code depends on `internal/protocol`'s vocabulary, not on
  which VNC library a backend happens to use internally.
- `internal/protocol/vnc/vnc.go`: the one substantive design problem
  solved here is disconnect detection. The chosen library
  (`github.com/mitchellh/go-vnc`) has no disconnect callback and never
  closes its `ServerMessageCh` on a dead connection (verified by reading
  its `client.go` source directly, not assumed) — it just silently stops
  sending. A pure "wait for the next message" read loop would hang forever
  on a connection that dies while idle. Fixed with a `keepAliveInterval`
  ticker that re-requests an incremental update periodically; a failed
  write on either path (post-message request or ticker tick) is what
  detects death. Documented inline with the reasoning, not just the code.
- Pixel format is pinned to a fixed 32bpp truecolor format
  (`vncPixelFormat`) explicitly via `SetPixelFormat` right after
  connecting, rather than adapting to whatever the server defaults to.
  This was a deliberate simplification: `RawEncoding.Read` in the library
  already normalizes into `Color{R,G,B uint16}` scaled against the
  server's `RedMax`/`GreenMax`/`BlueMax`, but forcing Max=255 on every
  channel means the `Session` code never has to do that scaling itself —
  one less place to get an off-by-scale bug.
- `applyUpdate` composites incremental rectangles onto a persistent
  `*image.RGBA` (`s.fb`, guarded by `s.fbMu`) and sends a **copy** on
  `Frames()`, not the live buffer — verified by test
  (`TestSession_Connect_ReceivesDecodedFramebuffer` asserts a pixel
  *outside* the received rectangle is still zero-valued, proving the
  backend paints only what the server actually sent and the consumer gets
  an independent snapshot). Costs one full-framebuffer copy per update;
  noted as a possible future optimization (avoid the copy, or diff-only
  delivery) if profiling ever shows it matters — not attempted here since
  there's no evidence yet that it does.
- `Disconnect` idempotency: unlike the stage 2.2 backends, this one
  doesn't use `protocol.WatchedStream` (VNC's "stream" is owned
  internally by the library's `mainLoop`, not something this package reads
  directly), so it has its own `sync.Once` guarding `close(s.done)` +
  `client.Close()`. Verified idempotent by test
  (`TestSession_Disconnect_ClosesFramesChannel` calls `Disconnect` twice).
- No duplication, no function over ~50 lines, no discarded errors.

## 2. Performance

One full-framebuffer `image.RGBA` copy per server-pushed update (see
above) — acceptable for interactive VNC usage (update rates driven by
actual screen changes, not a tight loop) but would need revisiting for
very large/high-frequency virtual displays. Not benchmarked; no evidence
yet that it needs to be.

## 3. Architecture

- New dependency: `github.com/mitchellh/go-vnc` — chosen over a native
  implementation because the blueprint's stage description explicitly
  prefers building on an existing library ("Build on the most complete
  existing Go VNC client library"), unlike stage 2.2 where the blueprint
  asked for native implementations. Verified its actual API via `go doc`
  and by reading the vendored source directly (not from memory) before
  committing to it: `Client()`/`ClientConn` with `PasswordAuth`,
  `RawEncoding`, `KeyEvent`/`PointerEvent`, matches what stage 2.4 needs.
  **Known gap, honestly recorded**: last commit 2015, unmaintained; only
  `RawEncoding` ships (no CopyRect/Hextile/Tight), meaning every update
  transfers full uncompressed pixel data for the changed region — fine
  for LAN/localhost, potentially slow over a WAN link. The blueprint
  explicitly allows "fill gaps... in a vendored fork if needed" for
  exactly this scenario; not attempted in this pass since it wasn't
  blocking any test, and premature optimization without a real slow-link
  test case to justify it would be scope creep.
- Stays inside `internal/protocol/vnc/` plus the additive
  `internal/protocol/framebuffer.go`, consistent with stage 2.1's
  package-ownership model; no other backend package touched.
- `cmd/mremoteng/main.go` blank-imports the new package, same pattern as
  stage 2.2's five backends.
- No impact on closed Phase 1 packages or on the `Protocol`/
  `TerminalProtocol` contracts from stages 2.1/2.2.

## 4. Evidence

- `./scripts/check.sh`: **green**.
- `./scripts/smoke.sh`: **green** (binary builds with `vnc` now
  blank-imported, `mremoteng` starts, changelog reproducible).
- New tests (`internal/protocol/vnc/vnc_test.go`, 3 tests) against a fake
  in-process RFB 3.8 server implementing the real wire protocol (version
  negotiation, no-auth security handshake, ServerInit, and a hand-encoded
  raw-pixel `FramebufferUpdateMessage`) — not just construction/validation
  stubs: `TestSession_Connect_ReceivesDecodedFramebuffer` proves an actual
  server-sent pixel decodes to the correct color at the correct
  coordinate and that untouched pixels stay zero-valued;
  `TestSession_Disconnect_ClosesFramesChannel` proves `OnClose` fires and
  `Frames()` closes on `Disconnect`, twice (idempotency). `SendKey`/
  `SendPointer` are exercised against the live fake server (no assertion
  beyond "doesn't error", since the fake server doesn't decode/echo input
  events back — asserting on outbound VNC wire bytes for input would be
  the next increment if this needs tighter coverage later).
- `go test -race`: still unavailable in this environment (no C compiler,
  same pre-existing gap noted in stage 2.2's audit).

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- **Only RawEncoding is supported** — no CopyRect/Hextile/Tight. Fine for
  LAN use; revisit if WAN/slow-link VNC usage turns out to matter, per
  the blueprint's own allowance for filling encoding gaps later.
- **`github.com/mitchellh/go-vnc` is unmaintained** (2015). It works and
  is well-exercised by this stage's tests, but if a real-world bug
  surfaces against it, a vendored fork or a native reimplementation
  (following stage 2.2's SSH/Telnet precedent) are the fallback options —
  both already anticipated by the blueprint.
- **Race detector still not run** anywhere in this module (environment
  gap, not specific to this stage — repeating the note from stage 2.2's
  audit so it doesn't get lost).
- Commit the working tree — not done without explicit request.
