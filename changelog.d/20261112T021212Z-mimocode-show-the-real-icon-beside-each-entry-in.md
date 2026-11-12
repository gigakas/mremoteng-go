---
timestamp: 2026-11-12T02:12:12Z
agent: mimocode
files:
  - internal/ui/properties.go
  - internal/ui/welcome_test.go
---

Show the real icon beside each entry in the welcome button reference

The welcome reference listed the buttons by name only, so it said menu icon, gear icon and arrow icons in words while the window showed shapes: the reader still had to translate. Each row now carries the same fyne.Resource the button uses, drawn by widget.NewIcon in a fixed 28 pixel box so the names line up in one column, with the explanation under each name. Two entries legitimately repeat an icon and one has none: Ignore cert and Legacy TLS are both checkboxes, and Ctrl-C and Ctrl-D are text buttons, which the test allows for that row alone. Split right, split down and close pane became three separate rows instead of one, so each arrow is paired with what it does. The reference grew from fourteen to seventeen entries. Extended TestWelcomeHelp_DocumentsEveryIconButton to fail when a row that should show an icon sample has none. Validated with ./scripts/check.sh.
