---
timestamp: 2026-07-23T20:55:04Z
agent: claude-code
files:
  - auditory/phase3-stage1-20260723-claude-code.md
  - blueprint/phase-3-ui.md
  - cmd/mremoteng/main.go
  - go.mod
  - go.sum
  - internal/ui/shell.go
  - internal/ui/shell_test.go
  - scripts/smoke.sh
---

Phase 3 stage 3.1: add the Fyne application shell

Implemented the application shell: internal/ui/Shell wraps a fyne.App/Window with a minimal File/View/Help menu and a fixed two-pane layout (tree placeholder + tabs placeholder), matching the blueprint's v1 no-docking decision. SetTree/SetTabs let stages 3.2/3.3 replace the placeholders without touching shell.go. cmd/mremoteng/main.go now launches the real app (app.NewWithID + ui.NewShell + Window.ShowAndRun) instead of printing a skeleton message -- this is the actual deliverable of 'application shell', not scope creep. AppID is set now specifically because Preferences() (needed by stage 3.5) is silently disabled without a unique app ID, confirmed by the very first Fyne probe run before NewWithID was used. Before writing any UI code, verified Fyne itself builds and runs with the session's portable mingw toolchain (a real probe window gets a valid Win32 handle -- IsWindowVisible=true, on-screen coordinates, same SessionId as the interactive session) but discovered that window never appears in a screenshot taken from this session, with or without FYNE_RENDERER=software -- root cause not confirmed (ruled out: different Windows session). Raised this with the user before proceeding; they chose headless-only verification (fyne.io/fyne/v2/test) for the rest of Phase 3 rather than pausing to chase the screenshot issue. Necessarily updated scripts/smoke.sh: it used to grep the binary's stdout for a marker string, which doesn't apply to a GUI app whose ShowAndRun() blocks until closed and prints nothing -- replaced with launch-wait-verify-alive-then-kill, the same technique used to validate every GUI process probed in this session. New tests (internal/ui/shell_test.go, in package ui for direct field access rather than brittle content-tree type assertions): menu structure, placeholder types, Quit item wiring, and both SetTree/SetTabs mutators, all via Fyne's own headless test driver. check.sh and smoke.sh green (smoke.sh needed extra time for Fyne's first cgo/OpenGL compile, several minutes, same category as Fyne's own build time noted informally during the environment probe).
