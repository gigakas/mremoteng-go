---
timestamp: 2026-11-14T18:03:40Z
agent: mimocode
files:
  - internal/ui/filepane.go
  - internal/ui/filepane_test.go
  - internal/ui/remote_files.go
---

Select several entries with Shift and Control in the transfer panes

The panes held one selection, so moving a set of files meant one transfer per file. desktop.MouseEvent carries the modifier state, unlike the scroll events that have already caused trouble here, so the rows now read the click through MouseDown and the pane keeps a set of indexes with an anchor. A plain click selects one row and sets the anchor, Shift extends the range from the anchor in either direction, and Control or Super toggles a single row, which is the range behaviour every file manager has. The highlight is painted from the set, so a clicked row is visibly in or out. Transfers and Delete act on the whole selection: the status line names one entry or counts the set with its folders and files, transferMany copies the entries in order through the same progress bar, Stop button and log, and a cancel stops the queue between entries as well as inside a file, while Delete confirms a count and warns how many folders take their contents with them. Rows no longer implement Tapped: it would fire after MouseDown without modifiers and undo a Control click. Tests cover the plain, Shift in both directions and Control toggle cases, the ordered selection result, and the existing pane behaviour. Validated with ./scripts/check.sh.
