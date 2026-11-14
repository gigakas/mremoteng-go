---
timestamp: 2026-11-14T19:13:52Z
agent: mimocode
files:
  - internal/ui/session.go
  - internal/ui/panes.go
  - internal/ui/properties.go
  - internal/ui/welcome_test.go
  - internal/ui/panes_test.go
  - blueprint/linux-preview.md
---

Document what the X does and ask only for the last session

The start page explained the X as an unconditional disconnect and still listed a Close pane button that had moved to the View menu. That line now shows the icon the bar actually uses, states the confirmation, and is joined by the SFTP button it used to omit. The X itself asks for approval only when the session being closed is the last open one of its tab: a split tab holds one group of sessions, so disconnecting a pane whose siblings are still connected changes nothing and the question would only be in the way. TestRequestDisconnect_* cover the skipped question, an empty sibling pane and the declined case, and TestWelcomeHelp_Disconnect_StatesTheConfirmation keeps the card in step with the behaviour. Validated with ./scripts/check.sh.
