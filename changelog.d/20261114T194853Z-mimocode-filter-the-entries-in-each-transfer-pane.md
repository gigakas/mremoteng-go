---
timestamp: 2026-11-14T19:48:53Z
agent: mimocode
files:
  - internal/ui/filepane.go
  - internal/ui/filepane_test.go
  - blueprint/linux-preview.md
---

Filter the entries in each transfer pane

A directory with hundreds of entries could only be read by scrolling, and the panes already had every other verb. Each pane now carries a name filter under its path bar: typing keeps the entries whose name contains the text, ignoring case, and the status line reports how many match so a short list is never mistaken for an empty directory. A Clear button empties the field, because a filtered pane with no matches reads exactly like a directory with nothing in it. The pane keeps the listed directory next to what the filter leaves visible: clearing the filter then costs no second listing, which over SFTP would be another round trip, and sorting reorders the full listing so the order survives the filter. Filtering drops the selection on purpose, since row indices move with the filter and a stale index would rename or transfer a different entry. TestFilePane_Filter_* cover the case-insensitive match, the count, clearing, the dropped selection and navigation under a filter; TestFilePane_Sort_KeepsTheWholeListingInOrderAfterClearingTheFilter covers the ordering. Validated with ./scripts/check.sh.
