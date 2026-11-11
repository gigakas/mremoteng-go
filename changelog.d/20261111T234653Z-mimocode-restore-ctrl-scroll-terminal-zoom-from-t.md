---
timestamp: 2026-11-11T23:46:53Z
agent: mimocode
files:
  - internal/ui/terminal.go
  - internal/ui/zoom.go
---

Restore Ctrl+scroll terminal zoom from the last build that worked

The discrete Ctrl+= / Ctrl+- shortcuts never resized the glyphs: the commit that introduced them removed the fit and render calls from adjustZoom, so the grid refit stopped happening. Both are restored now, together with the Ctrl+scroll wheel path the user asked for and confirmed working earlier: Fyne scroll events carry no modifiers, so controlPressed asks the X server, and a Ctrl+wheel notch zooms the console by one step while a plain wheel keeps scrolling the history. The known caveat remains that the text size can revert when output arrives or the view scrolls, tracked separately. Validated with ./scripts/check.sh.
