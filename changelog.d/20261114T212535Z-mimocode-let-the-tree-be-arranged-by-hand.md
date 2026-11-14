---
timestamp: 2026-11-14T21:25:35Z
agent: mimocode
files:
  - internal/profiles/store.go
  - internal/profiles/profiles_test.go
  - internal/ui/order.go
  - internal/ui/order_test.go
  - internal/ui/tree.go
  - internal/ui/menu.go
  - internal/ui/actions.go
  - internal/ui/workspace.go
  - internal/ui/tree_test.go
  - internal/ui/helpers_test.go
  - internal/ui/screen_test.go
  - auditory/linux-l10-order-20261114-mimocode.md
  - blueprint/backlog.md
  - blueprint/linux-preview.md
---

Let the tree be arranged by hand

The tree always sorted itself: folders first, then connections, each group by label, so a user who wanted their busiest connection at the top could not have it. Move up and Move down in the context menu of a connection or a folder now arrange the row, and the arrangement is stored as a flat list of tree node identifiers, the smallest shape that orders both profiles and folders without adding a field to every record. Folders stay ahead of connections whatever the arrangement, so navigation never depends on where a folder sits, and the menu entry is disabled at the edge of its group instead of accepting a click that cannot act. Unknown keys in a stored order are ignored and stale ones drop out on the next move. Reordering while a filter hides rows writes the unfiltered order back, so clearing the filter does not shuffle the hidden rows to alphabetical order (TestMoveInOrder_WhileFiltered_KeepsTheHiddenNodesInPlace). Validated with ./scripts/check.sh and ./scripts/smoke.sh.
