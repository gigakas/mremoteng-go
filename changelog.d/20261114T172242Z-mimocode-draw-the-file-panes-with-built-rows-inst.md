---
timestamp: 2026-11-14T17:22:42Z
agent: mimocode
files:
  - internal/ui/filepane.go
  - internal/ui/filepane_test.go
---

Draw the file panes with built rows instead of list templates

Reported as file names not visible in either pane, local or remote, with the icons showing. The data and the row widgets were correct: the pane held the entries, and formatting a row by hand produced the right name, size and date. What failed was widget.List: its item templates were created but never populated in the running window, so every row drew empty labels, while the headless renderer walk found no labels at all. Both panes now rebuild a column of rows inside a scroll container, one widget per entry, each holding its own labels and receiving its own tap. Rows being built rather than recycled removes the whole Length, template and update dance that kept the names away, and tapping a row now carries its index instead of relying on the list selection. TestFilePane_Navigate_BuildsRowsWithNames asserts the built rows carry the name, the readable size, the date and their index, which is the regression guard for this bug. This also updates the layout described in auditory/linux-sftp-dualpane-20261114-mimocode.md: the typed row is kept, the widget.List is gone. Validated with ./scripts/check.sh, and a temporary renderer walk confirmed the names reach the rendered widget tree.
