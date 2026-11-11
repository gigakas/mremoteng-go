---
timestamp: 2026-11-11T23:25:43Z
agent: mimocode
files:
  - internal/ui/session.go
  - internal/ui/shell.go
  - internal/ui/stream.go
  - internal/ui/terminal.go
  - internal/ui/ui_test.go
  - internal/ui/zoom.go
  - internal/ui/zoom_test.go
---

Give the terminal text grid its own theme override so zoom changes the glyphs

Zoom kept refitting the emulator without changing the glyph size in the running window: /tmp/mremoteng-resize.log showed cols and rows following the new text size while the text stayed put. The panel-level override was not reaching the grid. Fyne caches the theme a widget resolves (BaseWidget.themeCache, filled from cache.WidgetTheme) and widget.TextGrid recalculates its cached row height only when its own renderer refreshes, reading the TextGrid widget theme. The override now lives with the grid: termWidget builds the TextGrid and a zoomTheme override around it in installGrid, keeps both in gridTheme and gridWrap, and rebuildGrid replaces the pair. adjustZoom mutates the theme size, rebuilds the grid and refits, so the size no longer depends on inheritance from the tab. The panel no longer wraps the terminal and loses terminalTheme, terminalWrap, forgetTerminalOverride and the onZoom hook; resetFocusedTerminalZoom goes through adjustZoom. Added traceZoom behind MRE_TRACE_ZOOM, which is how this was diagnosed. Updated the tests to the new invariant. Validated with ./scripts/check.sh.
