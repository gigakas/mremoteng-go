---
timestamp: 2026-11-11T23:31:26Z
agent: mimocode
files:
  - blueprint/linux-preview.md
  - internal/ui/shell.go
  - internal/ui/terminal.go
  - internal/ui/zoom.go
  - internal/ui/zoom_test.go
---

Remove wheel zoom; it hijacked the scrollback on Wayland

Reported after the Ctrl+scroll change: the wheel did not scroll and the console froze. The zoom trace showed text size flipping constantly on wheel input, which means ctrlHeld misreported Ctrl as held on Wayland/XWayland, so every notch went to zoomByScroll instead of the scrollback and each step rebuilt the grid. Fyne ScrollEvent carries no modifier information and the X QueryPointer fallback is not trustworthy on that setup, so the wheel path is removed: the wheel is always scrollback again, and zoom stays on Ctrl+=, Ctrl+-, Ctrl+0 and the View menu, which do not depend on modifier state. The grid-owned theme override from the previous change is kept. Replaced TestScrolled_CtrlHeld_ZoomsInsteadOfScrolling with TestScrolled_MovesThroughScrollbackWithoutChangingZoom. Updated the Linux preview notes. Validated with ./scripts/check.sh.
