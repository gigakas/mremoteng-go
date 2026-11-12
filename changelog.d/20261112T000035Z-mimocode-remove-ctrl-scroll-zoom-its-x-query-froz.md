---
timestamp: 2026-11-12T00:00:35Z
agent: mimocode
files:
  - internal/ui/shell.go
  - internal/ui/terminal.go
  - internal/ui/zoom.go
---

Remove Ctrl+scroll zoom; its X query froze the console on Wayland

Freezing plus the need to defocus the console while scrolling pointed at controlPressed: it asks the X server whether Ctrl is held on every wheel event, but on Wayland/XWayland it misreports, so a plain scroll was treated as zoom and every notch rebuilt the whole text grid. Fyne scroll events carry no modifier information, so there is no reliable way to distinguish Ctrl+scroll here. The wheel is now always scrollback again, and zoom stays on the discrete shortcuts Ctrl+=, Ctrl+-, Ctrl+0 and the View menu, which need no modifier detection. The grid-owned text size from the previous change is kept, so a zoom done via the keys persists across typing, scroll and selection. Removed the controlPressed X query and its imports. Validated with ./scripts/check.sh.
