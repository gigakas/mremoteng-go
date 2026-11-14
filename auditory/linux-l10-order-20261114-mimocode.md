# Audit — Linux track, Stage L10 (manual order of the tree)

- **Date (UTC)**: 2026-11-14
- **Agent**: mimocode
- **Audited stage**: L10 — session UX, the last item the backlog listed: manual ordering
- **Commits covered**: this commit

## 1. Code quality

The tree sorted itself: folders first, then connections, each group by label. The user
can now arrange rows with **Move up / Move down** in the context menu of a connection
or a folder, and the arrangement is stored.

`profiles.Document.Order` is a flat list of **tree node identifiers** (`c:<id>`,
`f:<path>`), which is the smallest shape that expresses an order over both profiles and
folders without adding an order field to every record or turning `Folders` (a
`[]string`) into a struct. `validateOrder` refuses empty keys and control characters;
the store runs it on save.

`treeIndex.sortChildren` gained the ranks: folders before connections as before, then
the stored position, and the label rule only for the nodes the order does not mention.
Keeping folders ahead of connections is deliberate — navigation must not depend on
where a folder happens to sit — so a connection can never jump over a folder, and the
edge message says which group it ran out of
(`TestMoveInOrder_NeverCrossesIntoTheFolderGroup`).

`moveInOrder` (`internal/ui/order.go`) swaps the row with the previous or next sibling
**of its own kind** and writes the whole displayed list back. Swapping two entries is
enough because only the relative order of siblings matters; subtrees in between stay
put. Unknown keys in a stored order are ignored
(`TestManualOrder_UnknownKeysAreIgnored`), and stale keys are dropped for free on the
next move, because the list written back comes from the tree as it stands.

**Filtering.** With a filter active, the displayed tree shows a subset, so writing back
"what is on screen" would forget where the hidden rows were and clearing the filter
would shuffle them to alphabetical order. `orderedKeysForWrite` therefore walks the
**unfiltered** tree when a filter is set, while the move itself targets the previous
*visible* sibling — which is what the user is looking at.
`TestMoveInOrder_WhileFiltered_KeepsTheHiddenNodesInPlace` pins it.

**Menu state.** `orderItem` disables the entry at the edge of its group, so the menu
says what is possible instead of accepting a click that cannot do anything; the status
line still explains an edge that is reached another way (menu built before the tree
changed).

## 2. Performance

A move rebuilds the tree index once and writes one small JSON file. `parentOf` walks the
index instead of threading a parent map through `treeIndex`; the tree is a few hundred
rows, and a second index would be state to keep in sync for no measurable gain.

## 3. Architecture

No new dependency. The store gains one optional list, and a file written before it
loads with the alphabetical rule everywhere — the schema version stays 2. Manual order
composes with the existing features: drag to move a connection between folders keeps its
identifier, so its place in the order still applies inside the new parent, and a rename
does not touch the order at all.

## 4. Evidence

- `./scripts/check.sh`: OK, `./scripts/smoke.sh`: OK.
- New tests: `TestStore_Order_RoundTripsAndIsValidated`,
  `TestMoveInOrder_SwapsWithTheNextConnectionAndPersists`,
  `TestMoveInOrder_NeverCrossesIntoTheFolderGroup`,
  `TestMoveInOrder_LastConnection_ReportsTheEdge`,
  `TestMoveInOrder_Folder_ReordersAmongFolders`,
  `TestMoveInOrder_WhileFiltered_KeepsTheHiddenNodesInPlace`,
  `TestManualOrder_SurvivesARestart`, `TestManualOrder_UnknownKeysAreIgnored`,
  `TestOrderMenu_IsDisabledAtTheEdges`.
- Two context-menu tests were extended, not weakened: the folder menu gained the two
  entries and the connection menu went from four to six.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Human: right-click a connection, **Move up** twice, restart — the row must stay where
  it was put, with folders still ahead of the connections.
- Manual order is per node, not per group: dragging a connection into a folder drops it
  at the position it had among its new siblings rather than at the end. Acceptable, but
  worth revisiting if it surprises the user.
- The tree has no drag-to-reorder; reordering is menu-only for now.
