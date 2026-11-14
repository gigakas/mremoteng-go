---
timestamp: 2026-11-14T22:17:31Z
agent: mimocode-l11
files:
  - internal/ui/diagnostics.go
  - internal/ui/session_test.go
  - internal/ui/rdp_test.go
  - internal/profiles/profiles_test.go
---

Prove that closing the window leaves nothing behind

L11 asked for proof that no X11 child or child process survives the application. The diagnostics counter now includes the embedded web view, which it had missed since HTTP stopped opening a browser, and three tests pin the cleanup: shutdownSessions ends the shell, the stream, the FreeRDP process and the web helper and destroys the embedded window, disconnecting an RDP session destroys its window, and a version 1 store that carries the later optional blocks (restored tabs, layouts, manual order) keeps them instead of dropping the workspace. The migration test itself, the unsupported-version rejection and the copy-diagnostics button already existed, so this closes the remaining gap rather than repeating them. Validated with ./scripts/check.sh.
