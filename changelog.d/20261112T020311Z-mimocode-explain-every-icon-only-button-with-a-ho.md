---
timestamp: 2026-11-12T02:03:11Z
agent: mimocode
files:
  - blueprint/linux-preview.md
  - internal/ui/actions.go
  - internal/ui/properties.go
  - internal/ui/session.go
  - internal/ui/shell.go
  - internal/ui/tooltip.go
  - internal/ui/tooltip_test.go
  - internal/ui/vault.go
---

Explain every icon-only button with a hover hint in the status bar

Eleven buttons showed an icon and no text at all, so what each one does had to be guessed: the four status-bar toggles, the RDP options and copy-screen actions, the three pane split controls, the vault save and the delete action. Fyne 2.8 has no tooltip support, and its popups are not usable for this either: the popup renderer resizes a background over the whole canvas, which takes the hover away from the button that opened it and makes a floating tooltip flicker. This repository already answers the same problem for tree rows by reporting in the status bar, so an icon button now does the same through a new hintButton: it keeps every widget.Button behaviour, waits 300 ms with the pointer resting, shows its one line in the status bar as secondary information, and on leave restores the previous text and its importance, so a problem message is not downgraded. The Config panel Delete button carries a text label instead, since that row has room. Two deviations kept deliberately: the split-pane controls and the vault button did not get labels, because they live in the session bar which the code already documents as overflowing into horizontal scrolling, so they use the hint instead. Added five tests covering the hover hint, leaving before the delay, restoring a danger message, the tap action surviving the embedded button, and the no-op clear. Validated with ./scripts/check.sh.
