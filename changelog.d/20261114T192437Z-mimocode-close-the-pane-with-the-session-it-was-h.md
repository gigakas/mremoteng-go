---
timestamp: 2026-11-14T19:24:37Z
agent: mimocode
files:
  - internal/ui/session.go
  - internal/ui/properties.go
  - internal/ui/welcome_test.go
  - internal/ui/panes_test.go
  - blueprint/linux-preview.md
---

Close the pane with the session it was holding

The X ended the transport and left the sub-panel in place, showing a closed session with nothing to do and still taking half the tab. It now closes a split pane as well: closeSelf ends the session and hands the space back to the sibling, which is what View > Close pane already did, while the last pane of a tab stays so Connect can open the next session in the same place. The confirmation rule is unchanged and still counts open sessions, not panes, so an empty sibling does not silence the question. TestCloseSelf_SplitPane_SurvivorKeepsTheKeyboard and the updated TestRequestDisconnect_* fail if the collapse is removed. Validated with ./scripts/check.sh.
