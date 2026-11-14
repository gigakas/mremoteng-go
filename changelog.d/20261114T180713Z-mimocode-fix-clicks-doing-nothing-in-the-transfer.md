---
timestamp: 2026-11-14T18:07:13Z
agent: mimocode
files:
  - internal/ui/filepane.go
  - internal/ui/filepane_test.go
---

Fix clicks doing nothing in the transfer panes

Reported right after multi-selection landed: nothing could be selected at all. The rows moved from Tapped to MouseDown to read the Shift and Control modifiers, but desktop.Mouseable is not a one-method interface: it requires MouseDown and MouseUp. With only half of it the row satisfied no click interface, and because Tapped had been removed at the same time, a click found no handler and did nothing, while double click still worked because DoubleTappable was intact. The row now implements MouseUp as well. Two guards for this class of mistake: compile-time assertions that the row is a desktop.Mouseable and a fyne.DoubleTappable, so a missing method fails the build instead of silently swallowing clicks, and TestFileRow_MouseDown_SelectsTheRowWithItsModifiers, which drives the row rather than the pane so the modifier path is covered from the mouse event to the selection, including the secondary-button case that must not select. Validated with ./scripts/check.sh, without Fyne errors in its output.
