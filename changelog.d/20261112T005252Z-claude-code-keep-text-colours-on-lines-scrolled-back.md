---
timestamp: 2026-11-12T00:52:52Z
agent: claude-code
files:
  - internal/terminal/terminal.go
  - internal/terminal/terminal_test.go
  - internal/ui/terminal.go
---

Keep text colours on lines scrolled back into the terminal history

The scrollback stored only plain text, so lines scrolled back into history rendered with the default colours. The emulator now keeps a parallel historyCells buffer (trimmed together with history) captured from the same snapshot used for scroll detection, trackScroll takes one snapshot instead of two, and render paints history rows from HistoryCells. Added TestEmulator_ScrolledLines_KeepTheirColours. Validated with gofmt, go vet and go test for internal/ui and internal/terminal.
