---
timestamp: 2026-07-15T22:16:37Z
agent: claude-code
files:
  - internal/spike/reparent/main.go
  - internal/spike/reparent/x11.go
  - internal/spike/reparent/win32.go
---

Stage 0.2: set mixed DPI hosting on the main thread before window creation — the behavior is captured per-window at creation, fixing the silent SetParent refusal
