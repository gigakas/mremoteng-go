---
timestamp: 2026-11-14T22:25:24Z
agent: mimocode-d2
files:
  - internal/protocol/serial/serial.go
  - internal/protocol/serial/serial_test.go
  - internal/profiles/profiles.go
  - internal/ui/serial.go
  - internal/ui/serial_test.go
---

Offer the serial handshake mRemoteNG offers

A serial connection had no flow control at all: raw mode cleared both handshakes and the dialog had no field for them, which is the first thing to check when a device stops sending. The Line dialog gains none, RTS/CTS and XON/XOFF, stored in the profile with none as the default so an older store file keeps working, validated at the store boundary and again in the protocol package, applied as CRTSCTS for the hardware handshake and IXON|IXOFF for the software one, and named in the session status line. TestTermios_MapsFlowControl pins each flag and the default; the profile round trip and the dialog validation cover the storage side. Validated with ./scripts/check.sh.
