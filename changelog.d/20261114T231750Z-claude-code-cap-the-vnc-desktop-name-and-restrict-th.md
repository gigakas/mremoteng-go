---
timestamp: 2026-11-14T23:17:50Z
agent: claude-code
files:
  - internal/protocol/vnc/vnc.go
  - internal/protocol/vnc/vnc_test.go
  - internal/ui/stream.go
  - internal/ui/stream_test.go
---

Cap the VNC desktop name and restrict the browser launcher to web URLs

A hostile VNC server could announce a 4 GiB desktop name and make the client allocate it; the name is now capped at 64 KiB. xdg-open only receives http and https URLs. Both have tests.
