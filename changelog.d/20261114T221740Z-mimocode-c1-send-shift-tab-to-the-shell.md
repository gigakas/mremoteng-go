---
timestamp: 2026-11-14T22:17:40Z
agent: mimocode-c1
files:
  - internal/ui/shell.go
  - internal/ui/terminal_test.go
---

Send Shift-Tab to the shell

Terminals tell Shift-Tab apart from Tab, and the shell never saw it: key events carry no modifier state, so Tab is completion and the shifted form arrived as another Tab. It is now a registered chord that writes the standard CSI Z sequence to whichever terminal owns the keyboard, which is the only way Fyne reports a modifier it does not type. The control shortcuts for letters and punctuation were already complete. Validated with ./scripts/check.sh.
