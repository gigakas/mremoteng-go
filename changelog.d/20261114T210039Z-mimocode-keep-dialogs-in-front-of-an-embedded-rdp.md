---
timestamp: 2026-11-14T21:00:39Z
agent: mimocode
files:
  - internal/ui/dialogs.go
  - internal/ui/shell.go
  - internal/ui/session.go
  - internal/ui/transfer.go
  - internal/ui/rdp_test.go
---

Keep dialogs in front of an embedded RDP session

With an embedded remote desktop open, every dialog appeared behind it and the only way to read one was to disconnect the session. The cause is compositing: the embedded session is a native X11 child window, and X draws it above everything this process paints into the same window, so a Fyne popup cannot cover it. A dialog now unmaps the session windows while it is open and maps them back once the last one closes, using a depth counter so a dialog opened from a dialog does not make the remote desktop flash between them; the process keeps running throughout. Confirmations and the connection-file notice also go through our own modal now, because Fyne's dialog package draws straight into the canvas and would have landed behind the session for the same reason, and Fyne's file chooser — the only dialog this package does not draw — is bracketed with the same guard through SetOnClosed. TestShowModal_*, TestShowConfirm_* and TestModalGuard_RestoresOnce cover the hiding, the nesting, a repeated close and both answers. Validated with ./scripts/check.sh and ./scripts/smoke.sh.
