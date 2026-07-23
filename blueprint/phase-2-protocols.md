# Phase 2 — Protocols (increasing risk order)

**Goal**: working protocol implementations behind a common lifecycle
interface (Connect/Disconnect/Focus, mirroring the original `ProtocolBase`).
**Depends on**: Phase 1 (data core) for all stages; Phase 0 (spike) for
stages 2.5 and 2.7.

**Owned packages**: `internal/protocol/` and its per-protocol subpackages.

## Stages

| # | Stage | Status | Agent |
|---|---|---|---|
| 2.1 | Protocol interface + factory | done | claude-code |
| 2.2 | SSH, Telnet, rlogin, raw socket, serial | done | claude-code |
| 2.3 | HTTP/HTTPS (native webview) | done | claude-code |
| 2.4 | VNC | done | claude-code |
| 2.5 | RDP (external xfreerdp + reparent) | done | claude-code |
| 2.6 | PowerShell remoting (WinRM) | pending | opencode |
| 2.7 | AnyDesk (external process) | pending | claude-code |

### Parallelism & collision notes

- 2.1 blocks everything else: it owns `internal/protocol/` (interface,
  factory, shared lifecycle types). No other stage starts before it closes.
- After 2.1, each protocol stage owns its own subpackage —
  `internal/protocol/ssh/`, `internal/protocol/web/`,
  `internal/protocol/vnc/`, `internal/protocol/rdp/`,
  `internal/protocol/winrm/`, `internal/protocol/anydesk/` — disjoint, so
  protocol stages can run in parallel across agents.
- **Factory registration is the only shared touch point**: the factory file
  belongs to 2.1. A protocol stage wires itself in via a separate, minimal
  commit that only adds its registration, justified in its changelog
  fragment (rule 3 in `blueprint/README.md`).
- 2.2 stays in claude-code because the terminal emulator widget needs
  visual iteration; 2.5/2.7 reuse the Phase 0 reparenting findings and need
  a desktop. 2.4 and 2.6 are library-driven and test-first — good OpenCode
  candidates. 2.3 involves a cgo webview wrapper on two platforms — either
  agent, decided at claim time.

### 2.1 Protocol interface + factory
- `Protocol` interface: Connect/Disconnect/Focus/Resize lifecycle + error
  and close events. Single construction point keyed on the protocol type
  (mirrors `ProtocolFactory.cs`).
- Tests with a fake protocol implementation.

### 2.2 SSH, Telnet, rlogin, raw socket, serial
- SSH via `golang.org/x/crypto/ssh`; Telnet/rlogin/raw as thin custom
  implementations; serial via a Go serial-port library. Rendering into an
  embedded terminal emulator widget.
- Tests: connection lifecycle against local in-process servers.
- **The real cost driver is the terminal emulator widget** (VT100/xterm
  escape-sequence parsing, rendering on Fyne, scrollback, selection/copy),
  not the protocols themselves. Estimate it separately before starting.
- **Contingency — PuTTY as external backend**: if the terminal widget proves
  too costly, fall back to launching PuTTY as an external process embedded
  via window reparenting — the same mechanism validated in Phase 0 for
  xfreerdp, and the exact pattern the original mRemoteNG uses with
  `PuTTYNG.exe` for SSH/Telnet/rlogin/serial. Unlike FreeRDP there is no
  license barrier (PuTTY is MIT, GPLv2-compatible even for linking).
  Trade-offs to accept if triggered: credentials passed via CLI/session
  files instead of in-process API, SSH tunnels need local port coordination
  with external processes instead of in-process channels, and an external
  runtime dependency for terminal protocols. Native Go remains the target;
  invoking this fallback must be justified in the stage audit.

### 2.3 HTTP/HTTPS
- OS-native webview (WebView2 on Windows, WebKitGTK on Linux) via a thin
  wrapper. This is the only place where cgo is acceptable, and only through
  the wrapper library.
- **2026-07-23 (claude-code)**: first attempt blocked — this dev
  environment had no C compiler and no admin rights to install one via
  chocolatey. Unblocked in the same session by downloading a portable,
  no-installer mingw-w64 build (winlibs.com/GitHub releases, plain zip
  extract, no admin needed) to a session-scratch directory and prepending
  its `bin/` to `PATH`. **This is not a permanent fixture of the dev
  environment** — it lives outside the repo and outside any durable
  machine state, so a future session/agent on a machine without a C
  compiler will hit the same block and needs to repeat this (or have a
  proper mingw-w64 install with admin rights, or work from Linux with
  WebKitGTK dev headers instead).
- Implemented on `github.com/webview/webview_go`. Found and fixed a real
  cross-thread bug in that library while writing the first integration
  test: its Windows backend's `Terminate()` is a bare `PostQuitMessage(0)`,
  which is thread-local in Win32 and silently does nothing when called
  from a goroutine other than the one running the message loop — despite
  the library's own doc comment claiming `Terminate` is safe to call from
  another thread (true for its GTK backend, which internally dispatches;
  not true for Win32). Worked around by routing `Terminate` through
  `Dispatch` (which correctly marshals via `PostMessageW` to the
  webview's message-only window) instead of calling it directly. See the
  stage audit for the full account.

### 2.4 VNC
- Build on the most complete existing Go VNC client library; fill gaps
  (encodings, auth methods) upstream-style in a vendored fork if needed —
  justify in the stage audit.

### 2.5 RDP — external process only
- Launch `xfreerdp` (Linux) / FreeRDP or `mstsc` (Windows) and reparent its
  window using the mechanism validated in Phase 0.
- v1 scope: single-monitor session, no device redirection (disks, printers,
  clipboard) — controlled only via CLI flags and `.rdp` files; redirection
  is v2 backlog.
- **Never link libfreerdp** (GPLv2 vs Apache-2.0).
- **2026-07-23 (claude-code)**: implemented as `protocol.WindowProtocol`
  (same interface stage 2.3 introduced). Windows: launches
  `sdl-freerdp.exe` (not `mstsc` — the spike found it an unreliable
  embedding target, kept only as documented fallback), then locates its
  session window with the spike's find-and-adopt retry loop and reparents
  it with the exact validated recipe (`EmbedChild` in `embed_windows.go`:
  `SetParent` → restyle → `SetWindowPos(FRAMECHANGED)` →
  `GetAncestor`-verify). Genuinely tested end-to-end — including the
  DPI-awareness dance — using a self-created parent window and
  `mspaint.exe` as an external-process stand-in for FreeRDP (Notepad
  doesn't work as a test target on this Windows 11 install: it's now an
  MSIX-packaged app whose launched pid isn't the one owning the window;
  `mspaint.exe` was checked and confirmed not redirected before using it).
  Linux: launches `xfreerdp` and tracks its lifecycle, but window
  discovery/embedding is **not implemented** — this session has no X
  server, no `xfreerdp` binary, and no way to validate `xgb`-based
  EWMH-parsing code, so `NativeWindowHandle()` returns 0 on Linux rather
  than shipping unverified protocol code. See the stage audit for the
  full account and pending actions.

### 2.6 PowerShell remoting
- WinRM via existing Go libraries; lowest priority of the phase.

### 2.7 AnyDesk
- Same external-process + reparent pattern as RDP (proprietary protocol).

## Exit criteria
Stages done with audits; a demo config file connects successfully over SSH,
VNC and RDP on both platforms; top-level README updated.
