---
timestamp: 2026-11-14T20:49:20Z
agent: mimocode
files:
  - internal/profiles/store.go
  - internal/profiles/profiles_test.go
  - internal/ui/workspace.go
  - internal/ui/shell.go
  - internal/ui/folders.go
  - internal/ui/preferences.go
  - internal/ui/sessions_test.go
  - internal/ui/helpers_test.go
  - internal/ui/screen_test.go
  - internal/ui/actions_test.go
---

Reopen the tabs of the last session at startup

The window forgot its tabs on every restart: the workspace document stored folders and collapsed branches but not which sessions were open. The document now carries open_tabs and active_tab, written whenever a tab opens, closes or is selected, and replayed at startup. Restored tabs stay unconnected on purpose, because no credential is stored: each one waits with Connect enabled, so reopening a workspace never needs a password. Tabs whose connection no longer exists are dropped and the pruned list is written back, quick-connect tabs are never recorded because their profile is not stored, and a window whose list did not change does not write at all. The tab list travels inside the same document as everything else, so a rename rewrites it intact (TestScreen_OtherSaves_KeepTheOpenTabs). TestScreen_RestoreSessions_* cover the restore, the skipped connection and the preference that turns the feature off. Validated with ./scripts/check.sh.
