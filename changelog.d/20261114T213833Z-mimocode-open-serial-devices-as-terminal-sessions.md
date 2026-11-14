---
timestamp: 2026-11-14T21:38:33Z
agent: mimocode
files:
  - internal/protocol/serial/serial.go
  - internal/protocol/serial/serial_test.go
  - internal/profiles/profiles.go
  - internal/profiles/profiles_test.go
  - internal/ui/serial.go
  - internal/ui/serial_test.go
  - internal/ui/stream.go
  - internal/ui/session.go
  - internal/ui/properties.go
  - internal/ui/shell.go
  - internal/ui/session_test.go
  - internal/ui/stream_test.go
  - internal/ui/terminal_test.go
  - auditory/linux-l8-serial-20261114-mimocode.md
  - blueprint/backlog.md
  - blueprint/linux-preview.md
---

Open serial devices as terminal sessions

The last protocol of L8 that did not need a webview: a SERIAL connection opens a device path as a byte stream and hands it to the same terminal as Telnet and RAW. The line discipline is cleared explicitly (no echo, no signals, no CR/LF or case translation, VMIN=1 so a read blocks) because a serial port that looks dead is almost always a setting nobody looked at, and the four line settings live in the profile with zero values meaning 9600 8N1 so older store files still open. A serial connection addresses a device, so its host is validated as an absolute path and it has no network port at all. A Line button on the serial session bar opens the settings dialog, and the status line spells them out (serial 115200 8N1 session with /dev/ttyS0). startStream lost its telnet mode parameter and takes a label, which is what let serial reuse the stream path. Verified over a pseudo terminal created with /dev/ptmx rather than hardware, which the audit records as the limit. Validated with ./scripts/check.sh and ./scripts/smoke.sh.
