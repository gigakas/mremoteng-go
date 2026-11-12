---
timestamp: 2026-11-12T01:25:48Z
agent: mimocode
files:
  - internal/ui/workspace_test.go
---

Cover the workspace persistence rules without a window

Phase C and the close of the screen extraction. The persistence rules were only reachable through a screen driven by Fyne's test driver, so the invariants this extraction depends on were untested in isolation: what happens when the store fails to load, whether a rejected write really leaves the state untouched, how a rename rewrites a subtree, or what replace mode does to duplicate names. Added workspace_test.go, which imports no Fyne package and drives the workspace directly with the existing fakeStore: thirteen cases covering a failed load leaving the workspace read-only and empty, a successful load rebuilding the collapsed set, a rejected write keeping the previous profiles, putProfile adding its parent folders, deleteProfile, repeated addFolder, defaults stored and cleared, renameFolder rewriting the whole subtree while leaving unrelated connections alone, deleteFolder dropping contained connections but keeping siblings, the difference between renameProfile and putConnection, import merge counting one added and one skipped, import replace keeping everything incoming after dropping the old data, and reset re-enabling writes after a failed load. All of them pass; the package now has 144 tests, up from 131. Validated with ./scripts/check.sh.
