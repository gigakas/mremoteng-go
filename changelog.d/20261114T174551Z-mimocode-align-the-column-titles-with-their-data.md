---
timestamp: 2026-11-14T17:45:51Z
agent: mimocode
files:
  - internal/ui/filepane.go
  - internal/ui/filepane_test.go
---

Align the column titles with their data in the transfer panes

Reported as the column names not sitting over their values. The header and the rows were laid out differently: the rows padded their tail group and centred the icon box, the header did neither, and a button is wider than a label, so every header drifted away from the column it named. Both now go through one layout, columns(), which fixes the three widths once (icon 26, size 92, date 130 pixels) and gives the name the rest; a data row and the header call it with their own widgets, so the titles cannot drift again no matter what a row contains. The numeric titles align their text to the trailing edge, matching how the sizes and dates are drawn. Added TestColumns_HeaderAndRow_ShareTheSameOffsets, which lays a row and a header out at the same width and fails if the size and date slots do not land on identical x positions. Validated with ./scripts/check.sh, without Fyne errors in its output.
