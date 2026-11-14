---
timestamp: 2026-11-14T22:25:16Z
agent: mimocode-c3
files:
  - internal/profiles/profiles.go
  - internal/ui/panes.go
  - internal/ui/panes_test.go
---

Remember where the divider was left

A restored divided tab came back at half, whatever the user had arranged. PaneLayout now stores the divider position, taking it from the split widget when the tabs are written and putting it back when the tree is rebuilt; the half is treated as the default and left out of the file, so a tab nobody dragged stays clean, and a position outside 0-1 is refused by the same validation that guards the shape. The test is a unit test on the pane node because Fyne's split renderer clamps the offset to the children's minimum size, and a headless window is too small for any position other than the half to survive a layout pass - testing it end to end would only assert the clamp. Validated with ./scripts/check.sh.
