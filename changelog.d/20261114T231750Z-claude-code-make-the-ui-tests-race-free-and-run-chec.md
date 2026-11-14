---
timestamp: 2026-11-14T23:17:50Z
agent: claude-code
files:
  - internal/ui/dispatch.go
  - internal/ui/dispatch_test.go
  - internal/ui/helpers_test.go
  - internal/ui/screen_test.go
  - internal/ui/rdp_test.go
  - internal/ui/tooltip_test.go
  - internal/ui/embed.go
  - internal/ui/remote_files.go
  - internal/ui/serial.go
  - internal/ui/sessionstate.go
  - internal/ui/tooltip.go
  - internal/ui/vnc.go
  - internal/ui/webview.go
  - scripts/check.sh
---

Make the UI tests race-free and run check.sh with -race

Worker callbacks now go through runOnUI, which tests queue and drain on the test goroutine because Fyne's test driver runs fyne.Do inline. The session fakes block until closed like a real pty. go test -race ./... now passes and check.sh runs it.
