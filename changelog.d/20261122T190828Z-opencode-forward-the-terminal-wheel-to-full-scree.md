---
timestamp: 2026-11-22T19:08:28Z
agent: opencode
files:
  - internal/terminal/terminal.go
  - internal/terminal/terminal_test.go
  - internal/ui/terminal_input.go
  - internal/ui/terminal_test.go
  - README.md
---

Forward the terminal wheel to full-screen programs

SSH sessions running TUI clients (opencode, Claude Code, editors, multiplexers) ignored the mouse wheel: termWidget.Scrolled always consumed it for the local scrollback, and the emulator never told the widget that the program had asked for mouse reporting. internal/terminal now exposes Modes() with the mouse reporting, SGR, alternate-screen and application-cursor flags that DECSET/DECRST set, and stops recording scrollback while the alternate screen is active, where a program's redraws are its own interface rather than shell output. termWidget.Scrolled now sends one wheel event per notch to the program when mouse reporting is on (SGR buttons 64/65, or the legacy X10 buttons 4/5 at the pointer cell), turns the wheel into arrow keys on the alternate screen without mouse reporting, and keeps the previous behaviour otherwise. Fyne reports one wheel click as 25 units on Linux, so the conversion divides by that instead of sending one event per unit; a fractional touchpad delta still counts as one notch and a single event is capped at ten. Verified against a local TCP server that enables SGR reporting: each wheel click now arrives as one \x1b[<64;x;yM or \x1b[<65;x;yM.
