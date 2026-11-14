---
timestamp: 2026-11-14T21:31:26Z
agent: mimocode
files:
  - internal/ui/specialkeys.go
  - internal/ui/specialkeys_test.go
  - internal/ui/embed.go
  - internal/ui/embed_test.go
  - internal/ui/session.go
  - internal/ui/shell.go
  - internal/ui/properties.go
  - internal/ui/helpers_test.go
  - auditory/linux-l7-keys-fullscreen-20261114-mimocode.md
  - blueprint/backlog.md
---

Send the chords a desktop swallows, and fill the screen

Three L7 items: special keys, full screen and multi-monitor. Ctrl+Alt+Del never leaves the local X server and Alt+Tab is grabbed by the window manager, so an embedded session cannot receive either however hard the user types; the new control focuses the session window and fakes the keys with XTEST, resolving keycodes from the X keyboard mapping so it is not a US-layout assumption. The session bar gets one icon-only button for Ctrl+Alt+Del and View > Send keys carries the rest, because a bar of five labels would push the session into horizontal scrolling. View > Full screen toggles the window, reports its state and marks itself when active, since a desktop that hides its window controls would otherwise leave no way back. Multi-monitor was not pending but broken: /multimon was passed to an embedded session, where it cannot mean anything because the window is clamped to the tab; the flag now opens FreeRDP's own window. Validated with ./scripts/check.sh and ./scripts/smoke.sh, and XTEST plus every chord keysym probed on this machine's display.
