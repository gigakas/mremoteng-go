---
timestamp: 2026-11-11T22:56:49Z
agent: mimocode
files:
  - internal/ui/session.go
  - internal/ui/terminal.go
  - internal/ui/zoom_test.go
---

Rebuild the terminal grid on every zoom step instead of relying on theme refresh

Follow-up to the previous zoom fix, reported as the zoom not working at all. Mutating the shared theme and refreshing the override was not enough in the running window: widget.TextGrid caches the theme text size inside its row renderers and its ApplyTheme is empty, so whether the glyphs resized depended on how far a Refresh propagated. termWidget now keeps its renderer and exposes rebuildGrid, which installs a fresh TextGrid, resizes it and repaints; installTerminal calls it before refreshing the wrapper, so both axes follow the new size deterministically. Converted the shortcut reproduction into TestZoomFocusedTerminal_ShortcutPath_ChangesTextSize, which drives the View menu and Ctrl shortcuts path through the canvas focus and checks zoom in, zoom out and reset; strengthened the existing test to assert the grid is actually replaced. The resize trace now also records the text size, so a future zoom problem can be diagnosed from /tmp/mremoteng-resize.log with MRE_TRACE_RESIZE=1. Validated with ./scripts/check.sh.
