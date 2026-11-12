---
timestamp: 2026-11-12T01:13:05Z
agent: claude-code
files:
  - internal/ui/panes.go
  - internal/ui/panes_test.go
  - internal/ui/session.go
  - internal/ui/shell.go
  - internal/ui/quickconnect.go
  - internal/ui/folders.go
  - internal/ui/actions.go
  - internal/ui/diagnostics.go
---

Split a terminal tab into tmux-style panes that duplicate the connection

A session tab can now be split side by side or stacked, nested to any depth, with each pane a full sessionPanel connected to the same profile (typed password copied from the origin pane). New panes.go holds a small paneNode tree: leaves show a panel, splits show an HSplit/VSplit, and a leaf turns into a split in place, so closing a pane gives its space to the sibling. Controls are three icon buttons on terminal session bars and View > Split pane right / Split pane down / Close pane acting on the pane whose terminal has the keyboard. Only SSH, Telnet, RAW and rlogin split: RDP and VNC embed a native view. The last pane of a tab cannot be closed (close the tab). Tab close, connection delete, the active-session count and app shutdown now go through tabPanels/allPanels so every pane's session is ended; the tab pointer moves to a survivor if the original pane closes; reopening a split tab no longer rebuilds its layout. A duplicate to a host with an unverified key asks again. No keyboard shortcut yet: Ctrl combinations are forwarded to the shell. Added panes_test.go; the test profile has no username so the duplicate fails synchronously, since the test driver runs fyne.Do inline. Validated with ./scripts/check.sh.
