---
timestamp: 2026-11-11T23:55:23Z
agent: mimocode
files:
  - internal/ui/session.go
  - internal/ui/shell.go
  - internal/ui/terminal.go
  - internal/ui/ui_test.go
---

Own the terminal text size in the grid so refreshes cannot revert it

The screenshot and report confirm the exact symptom: after zooming, typing, scrolling without Ctrl or selecting text made the glyphs fall back to the original size while the row height stayed zoomed, producing text floating in tall rows. Cause: the TextGrid resolves its glyph size through its own Theme(), which returns the app theme whenever it is refreshed outside a layout pass; the row height was fixed earlier during layout. The terminal widget now owns its zoom theme and wraps every text grid it creates in that theme override, so grid.Theme() returns the terminal size on every refresh, not the app theme. This replaces the panel-level wrapping, removing the double override; installTerminal now just places the terminal in the slot, and zoom mutates the theme then rebuilds the grid and refits. Ctrl detection is an injectable seam. Validated with ./scripts/check.sh.
