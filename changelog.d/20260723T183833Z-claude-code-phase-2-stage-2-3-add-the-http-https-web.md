---
timestamp: 2026-07-23T18:38:33Z
agent: claude-code
files:
  - auditory/phase2-stage3-20260723-claude-code.md
  - blueprint/phase-2-protocols.md
  - cmd/mremoteng/main.go
  - go.mod
  - go.sum
  - internal/protocol/web/web.go
  - internal/protocol/web/web_test.go
  - internal/protocol/window.go
---

Phase 2 stage 2.3: add the HTTP/HTTPS webview backend (unblocked)

Unblocked stage 2.3 mid-session by downloading a portable, no-installer mingw-w64 build (winlibs.com, plain zip extract, no admin rights needed) after chocolatey's admin-gated install failed -- MSVC's cl.exe (already installed via Visual Studio Build Tools) was tried first and confirmed NOT usable for cgo (Go's cgo passes gcc-style flags like -Werror that cl.exe rejects outright). This toolchain is session-local only, not committed to the repo or the machine's durable state -- documented three times (blueprint 2.3 notes, the stage audit, this fragment) so it isn't missed by whoever continues this work without their own C compiler. Implemented on github.com/webview/webview_go per the blueprint's explicit preference. Added protocol.WindowProtocol (NativeWindowHandle() uintptr), a third composed interface alongside TerminalProtocol/FramebufferProtocol, shaped to serve both this stage's in-process webview window and the external-process windows RDP/AnyDesk (stages 2.5/2.7) will need to reparent later -- established now so those stages reuse it. New registers for both ProtocolHTTP and ProtocolHTTPS (only the URL scheme differs). Found and fixed a real bug in the dependency while writing the first integration test: webview_go's Windows Terminate() is a bare PostQuitMessage(0), which Win32 documents as thread-local -- calling it from a different goroutine than the one running the locked-OS-thread message loop is silently a no-op, contradicting the library's own doc comment (true for its GTK backend, which dispatches internally; false for Win32). Fixed by routing Terminate through Dispatch (PostMessageW-based, correctly cross-thread) instead of calling it directly. The integration test creates a real OS-native window (verified safe to attempt in this environment first via a disposable, hard-timeout-guarded probe program) and asserts NativeWindowHandle() is non-zero and OnClose fires after Disconnect -- this is the test that caught the bug. Bounded by a context timeout so a future headless environment fails the test cleanly instead of hanging. check.sh and smoke.sh green with CGO_ENABLED=1 and the portable mingw on PATH; every other package still builds fine with cgo disabled.
