---
timestamp: 2026-11-11T22:53:15Z
agent: mimocode
files:
  - internal/ui/session.go
  - internal/ui/zoom_test.go
---

Keep the terminal zoom when output arrives or the view scrolls

Reported bug: after zooming a session terminal the text fell back to its previous size as soon as remote output arrived or the view scrolled. Root cause is how Fyne handles theme overrides: container.NewThemeOverride registers the theme in a per-object cache, and ThemeOverride.Refresh re-registers its own theme. installTerminal created a NEW override on every zoom step, so the previous wrapper stayed alive holding the old text size, and any refresh that reached that orphan put the old size back into the cache. internal/ui/session.go now keeps one override per panel in terminalTheme and terminalWrap and mutates the theme size in place, then refreshes the wrapper so the TextGrid re-reads the size it caches, recomputes the grid through fit and repaints. Added TestInstallTerminal_ZoomThenActivity_KeepsTheTextSize, which was verified to fail against the previous re-wrapping behaviour. Validated with ./scripts/check.sh.
