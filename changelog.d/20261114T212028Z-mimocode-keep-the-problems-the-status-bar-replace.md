---
timestamp: 2026-11-14T21:20:28Z
agent: mimocode
files:
  - internal/ui/notifications.go
  - internal/ui/notifications_test.go
  - internal/ui/shell.go
  - internal/ui/actions.go
  - internal/ui/session.go
  - internal/ui/properties.go
  - internal/ui/welcome_test.go
  - auditory/linux-l10-notifications-20261114-mimocode.md
  - blueprint/backlog.md
  - blueprint/linux-preview.md
---

Keep the problems the status bar replaces

The window could only ever show one message: a failure painted the status bar and the activity log, and the next message replaced it before anyone read it. A notifications panel now keeps a bounded history of the failures and warnings, with a count in its title, a Clear button and a badge in the status bar so the panel is discoverable after the first problem. setProblem records every failure (SSH, RDP, VNC, transfers and the store all funnel through it) and the new setWarning covers the case that is neither a success nor a hard failure, FreeRDP ending after a session had been open. Both bottom panels share one slot and hide each other, because two stacked panels would take the height the terminal needs. Nothing is persisted: a notification describes this run, and a problem that outlives the window belongs in the activity log the user can copy. The start page's Buttons card documents the new icon. Validated with ./scripts/check.sh and ./scripts/smoke.sh.
