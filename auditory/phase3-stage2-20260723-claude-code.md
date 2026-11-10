# Audit — Phase 3, Stage 3.2

- **Date (UTC)**: 2026-07-23
- **Agent**: claude-code
- **Audited stage**: Connection tree panel
- **Commits covered**: uncommitted at audit time; see the stage 3.2
  implementation and closing-audit changelog fragments created alongside
  this file

## 1. Code quality

- `internal/ui/tree.go`: `ConnectionTree` adapts `connection.ContainerInfo`
  (Phase 1's tree model) to Fyne's callback-based `widget.Tree` — Fyne
  asks for children/branch-ness/labels by string ID on demand rather than
  taking a data structure, so the type's whole job is answering those
  callbacks from an ID→`connection.Node` index built by walking the tree.
  Follows Fyne's own convention of treating `Root=""` as an implicit,
  invisible root whose children are the visible top-level items — our
  model's actual root container (always a synthetic "Connections" folder,
  see `connection.NewRootInfo`) is therefore never itself shown as a row,
  which is the correct behavior, not an oversight (nothing in the
  original app shows a root node either).
- **A real design subtlety, found and fixed via a failing test**: the
  first version of `Reload`'s doc comment claimed the ID index was a
  "snapshot" requiring `Reload` after *any* tree mutation. A test written
  to prove that turned out to fail — `childUIDs` calls `.Children()` live
  on whatever `*connection.ContainerInfo` pointer is already indexed, so
  adding a child under an *already-indexed* container is visible
  immediately with no `Reload` needed. What actually needs `Reload` is a
  container that didn't exist in the tree at index time — until
  reindexed, it's simply absent, so `IsBranch` misreports it as a leaf.
  Rewrote both the test and `Reload`'s doc comment to describe this
  precisely instead of the initially-assumed (and wrong) blanket rule.
- `createNode`/`updateNode` render an icon (folder/folder-open/computer,
  from `theme`) plus a label; `OnBranchOpened`/`OnBranchClosed` call
  `RefreshItem` so the folder icon's open/closed state updates immediately
  rather than waiting for an unrelated refresh.
- `OnSelect` is an exported, nil-by-default callback field — deliberately
  not wired to anything in this stage, since its real consumers (open a
  session tab, stage 3.3; show the properties panel, stage 3.4) don't
  exist yet. Documented as such rather than left unexplained.
- No duplication, no function over ~50 lines, no discarded errors.

## 2. Performance

Not applicable at this scale: `Reload` walks the whole tree, which is
fine for the connection-list sizes this application targets (this is
exactly what the original C# app's own in-memory tree does too, no
lazier scheme needed).

## 3. Architecture

- Stays inside `internal/ui/`, the package this phase owns; imports
  `internal/connection` (Phase 1, closed) but nothing else — no new
  dependencies.
- `cmd/mremoteng/main.go` now creates a real (currently empty)
  `connection.ContainerInfo` root and wires it into the shell via
  `shell.SetTree(ui.NewConnectionTree(root).Widget)`, replacing the
  placeholder label from stage 3.1. Loading an actual `.xml` connections
  file into this tree — not yet wired, no "open file" flow exists — is
  what will satisfy Phase 2/3's shared "demo config file" exit criterion
  once stage 3.5 (persistence) or an earlier ad-hoc load path lands.
- No impact on `internal/protocol` or already-closed Phase 1/2 packages.

## 4. Evidence — same visual-verification limitation as stage 3.1

- `./scripts/check.sh`: **green**.
- `./scripts/smoke.sh`: **green** (binary still starts and stays alive
  with the real tree wired in).
- **No visual verification was possible or attempted**, for the same
  reason recorded in stage 3.1's audit and now in
  `blueprint/phase-3-ui.md`'s phase-wide note: this development
  environment cannot screenshot windows it launches. Icon choice, spacing,
  and general tree appearance are unverified beyond "Fyne's headless
  renderer builds this widget tree without erroring."
- New tests (`internal/ui/tree_test.go`, 5 tests, package `ui_test` using
  only `ConnectionTree`'s public API — `Widget.ChildUIDs`/`IsBranch`/
  `CreateNode`/`UpdateNode`/`OnSelected` are all exported struct fields,
  so no white-box access was needed here unlike `shell_test.go`):
  `ChildUIDs` reflects the model, `IsBranch` distinguishes containers from
  connections, `UpdateNode` sets the label from `Effective().Name` (using
  the inheritance-resolved name from Phase 1.2, not the raw one — worth
  calling out since it's an easy field to get wrong), `OnSelect` fires
  with the right node, and `Reload` correctly indexes a newly-added,
  previously-unreachable container (the scenario described in section 1).

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- **No visual confirmation this looks acceptable** — repeating the
  phase-wide note; applies to every 3.x UI stage.
- **No context menu / drag-drop / rename-in-place yet** — this stage only
  covers *displaying* the tree and reporting selection; editing operations
  (new connection, new folder, delete, move, rename) are natural
  candidates for stage 3.4's properties panel or a follow-up, not
  attempted here since the blueprint scopes 3.2 as "Connection tree
  panel," not "connection tree editing."
- **`OnSelect` has no consumer yet** — by design (see section 1), but
  worth remembering it needs wiring once 3.3/3.4 exist.
- Commit the working tree — not done without explicit request.
