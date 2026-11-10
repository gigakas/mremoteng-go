---
timestamp: 2026-07-23T19:00:43Z
agent: claude-code
files:
  - auditory/phase2-stage5-20260723-claude-code.md
  - blueprint/phase-2-protocols.md
  - cmd/mremoteng/main.go
  - go.mod
  - internal/protocol/rdp/embed_linux.go
  - internal/protocol/rdp/embed_windows.go
  - internal/protocol/rdp/embed_windows_test.go
  - internal/protocol/rdp/rdp.go
  - internal/protocol/rdp/rdp_test.go
---

Phase 2 stage 2.5: add the RDP backend (external process + reparent)

Implemented RDP as protocol.WindowProtocol (the interface introduced in stage 2.3), launching an external FreeRDP client and never linking libfreerdp, per the blueprint's non-negotiable principle. Rebuilt the reparenting mechanism from docs/spike-win32.md/docs/spike-x11.md's documented findings, since the actual Phase 0 spike code was deleted at that phase's close. Windows (embed_windows.go): launches sdl-freerdp.exe (not mstsc -- the spike found it unreliable, kept only as documented fallback), finds its session window with the spike's find-and-adopt retry loop (EnumWindows/GetWindowThreadProcessId/GetClassName, skipping #32770 dialogs), and reparents it via EmbedChild following the exact validated recipe: SetParent, restyle (strip WS_POPUP/WS_CAPTION/WS_THICKFRAME, add WS_CHILD), SetWindowPos(FRAMECHANGED), then verify via GetAncestor rather than trusting SetParent's return value -- the DPI_AWARENESS_CONTEXT mismatch that makes SetParent silently no-op on Windows 10+ is the exact failure mode the spike spent multiple rounds debugging. All seven user32.dll functions needed for this (SetParent, Get/SetWindowLongPtrW, SetWindowPos, GetAncestor, SetThreadDpiHostingBehavior, SetProcessDpiAwarenessContext) aren't wrapped by golang.org/x/sys/windows, so they're hand-bound via LazyDLL, using the exact constant values the spike's own hard-won documentation records (DPI_HOSTING_BEHAVIOR_MIXED=1, not 2 -- passing 2 fails silently and cost the spike several rounds). Tested for real: TestEmbedChild_ReparentsARealExternalWindow creates a genuine parent window (with the DPI hosting behavior dance actually performed, exactly as EmbedChild's doc comment requires of a caller) and reparents mspaint.exe's real window into it, verified via GetAncestor -- this would fail if the DPI handling were wrong. Before writing the test window's WNDCLASSEXW struct, verified its actual field layout (sizeof 80, specific offsets) by compiling a throwaway C probe against the session's mingw headers rather than trusting memory. Also discovered and documented that notepad.exe is unusable as an external-process test target on this Windows 11 install: it's MSIX-packaged and the launched pid doesn't own the window (confirmed via Get-Process); mspaint.exe was checked and used instead. Linux (embed_linux.go): launches xfreerdp and tracks its lifecycle, but window discovery/embedding is explicitly NOT implemented -- no X server, no xfreerdp binary, no way to validate xgb-based EWMH code in this session, so NativeWindowHandle() honestly returns 0 there rather than shipping unverified protocol parsing. check.sh and smoke.sh green; internal/protocol/rdp alone cross-compiles cleanly for GOOS=linux with cgo disabled (the whole-module cross-compile fails only because of stage 2.3's pre-existing, expected cgo requirement for the web backend, not because of anything in this stage -- flagged as a Phase 4 packaging consideration).
