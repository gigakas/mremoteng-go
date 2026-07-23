---
timestamp: 2026-07-23T21:39:58Z
agent: claude-code
files:
  - auditory/phase3-stage3-20260723-claude-code.md
  - blueprint/phase-3-ui.md
  - cmd/mremoteng/main.go
  - internal/protocol/winembed/winembed.go
  - internal/ui/framebuffer.go
  - internal/ui/framebuffer_test.go
  - internal/ui/nativewindow.go
  - internal/ui/nativewindow_other.go
  - internal/ui/nativewindow_test.go
  - internal/ui/nativewindow_windows.go
  - internal/ui/sessiontabs.go
  - internal/ui/sessiontabs_test.go
---

Phase 3 stage 3.3b: add the VNC framebuffer view

Added FramebufferView (internal/ui/framebuffer.go), rendering a protocol.FramebufferProtocol's Frames() channel via canvas.Image and forwarding pointer/keyboard input back through SendPointer/SendKey. X11 keysyms for printable ASCII/Latin-1 characters are numerically identical to their character code (the keysymdef.h convention), so TypedRune needs no lookup table -- only non-printable keys (framebufferKeysyms) do. v1 simplification stated in the type's own doc comment: the image renders at native resolution (canvas.ImageFillOriginal) rather than scaled to fit the tab, specifically so widget-space pointer coordinates equal framebuffer pixel coordinates with no scale-factor math, trading fit-to-window flexibility for avoiding a real class of off-by-scale bugs. 4 new tests, including a real render-a-pushed-frame check confirming fyne.Do executes promptly even without a running app event loop in this headless test environment.
