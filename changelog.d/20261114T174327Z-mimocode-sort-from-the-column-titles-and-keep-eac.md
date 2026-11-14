---
timestamp: 2026-11-14T17:43:27Z
agent: mimocode
files:
  - internal/ui/filepane.go
  - internal/ui/filepane_test.go
  - internal/ui/remote_files.go
---

Sort from the column titles and keep each transfer button in its own pane

Two layout changes in the transfer window. Ordering was a dropdown in the pane header, which hid both the available columns and the current direction; each pane now draws clickable column titles, Name, Size and Date, laid out like a row. Clicking one orders by that column, clicking it again reverses the direction, and the active title carries an arrow. A new column starts with its natural direction: names A-Z, sizes and dates largest and newest first. Directories stay first in every order. Size is a new sort key. The two transfer buttons also left the shared footer, where they said nothing about which side they acted on: Upload to remote now sits in the local pane and Download to local in the remote pane, so each button is inside the pane it reads from, and the footer keeps only the progress line and Close. Tests cover the six order and direction combinations with the directory-first guarantee, that a second click on a column reverses it, that the active header carries the arrow while the others stay plain, and that re-ordering rebuilds the rows. Validated with ./scripts/check.sh, with no Fyne errors in its output.
