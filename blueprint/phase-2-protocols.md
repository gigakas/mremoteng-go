# Phase 2 — Protocols (increasing risk order)

**Goal**: working protocol implementations behind a common lifecycle
interface (Connect/Disconnect/Focus, mirroring the original `ProtocolBase`).
**Depends on**: Phase 1 (data core) for all stages; Phase 0 (spike) for
stages 2.5 and 2.7.

**Owned packages**: `internal/protocol/` and its per-protocol subpackages.

## Stages

| # | Stage | Status |
|---|---|---|
| 2.1 | Protocol interface + factory | pending |
| 2.2 | SSH, Telnet, rlogin, raw socket | pending |
| 2.3 | HTTP/HTTPS (native webview) | pending |
| 2.4 | VNC | pending |
| 2.5 | RDP (external xfreerdp + reparent) | pending |
| 2.6 | PowerShell remoting (WinRM) | pending |
| 2.7 | AnyDesk (external process) | pending |

### 2.1 Protocol interface + factory
- `Protocol` interface: Connect/Disconnect/Focus/Resize lifecycle + error
  and close events. Single construction point keyed on the protocol type
  (mirrors `ProtocolFactory.cs`).
- Tests with a fake protocol implementation.

### 2.2 SSH, Telnet, rlogin, raw socket
- SSH via `golang.org/x/crypto/ssh`; Telnet/rlogin/raw as thin custom
  implementations. Rendering into an embedded terminal emulator widget.
- Tests: connection lifecycle against local in-process servers.

### 2.3 HTTP/HTTPS
- OS-native webview (WebView2 on Windows, WebKitGTK on Linux) via a thin
  wrapper. This is the only place where cgo is acceptable, and only through
  the wrapper library.

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

### 2.6 PowerShell remoting
- WinRM via existing Go libraries; lowest priority of the phase.

### 2.7 AnyDesk
- Same external-process + reparent pattern as RDP (proprietary protocol).

## Exit criteria
Stages done with audits; a demo config file connects successfully over SSH,
VNC and RDP on both platforms; top-level README updated.
