---
timestamp: 2026-11-12T02:09:53Z
agent: mimocode
files:
  - internal/ui/properties.go
  - internal/ui/welcome_test.go
---

Document every button on the welcome page and correct its stale notice

The start page welcomed the user and explained how to create a connection, but said nothing about the icons on the three bars, which was the same problem the hover hints only solve one icon at a time: you cannot hover what you do not know exists. The welcome tab now carries a Buttons card listing all fourteen entries with what each does, in a scrollable column so a short window still shows the getting-started steps first. The rows live in a package-level welcomeHelp slice so a test can guard them: thirteen rows at minimum, complete, deduplicated, each a full sentence, and covering the tree, Config panel, activity log, quick connect, vault, split controls, clipboard copy and delete. The page also claimed that session support was not implemented and that SSH and RDP were the next stage, which stopped being true several stages ago; the notice now lists what works today and the single runtime requirement, xfreerdp3 on the PATH. Validated with ./scripts/check.sh.
