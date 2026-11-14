---
timestamp: 2026-11-14T20:49:47Z
agent: mimocode
files:
  - internal/settings/settings.go
  - internal/settings/settings_test.go
---

Delete the settings package nothing used

internal/settings described a plain-JSON preference file with window geometry, a theme choice, a last-connections-file field and an unimplemented enterprise policy path, and no code imported it: window geometry already lives in Fyne's preference store, and the theme choice and new-connection defaults landed there in the settings dialog. Keeping a second, unwired mechanism for the same data is how two sources of truth start, so the package and its tests are gone; the decision it documented (settings in a plain config file, no Windows registry) still holds, because Fyne's store is one. Validated with ./scripts/check.sh.
