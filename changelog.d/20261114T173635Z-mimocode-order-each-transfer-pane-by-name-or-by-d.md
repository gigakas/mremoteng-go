---
timestamp: 2026-11-14T17:36:35Z
agent: mimocode
files:
  - internal/ui/filepane.go
  - internal/ui/filepane_test.go
---

Order each transfer pane by name or by date

Both panes listed alphabetically with no way to change it, which is useless when you are looking for the file you just wrote. Each pane now carries an order control with four options: name A-Z, name Z-A, date newest and date oldest. The choice is per pane, because the local and the remote side rarely want the same view. Directories stay first in every order, so navigation never depends on the sort, and the comparison is case-insensitive by name with the date as the primary key when that is the chosen order. Changing the order re-sorts what is already on screen and rebuilds the rows without listing the directory again, which keeps it instant on a slow link. Tests cover the four orders including their directory-first guarantee, and that re-ordering works on the entries already listed. Validated with ./scripts/check.sh.
