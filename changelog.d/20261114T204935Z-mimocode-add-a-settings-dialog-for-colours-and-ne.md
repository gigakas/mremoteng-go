---
timestamp: 2026-11-14T20:49:35Z
agent: mimocode
files:
  - internal/ui/preferences.go
  - internal/ui/settings_dialog.go
  - internal/ui/settings_dialog_test.go
  - internal/ui/theme.go
  - internal/ui/shell.go
  - internal/ui/actions.go
  - cmd/mremoteng/main.go
---

Add a settings dialog for colours and new-connection defaults

There was nowhere to choose how the window looks or how a new connection starts: the theme followed the desktop and the Config panel always began with SSH on port 22. File > Settings now edits three preferences: the colour variant (system, light or dark), the protocol and port a new connection starts with, and whether the tabs of the last session come back. The variant is pinned inside the theme, because Fyne takes light or dark from the desktop and offers no setting for it; the startup path reads the same preference so the first paint already matches, and Apply repaints immediately. Values are validated before anything is stored, and a rejected dialog leaves the previous settings untouched. TestSoberThemeVariant_PinsTheColours, TestApplySettings_StoresAndValidatesTheValues and TestNewConnectionProfile_UsesTheStoredDefaults cover the choices; TestParseSettingsPort_EmptyMeansTheProtocolDefault pins an empty port to the protocol's own default. Validated with ./scripts/check.sh.
