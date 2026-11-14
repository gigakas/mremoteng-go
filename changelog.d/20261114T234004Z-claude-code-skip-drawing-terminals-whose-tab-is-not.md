---
timestamp: 2026-11-14T23:40:04Z
agent: claude-code
files:
  - internal/ui/terminal.go
  - internal/ui/terminal_test.go
  - internal/ui/shell_tabs.go
  - internal/ui/session.go
  - internal/ui/sessions_test.go
---

Skip drawing terminals whose tab is not selected

Every terminal refreshed its text grid on each chunk of output, including the ones in hidden tabs, which rebuilds canvas objects for a view nobody sees. A terminal is now paused while its tab is not the selected one: the emulator keeps consuming output, nothing is drawn, and the grid catches up once when the tab is shown. A terminal created in a hidden tab starts paused. Measured with four real SSH sessions redrawing at 10 Hz: the hidden three cost little, so the gain is modest there, but it no longer grows with the number of open tabs. The remaining cost is Fyne painting the visible terminal.
