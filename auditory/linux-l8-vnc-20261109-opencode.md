# Audit — Linux track, stage L8 (part 1: VNC)

- **Date (UTC)**: 2026-11-09
- **Agent**: opencode
- **Audited stage**: L8 partial — native VNC client and session view
- **Scope**: `internal/protocol/vnc`, `internal/ui/vnc.go`, VNC protocol in the model

## 1. Code quality

- The RFB client is dependency-free: handshake (3.8), security negotiation, VNC
  DES challenge-response (with the required bit-reversed key schedule), server
  init, SetPixelFormat to 32-bit true colour and Raw rectangle decoding.
- Unsupported security types and encodings return explicit errors instead of
  hanging; a zero security-type list reads the failure reason.
- The framebuffer is one reused `image.RGBA`; updates copy into it in place, so
  the canvas only needs a Refresh.
- `vncView` maps click coordinates from the scaled widget to framebuffer
  coordinates and sends press+release. Keyboard forwarding is implemented in the
  client (`Key`) but not wired to a focusable widget yet.
- Panel wiring is injectable (`vncDialer`, `vncSession`), which is what the UI
  tests exercise.

## 2. Performance

- Raw encoding only: bandwidth-heavy but correct; ZRLE/Tight are the next
  optimisation. Pixel copy is per rectangle, O(w·h).

## 3. Architecture

- VNC is native Go, so it needs no external process and no cgo, matching the
  protocol policy. `profiles.Protocol` gained `VNC` with port 5900 and its own
  tree badge.
- The client is unaware of Fyne; the UI owns rendering and coordinate mapping.

## 4. Evidence

- `./scripts/check.sh`: green.
- New tests: fake RFB server covering None security, VNC auth, framebuffer decode
  (red/green pixels), pointer and key forwarding, unsupported security type,
  unreachable host, bit reversal; UI tests for connect/render/click mapping and
  disconnect, VNC validation and badge.
- Not yet verified against a real VNC server on this machine.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Real-server check (x11vnc or a container) and a manual pixel/colour sanity test.
- Keyboard input for VNC (Fyne focusable widget mapping keysyms) and clipboard.
- Remaining L8 items: HTTP/HTTPS webview, Telnet/RAW/rlogin on the terminal
  widget, serial.
