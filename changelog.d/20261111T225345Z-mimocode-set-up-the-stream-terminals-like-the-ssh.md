---
timestamp: 2026-11-11T22:53:45Z
agent: mimocode
files:
  - internal/ui/session.go
  - internal/ui/stream.go
  - internal/ui/ui_test.go
---

Set up the stream terminals like the SSH one so zoom and copy work

startStream created the terminal widget and installed it in the slot directly, unlike startSession which wires clipboard, interrupt and zoom callbacks and wraps the widget in the zoom theme override. Telnet, RAW and rlogin sessions therefore ignored the zoom shortcuts entirely and their right-click copy did nothing, because onZoom and onCopy were nil. startStream now calls wireTerminal and installTerminal. Added forgetTerminalOverride, called whenever a session installs a new terminal widget: the previous wrapper belongs to the widget of the session that ended, and reusing it would refresh the wrong terminal on the first zoom of the next session. Extended TestSessionPanel_TelnetConnect_ShowsTerminalAndSendsInput to assert the override exists and the callbacks are wired. Validated with ./scripts/check.sh.
