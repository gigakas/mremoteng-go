---
timestamp: 2026-11-12T00:11:03Z
agent: claude-code
files:
  - internal/ui/terminal.go
---

Fix terminal zoom shortcuts swallowed by the focused widget's own TypedShortcut handler

Ctrl+=, Ctrl+- and Ctrl+0 never reached screen.zoomFocusedTerminal: Fyne's glfw driver hands every matching CustomShortcut straight to the focused widget's own TypedShortcut when it implements desktop.Shortcutable, before ever calling canvas.TypedShortcut where shell.go's registerControlShortcuts lives. termWidget implements TypedShortcut (for Ctrl-C/Ctrl-A/etc control bytes) and is focused whenever the console has the keyboard, which is exactly when zoom is wanted, so the canvas-level registration was unreachable and the keys were silently dropped. Fixed by handling KeyEqual/KeyMinus/Key0 inside termWidget.TypedShortcut's CustomShortcut case directly, calling adjustZoom the same way the (still-kept, for non-terminal focus) canvas shortcuts do. Found while investigating a separate reported SSH/Telnet console freeze on scroll, already fixed by 3b57861; this is the follow-up zoom-keys regression surfaced once that fix was verified. Validated with gofmt, go vet ./internal/ui/..., and a manual rebuild via scripts/dev.sh.
