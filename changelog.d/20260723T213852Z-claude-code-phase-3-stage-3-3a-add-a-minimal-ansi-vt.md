---
timestamp: 2026-07-23T21:38:52Z
agent: claude-code
files:
  - auditory/phase3-stage3-20260723-claude-code.md
  - blueprint/phase-3-ui.md
  - cmd/mremoteng/main.go
  - internal/protocol/winembed/winembed.go
  - internal/ui/ansi.go
  - internal/ui/ansi_test.go
  - internal/ui/framebuffer.go
  - internal/ui/framebuffer_test.go
  - internal/ui/nativewindow.go
  - internal/ui/nativewindow_other.go
  - internal/ui/nativewindow_test.go
  - internal/ui/nativewindow_windows.go
  - internal/ui/sessiontabs.go
  - internal/ui/sessiontabs_test.go
  - internal/ui/terminal.go
  - internal/ui/terminal_test.go
---

Phase 3 stage 3.3a: add a minimal ANSI/VT100 terminal widget

Added ansiState (internal/ui/ansi.go), a scoped ANSI/VT100 byte-stream interpreter over Fyne's widget.TextGrid (whose own doc comment names it as intended for terminal emulator use). Covers cursor movement (CUU/CUD/CUF/CUB/CUP), erase in display/line, basic SGR (reset, bold, 16-color fg/bg), CR/LF/backspace/tab, and safely discards OSC payloads instead of leaking them into the display. Deliberately does not attempt 256-color/truecolor, alternate screen, mouse reporting, or scroll regions -- the blueprint flags a full terminal emulator as stage 2.2/3.3's 'real cost driver' needing separate estimation before attempting one; this is a bounded v1 subset chosen over both a full xterm clone (too large) and the documented PuTTY-external-process fallback (which would leave stage 2.2's tested native Go SSH/Telnet/rlogin/raw/serial implementations unused for the UI path). Terminal (internal/ui/terminal.go) wraps ansiState as a focusable, typeable Fyne widget: Write feeds protocol output in, TypedRune/TypedKey relay keyboard input out via an OnInput callback, with a small key-to-control-sequence map for Enter/Backspace/Tab/Escape/arrows. 17 new tests (12 in ansi_test.go feeding exact byte sequences and asserting exact grid-cell contents -- pure logic, the strongest test coverage in this whole UI phase since no rendering is involved; 5 in terminal_test.go for the widget wiring).
