---
timestamp: 2026-11-11T23:08:25Z
agent: mimocode
files:
  - blueprint/linux-preview.md
  - internal/ui/shell.go
  - internal/ui/terminal.go
  - internal/ui/zoom.go
  - internal/ui/zoom_test.go
---

Zoom the terminal with Ctrl+scroll again, throttled

Wheel zoom had been removed because it rebuilt the text grid on every scroll event, which flickered and reset the size. Now that a zoom step rebuilds the grid deterministically it can come back, with the two things the old version lacked. Fyne ScrollEvent carries no modifier information, so ctrlHeld asks the X server through QueryPointer, but with a connection opened once and reused instead of one per event; without DISPLAY it reports false and the discrete shortcuts stay the portable path. zoomByScroll also throttles steps to one per 120 ms so a spinning wheel does not repaint the grid on every notch. The Ctrl state is behind an injectable seam so it can be tested headlessly, and TestScrolled_CtrlHeld_ZoomsInsteadOfScrolling checks that Ctrl+scroll zooms without moving the view, that notches inside the throttle window do not stack, and that a released Ctrl returns the wheel to the scrollback. Updated the Linux preview notes, which claimed Ctrl+scroll deliberately scrolled instead of zooming. Validated with ./scripts/check.sh.
