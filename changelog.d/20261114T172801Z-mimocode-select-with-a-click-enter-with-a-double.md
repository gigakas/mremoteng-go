---
timestamp: 2026-11-14T17:28:01Z
agent: mimocode
files:
  - internal/protocol/ssh/files.go
  - internal/protocol/ssh/files_test.go
  - internal/ui/filepane.go
  - internal/ui/filepane_test.go
  - internal/ui/remote_files.go
---

Select with a click, enter with a double click, and copy folders recursively

Three problems with the two-pane window. The name overlapped the size and the date on long entries, so it now takes the middle of the row and truncates with an ellipsis. A click silently either entered a directory or selected a file with no visible state, so a click now only selects: the row paints a selection background and the status line names what is selected, while a double click enters a directory, which means a folder can be selected and recognised. And folders were not transferable at all: RemoteFiles.Upload and Download now stat the source and, when it is a directory, walk it, creating the directories on the other side, so a tree both uploads and downloads. Tree progress is one running total: byteCounter turns the per-file callbacks into cumulative bytes against the size of the whole tree, computed by a local walk or a remote listing. Tests: nested trees in both directions with their content and their complete progress total, plus the pane selection now returning a directory as transferable. Validated with ./scripts/check.sh.
