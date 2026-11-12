---
timestamp: 2026-11-12T00:50:16Z
agent: claude-code
files:
  - internal/terminal/terminal.go
  - internal/ui/terminal.go
  - internal/ui/ui_test.go
---

Refresh the terminal grid in place on zoom instead of rebuilding the widget

Each zoom step tore down and recreated the TextGrid and its theme override, on the belief that TextGrid caches its row height. In fyne v2.8.2 textGridContentRenderer.Refresh re-measures the cell size from the theme on every refresh (verified with a throwaway test: changing the override size and calling grid.Refresh alone scaled MinSize), so rebuildGrid now refreshes the existing grid and only builds one when none exists. It also refreshes the widget itself, because without that a keyboard zoom stayed invisible until a focus change forced a repaint. adjustZoom and termRenderer.Layout dropped a redundant trailing render, and adjustZoom keeps the renderer font bookkeeping in step. Separately, render no longer takes a second full Snapshot per call: viewStart is now historyLen minus scrollOffset and history rows come from the new Emulator.History accessor, since on-screen runes already come from the snapshot. The zoom test was renamed and now asserts the grid instance is reused, going through test.WidgetRenderer so the renderer cache matches the real driver. Measured headless, a zoom step costs about 8 ms of Go work, so the remaining stall the user reported is on the real GPU painter and is NOT yet confirmed fixed on screen; the user also reports the mouse wheel does not move the view, which is still open. Validated with gofmt, go vet and go test for internal/ui and internal/terminal; full check.sh not run.
