---
timestamp: 2026-11-11T23:35:31Z
agent: mimocode
files:
  - internal/ui/session.go
  - internal/ui/shell.go
  - internal/ui/stream.go
  - internal/ui/terminal.go
  - internal/ui/ui_test.go
  - internal/ui/zoom_test.go
---

Force a layout pass when the terminal zoom theme changes

Definitive root cause: container.ThemeOverride applies its theme only inside overrideRenderer.Layout, which pushes it onto the rendering theme stack around the content layout. Mutating the theme (or refreshing the wrapper) never re-laid the content, so the TextGrid kept painting with the previous size even though the emulator refitted. installTerminal now mutates the shared theme and then calls slot.Refresh, which re-runs the slot layout and therefore the override layout with the new size; the wrapper is created once, so the earlier orphan-wrapper revert bug stays fixed. Reverted the grid-owned override experiment and the wheel zoom; zoom remains on Ctrl+= / Ctrl+- / Ctrl+0 and the View menu. Tests cover the two zoom transitions, the shortcut path, and that a plain wheel scrolls without touching the size. Validated with ./scripts/check.sh.
