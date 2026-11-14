---
timestamp: 2026-11-14T23:17:50Z
agent: claude-code
files:
  - cmd/changelog/main.go
  - cmd/changelog/main_test.go
  - internal/changelog/changelog.go
  - internal/connection/inheritance.go
  - internal/connection/inheritance_test.go
  - internal/profiles/profiles.go
  - internal/profiles/store.go
  - internal/protocol/rdp/rdp.go
  - internal/protocol/serial/serial.go
  - internal/protocol/telnet/telnet.go
  - internal/serialize/csv/csv.go
  - internal/serialize/xml
  - internal/transfer/transfer.go
  - internal/ui/session.go
  - internal/ui/session_rdp.go
  - internal/ui/shell.go
  - internal/ui/shell_shortcuts.go
  - internal/ui/shell_tabs.go
  - internal/ui/terminal.go
  - internal/ui/terminal_input.go
---

Split long functions and the largest UI files without changing behaviour

Extracted helpers from the XML and CSV serializers, the profile store, vault, VNC handshake, RDP arguments, serial termios, transfer import, changelog parser and Profile.Validate. Inheritance flags now copy through a field table derived by reflection, with tests that every flag maps to its value field. Moved RDP, terminal input, shortcut and tab code out of session.go, terminal.go and shell.go. XML attribute order is unchanged.
