---
timestamp: 2026-11-12T00:57:34Z
agent: claude-code
files:
  - internal/ui/terminal.go
  - internal/ui/ui_test.go
---

Return the terminal view to the live line when typing while scrolled back

While scrolled into history, Enter (and any typed key or paste) was sent to the session but the view stayed on the old lines. sendBytes now resets scrollOffset to 0 and repaints before forwarding the input. Arrow and paging keys still navigate the scrollback because scrollThroughHistory consumes them first. Added TestTermWidget_EnterWhileScrolledBack_ReturnsToTheLiveLine. Validated with gofmt, go vet and go test for internal/ui and internal/terminal.
