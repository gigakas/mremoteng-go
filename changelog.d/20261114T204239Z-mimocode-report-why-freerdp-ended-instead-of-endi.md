---
timestamp: 2026-11-14T20:42:39Z
agent: mimocode
files:
  - internal/ui/session.go
  - internal/ui/dialogs.go
  - internal/ui/shell.go
  - internal/ui/rdp_test.go
  - internal/ui/helpers_test.go
  - internal/ui/screen_test.go
  - internal/protocol/rdp/rdp.go
  - internal/protocol/rdp/rdp_test.go
  - auditory/linux-l7-options-20261114-mimocode.md
  - blueprint/backlog.md
---

Report why FreeRDP ended instead of ending silently

A session that died before opening anything said only that it had ended immediately, with a guess about certificates; the client had already written the reason to its own log and nobody read it. The watcher now reduces that log to one line, turns it into the next thing to try (gateway, certificate, credentials, security layer or address) and shows it in a dialog, because a status line behind the window that raised it is invisible. A session that ran for a while reports in the status bar with the cause instead, so the popup stays a failure signal. stripLogPrefixes also drops the separator FreeRDP puts between its logger prefix and the message, which used to lead the sentence with a dash. Verified against the real client through the application's Launch: connecting to a dead port yields 'connect to localhost:1 failed with error: Connection refused [111]', which is what the dialog shows. Validated with ./scripts/check.sh and ./scripts/smoke.sh.
