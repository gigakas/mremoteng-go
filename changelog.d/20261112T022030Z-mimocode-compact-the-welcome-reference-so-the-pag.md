---
timestamp: 2026-11-12T02:20:30Z
agent: mimocode
files:
  - internal/ui/properties.go
  - internal/ui/welcome_test.go
---

Compact the welcome reference so the page fits without scrolling

The button reference made the start page taller than the window, so it opened with a scrollbar and the getting-started steps were pushed out of sight. Three changes: each reference row now puts the bold name and its explanation on the same line instead of stacking them, the rows are laid out in two columns, and every text on the page was shortened, including the getting started steps, the intro and the session notice. The reference still carries one row per button with its icon. TestWelcomeHelp_DocumentsEveryIconButton now rejects an explanation longer than 60 characters, because a long line is exactly what wraps and starts the scrolling again. Validated with ./scripts/check.sh.
