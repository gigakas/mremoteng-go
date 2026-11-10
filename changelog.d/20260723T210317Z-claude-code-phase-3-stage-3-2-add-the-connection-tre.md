---
timestamp: 2026-07-23T21:03:17Z
agent: claude-code
files:
  - auditory/phase3-stage2-20260723-claude-code.md
  - blueprint/phase-3-ui.md
  - cmd/mremoteng/main.go
  - internal/ui/tree.go
  - internal/ui/tree_test.go
---

Phase 3 stage 3.2: add the connection tree panel

Implemented ConnectionTree, adapting connection.ContainerInfo (Phase 1's tree model) to Fyne's callback-based widget.Tree via an ID-to-Node index built by walking the tree. Follows Fyne's Root="" implicit-invisible-root convention, so the model's synthetic root container is never itself shown as a row. Icons (folder/folder-open/computer) differentiate branches from leaf connections; OnBranchOpened/OnBranchClosed call RefreshItem so the folder icon reflects open/closed state immediately. OnSelect is an exported, nil-by-default callback -- deliberately not wired to anything yet, since its real consumers (open a tab in 3.3, show properties in 3.4) don't exist. A test-driven correction: Reload's first doc comment claimed the ID index was a blanket snapshot needing a reload after any mutation; a test written to prove that failed, because childUIDs calls Children() live on already-indexed container pointers -- what actually needs Reload is a container that didn't exist in the tree at index time at all. Rewrote both the test and the doc comment to describe this precisely. cmd/mremoteng/main.go now creates a real (currently empty) connection.ContainerInfo root and wires it into the shell, replacing stage 3.1's placeholder label -- loading an actual .xml file into it is what will satisfy Phase 2/3's shared demo-config-file exit criterion once persistence (3.5) or an ad-hoc load path lands. check.sh and smoke.sh green. Same visual-verification limitation as stage 3.1 applies (documented phase-wide in blueprint/phase-3-ui.md) -- tests are all headless via ConnectionTree's public Widget API (ChildUIDs/IsBranch/CreateNode/UpdateNode/OnSelected are exported struct fields, so package ui_test could test everything without white-box access, unlike shell_test.go).
