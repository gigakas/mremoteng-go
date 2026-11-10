---
timestamp: 2026-07-23T22:02:06Z
agent: claude-code
files:
  - auditory/phase3-stage4-20260724-claude-code.md
  - blueprint/phase-3-ui.md
  - cmd/mremoteng/main.go
  - internal/ui/properties.go
  - internal/ui/properties_test.go
  - internal/ui/shell.go
  - internal/ui/shell_test.go
---

Phase 3 stage 3.4: add the reflection-driven connection properties panel

Implemented PropertiesPanel, editing connection.ConnectionValues with per-field connection.InheritanceFlags toggles via reflection rather than ~75 hand-written bindings -- buildPropertyFields matches ConnectionValues' fields to InheritanceFlags' by name once at package init (a field with no counterpart, like Name or Hostname, gets no Inherit checkbox), mirroring how the original C# app's own PropertyGrid control is itself reflection-based. Found and fixed a real feedback-loop bug via a failing test: refresh calls widget.Entry.SetText/widget.Check.SetChecked to display the current value, but Fyne fires OnChanged for those the same as a genuine user edit, so checking Inherit triggered commit -> refresh -> SetText(effective value) -> a second recursive commit -> the effective value got written back into Raw, silently overwriting the local override underneath. Fixed with a refreshing guard bool that commit checks before doing anything. v1 scope, stated in the type's own doc comment: a flat list in model declaration order (not the original grid's categorized/tabbed sections -- Go reflection can't see the source comments delimiting them, and a hand-maintained field-to-category map was deferred), plain text Entry for every non-bool field including the ~20 named string-enum types (no dropdowns yet) -- both chosen because this phase's inability to see the running app makes a denser form-like layout hard to get right blind. Shell (3.1) gained a third pane via SetProperties, tree+properties stacked in a VSplit on the left; shell_test.go updated alongside. Wired into cmd/mremoteng: tree.OnSelect now populates the properties panel for any selected node (connection or folder), before its existing leaf-only tab-opening logic. 10 new tests in properties_test.go (row generation against the real model, not assumed; bool-vs-string widget choice; raw and inherited value display; edit commits; the toggle-inherit feedback-loop bug, now passing; int parsing including invalid input; SetTarget(nil) clearing) plus 1 in shell_test.go. check.sh and smoke.sh green.
