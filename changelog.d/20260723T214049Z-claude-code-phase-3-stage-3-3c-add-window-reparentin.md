---
timestamp: 2026-07-23T21:40:49Z
agent: claude-code
files:
  - auditory/phase3-stage3-20260723-claude-code.md
  - blueprint/phase-3-ui.md
  - cmd/mremoteng/main.go
  - internal/protocol/winembed/winembed.go
  - internal/ui/nativewindow.go
  - internal/ui/nativewindow_other.go
  - internal/ui/nativewindow_test.go
  - internal/ui/nativewindow_windows.go
  - internal/ui/sessiontabs.go
  - internal/ui/sessiontabs_test.go
---

Phase 3 stage 3.3c: add window-reparenting tab hosting and the session-tabs assembly point

Added NativeWindowHost (internal/ui/nativewindow*.go), platform-split like internal/protocol/rdp: nativewindow_windows.go's Embed is winembed.EmbedChild's first production use against a real Fyne-owned window (via driver.NativeWindow/driver.WindowsWindowContext) rather than the hand-built test window stage 2.5/2.7 already validated it against; nativewindow_other.go stubs it with a clear not-implemented error, matching RDP/AnyDesk's own Linux gaps. Added winembed.SetWindowPosition (internal/protocol/winembed/winembed.go) for repositioning an already-embedded child as its host widget's geometry changes -- a distinct operation from EmbedChild's one-time placement, added to the package that owns the mechanism it extends rather than duplicated in internal/ui. Stated plainly rather than assumed correct: Move's position is relative to the widget's immediate parent container, not necessarily the window's absolute client-area origin for deeply nested layouts -- correct for a host placed directly in window content, unverified beyond that, no real display was available to check. Added SessionTabs (internal/ui/sessiontabs.go), the assembly point: Open dispatches to Terminal/FramebufferView/NativeWindowHost via a type switch on TerminalProtocol/FramebufferProtocol/WindowProtocol, shows the tab immediately, connects in the background with a 30s timeout, wires connect-dependent parts (read pump, frame attach, window embed) only after Connect succeeds, and OnClose removes the tab automatically; connect failure replaces the tab content with a visible error rather than a silently dead tab. Wired into cmd/mremoteng/main.go: shell.SetTabs(tabs.Widget), and the connection tree's OnSelect now calls protocol.Create + SessionTabs.Open for a selected connection leaf -- real wiring, currently unreachable since nothing populates the tree yet (persistence is stage 3.5). What's explicitly NOT tested: NativeWindowHost.Embed against a real, live Fyne window's real HWND -- judged impractical to stand up from inside go test without a full ShowAndRun() event loop (unlike stage 2.3's standalone webview probe), and winembed.EmbedChild itself already has a genuine integration test from stage 2.5 against a hand-built Win32 window, so what's untested here is specifically the last step of wiring that proven mechanism to a genuine Fyne-owned window. 9 new tests: 3 for NativeWindowHost (including confirming empirically, not assumed, that test.NewWindow() does not implement driver.NativeWindow), 6 for SessionTabs (dispatch to all three view types, the unsupported-protocol fallback, connect-failure display, OnClose removal). check.sh and smoke.sh green for the whole stage.
