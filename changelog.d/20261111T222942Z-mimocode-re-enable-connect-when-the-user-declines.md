---
timestamp: 2026-11-11T22:29:42Z
agent: mimocode
files:
  - internal/ui/session.go
  - internal/ui/ui_test.go
---

Re-enable Connect when the user declines an unverified SSH host key

applyConnect disabled Connect while dialling and returned on the unknown-host-key path after showing the confirmation, passing no decline callback. Declining the dialog therefore left the tab with Connect and Disconnect both disabled, and the only way out was to close and reopen the session tab. The confirmation now receives a decline handler that returns the panel to the idle state through setConnected(false), the same transition every other ended session uses. Added TestApplyConnect_DeclinedUnknownHostKey_ReEnablesConnect, which injects a confirm that invokes the decline handler; it fails if the callback is nil again. Found while extracting the session state helpers, kept as its own change so the refactor stays behaviour-preserving. Validated with ./scripts/check.sh.
