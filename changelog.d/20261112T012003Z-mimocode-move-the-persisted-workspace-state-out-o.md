---
timestamp: 2026-11-12T01:20:03Z
agent: mimocode
files:
  - internal/ui/actions.go
  - internal/ui/actions_test.go
  - internal/ui/diagnostics.go
  - internal/ui/folders.go
  - internal/ui/menu.go
  - internal/ui/repair.go
  - internal/ui/screen_test.go
  - internal/ui/shell.go
  - internal/ui/transfer.go
  - internal/ui/tree_test.go
  - internal/ui/workspace.go
---

Move the persisted workspace state out of the screen into its own type

Phase A of the screen extraction, a pure move with no behaviour change. The screen kept six fields that belong to the saved document (store, all, folders, defaults, collapsed, writable) and three methods that built or updated it (document, documentWithDefaults, upsert), mixed in with the window, the tree index and the open tabs. internal/ui/workspace.go now owns exactly that state and those helpers, plus load and save, and it imports no Fyne package, so the persistence rules can be tested without a window. The screen keeps a single ws field and every call site now reads s.ws.profiles, s.ws.folders, s.ws.defaults, s.ws.collapsed, s.ws.writable or s.ws.store. reload keeps doing the widget work and delegates the data part to ws.load; persistWorkspace delegates to ws.save. Nothing else changed: the save-before-assign order in every mutator is untouched, a failed load still leaves the workspace read-only with the same message, and branchChanged still only touches memory and persists on close. screen drops from 114 to 111 methods and from 49 to 45 fields. All 125 tests pass unchanged apart from the field spelling in three test files. Validated with ./scripts/check.sh.
