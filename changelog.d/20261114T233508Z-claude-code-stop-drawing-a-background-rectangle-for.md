---
timestamp: 2026-11-14T23:35:08Z
agent: claude-code
files:
  - internal/ui/terminal.go
  - internal/ui/terminal_test.go
---

Stop drawing a background rectangle for every default-background terminal cell

Profiling the app with four live SSH sessions showed over 90% of CPU in Fyne's GL painter: every terminal cell carried a background colour, so each repaint issued one drawRectangle and glBufferData per cell (1920 on an 80x24 screen). Cells on the default background now carry no background, because the window already paints it. Measured with a throwaway harness that opened four real SSH sessions against an in-process server: a full-screen redraw cost about 7 ms of CPU before and about half that after, and four idle sessions use under 1% of a core.
