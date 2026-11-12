---
timestamp: 2026-11-12T00:55:36Z
agent: claude-code
files:
  - internal/terminal/terminal.go
  - internal/terminal/terminal_test.go
  - internal/ui/terminal.go
---

Drop the terminal scrollback when the remote sends ESC[3J (clear)

After clearing the screen the old output stayed in the scrollback and reappeared when scrolling or drag-selecting past the top edge. The emulator now empties history and historyCells when a write contains ESC[3J, which is what clear and recent readline send, and the widget clamps scrollOffset and drops the selection if the history shrank. Added TestEmulator_EraseScrollback_DropsHistory. A sequence split across two writes is not detected. Validated with gofmt, go vet and go test for internal/ui and internal/terminal.
