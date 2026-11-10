# Audit — Linux track, stage L8 (part 2: Telnet/RAW/rlogin and HTTP)

- **Date (UTC)**: 2026-11-09
- **Agent**: opencode
- **Audited stage**: L8 partial — byte-stream protocols and HTTP handoff
- **Scope**: `internal/protocol/telnet`, `internal/ui/stream.go`, protocol list

## 1. Code quality

- `internal/protocol/telnet` implements Telnet (IAC negotiation, escaped 0xFF),
  RAW TCP and the rlogin RFC 1282 preamble in one connection type, so the UI has
  a single path for the three modes.
- Negotiation declines every option (`WONT`/`DONT`); that is sufficient for
  line-oriented servers and avoids partial option state. Unknown commands are
  skipped rather than emitted as garbage.
- The terminal view is reused unchanged, so keys, Tab and Ctrl-C/D buttons work;
  the panel states clearly that there is no remote pty, so full-screen programs
  may misrender.
- HTTP keeps its promise: it opens the address in the default browser and says
  so. A real webview needs a GTK window parented into the Fyne X11 window; webkit
  is installed (2.52.6) but that work is not done, and the code does not pretend
  otherwise.

## 2. Performance

- Byte copy per read/write; no buffering beyond the 32 KiB read loop.

## 3. Architecture

- No new dependencies: streams use `net` and the terminal widget; HTTP uses
  `os/exec` with `xdg-open`.
- Protocol validation now accepts TELNET, RAW, RLOGIN and HTTP with their default
  ports, and the tree badges distinguish terminal (T) and web (W) sessions.

## 4. Evidence

- `./scripts/check.sh`: green.
- New tests: Telnet negotiation stripping, escaped IAC, rlogin handshake content,
  RAW byte pass-through, mode parsing; UI tests for terminal install and typed
  input over a fake stream, protocol→mode mapping, and HTTP URL/browser handoff.
- No real Telnet/rlogin server was available for a manual session.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Real-server checks (Telnet router/emulator, `rlogind`).
- Optional: embedded webview via GTK window parenting; serial ports.
- These, plus the L9/L10/L11/L12 items, are the remaining backlog.
