---
timestamp: 2026-11-11T23:49:25Z
agent: mimocode
files:
  - internal/ui/terminal.go
  - internal/ui/zoom_test.go
---

Keep the terminal zoom when scrolling back without Ctrl

Reported after restoring Ctrl+scroll: zooming worked, but a plain upward scroll reverted the text size and flickered. Cause: widget.TextGrid resolves its size through its own Theme(), which returns the app theme outside a layout pass, so the refresh during scroll re-read the original size. The grid now carries its own zoom theme override: termWidget owns a zoomTheme, wraps every grid it creates in a ThemeOverride, and mutates the theme size on zoom before rebuilding. ctrlHeld became an injectable seam because the X query is not testable and misreported in the test environment; Scrolled uses it, defaulting to controlPressed. TestScrolled now pins Ctrl to false and confirms a plain wheel scrolls the history without touching the size. Validated with ./scripts/check.sh.
