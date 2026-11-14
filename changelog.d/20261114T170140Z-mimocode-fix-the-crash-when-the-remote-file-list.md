---
timestamp: 2026-11-14T17:01:40Z
agent: mimocode
files:
  - internal/ui/remote_files.go
  - internal/ui/remote_files_test.go
---

Fix the crash when the remote file list draws its first row

Opening Tools > Remote files closed the application. The dialog builds rows with widget.List, and its update step reached into the row container by index: container.NewBorder keeps its nil top and bottom slots, so the first type assertion ran on a nil interface and panicked before anything could be drawn. The row is now a typed remoteRow that carries its icon and labels, so nothing is recovered by position, and the list callbacks are methods (listLength, createRow, updateRow) instead of inline closures. Added TestRemoteBrowser_UpdateRow_FillsEachKindWithoutPanic, which builds the row and updates it for a directory and for a file, checks the name, the readable size and the date, and then feeds an out-of-range index and a foreign object because the list recycles rows while the entry slice shrinks. Grepped the rest of internal/ui for the same index-based assertion pattern: none left. Validated with ./scripts/check.sh.
