---
timestamp: 2026-07-23T17:28:19Z
agent: claude-code
files:
  - auditory/phase2-stage4-20260723-claude-code.md
  - blueprint/phase-2-protocols.md
  - cmd/mremoteng/main.go
  - go.mod
  - go.sum
  - internal/protocol/framebuffer.go
  - internal/protocol/vnc/vnc.go
  - internal/protocol/vnc/vnc_test.go
---

Phase 2 stage 2.4: add the VNC backend

Implemented VNC on github.com/mitchellh/go-vnc, per the blueprint's explicit preference for building on an existing library (unlike stage 2.2's native-implementation instruction). Verified the library's actual API via go doc and by reading its vendored source directly before committing to it -- notably that its mainLoop never closes ServerMessageCh on a dead connection, which shaped the backend's disconnect-detection design (a periodic keepalive re-request whose write failure is what surfaces death, since there's no other signal). Added protocol.FramebufferProtocol (Frames()/SendKey/SendPointer), composed with Protocol like stage 2.2's TerminalProtocol, since VNC's session shape (pixels in, key/pointer events out) is neither a byte stream nor a window-embedded external process. Pixel format is pinned to a fixed 32bpp truecolor format right after connecting so decoding never needs to scale against a server-chosen RedMax/GreenMax/BlueMax. applyUpdate composites incremental rectangles onto a persistent framebuffer and emits a copy on Frames(), verified by test that untouched pixels stay zero and only the server-sent rectangle is painted. Only RawEncoding is used (library ships nothing else); CopyRect/Hextile/Tight are v2 backlog per the blueprint's own allowance. Tests run against a fake in-process RFB 3.8 server implementing the real wire handshake and a hand-encoded FramebufferUpdateMessage, proving actual pixel decode correctness and OnClose/Frames-channel-close on Disconnect (including idempotency), not just construction stubs. check.sh and smoke.sh green; go test -race still unavailable (no C compiler in this environment, same pre-existing gap as stage 2.2).
