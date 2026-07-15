# Migration plan: mRemoteNG (C#/WinForms) → Go + FreeRDP (external process)

Source: analysis and planning done against the original repository at
`../mRemoteNG`. This document is the phase reference; the operational
per-stage detail lives in `blueprint/`. Do not duplicate its content in
other documents — link here instead.

## Non-negotiable architectural principle

RDP and AnyDesk are integrated **only as external processes embedded by
window reparenting**, never by linking code (no cgo against `libfreerdp` or
similar). The original mRemoteNG is GPLv2; FreeRDP is Apache-2.0
(incompatible for direct linking according to the FSF). For window
embedding:

- Linux: `github.com/BurntSushi/xgb` (pure-Go X11 protocol, no cgo, no
  libX11 linking).
- Windows: `golang.org/x/sys/windows` (`SetParent`/`SetWindowLong` via
  syscall).

`xfreerdp` is a *runtime* dependency (like `PuTTYNG.exe` in the original
project), never a *build* dependency.

## Phase 0 — Validation spike (2–3 weeks)

- Fyne prototype + `xfreerdp` window reparenting on X11.
- Same prototype on Windows with `SetParent`.
- Validate behavior under Wayland (expected: requires XWayland).
- Exit criterion: if reparenting is unreliable on modern GNOME/KDE, decide
  here whether the limitation is acceptable before continuing.

## Phase 1 — Data core (model parity, no UI or protocols)

Mapping from the original C#:

| Original C# | Go target |
|---|---|
| `Connection/AbstractConnectionRecord.cs`, `ConnectionInfo.cs` | `internal/connection` |
| `Connection/ConnectionInfoInheritance.cs` | Inheritance resolution without runtime reflection; use `go:generate` or an explicit type switch |
| `Container/ContainerInfo.cs` | `internal/connection` (homogeneous tree) |
| `Config/Serializers/ConnectionSerializers/Xml/*` (v26/27/28) | `internal/serialize/xml` (same versioned pattern keyed on `ConfVersion`) |
| `Config/Serializers/ConnectionSerializers/Csv/*` | `internal/serialize/csv` |
| `Security/SymmetricEncryption/AeadCryptographyProvider.cs` | `internal/security` (stdlib `crypto/cipher`, AES-GCM) |
| `Security/KeyDerivation/Pkcs5S2KeyGenerator.cs` | `internal/security` (`golang.org/x/crypto/pbkdf2`) |
| `LegacyRijndaelCryptographyProvider.cs` | `internal/security` (`crypto/aes` CBC, legacy read-only) |

**Critical acceptance test**: generate connection `.xml` files with the
original C# app (several `ConfVersion` values, encrypted and plain) and
verify the Go port produces identical results when decrypting/reading.
Blocking for Phase 2.

## Phase 2 — Protocols (increasing risk order)

1. SSH / Telnet / rlogin / raw socket — native Go (`golang.org/x/crypto/ssh`).
2. HTTP/HTTPS — OS-native webview (`github.com/webview/webview_go`).
3. VNC — build on an existing base library (gap-filling work).
4. RDP — external `xfreerdp`/`wlfreerdp` process + reparent. v1 without
   disk/printer/clipboard redirection (v2 backlog).
5. PowerShell remoting — WinRM via existing Go libraries.
6. AnyDesk — same pattern as RDP (proprietary protocol).

## Phase 3 — UI

- Toolkit: Fyne for the connection tree, tabs, dialogs, menus.
- Panel docking: **v1 simplifies to a fixed layout** (no auto-hide/floating
  equivalent to the original `PanelBinder.cs`/`DockPanelLayoutLoader.cs`);
  revisit as v2 based on demand.
- Theming: manual reimplementation of the original `Themes/` as Fyne
  palettes.
- External credentials (original `ExternalConnectors/`: AWS, 1Password,
  Vault/OpenBao, Passwordstate, Delinea) — REST/CLI clients, port well.

## Phase 4 — Packaging

- Single Go binary per platform (`GOOS=linux/windows`, `GOARCH=amd64/arm64`).
- `xfreerdp` is not vendored: package dependency on Linux
  (`.deb`/`.rpm`/Flatpak), official binary shipped next to the portable
  `.zip` on Windows.
- Keep the original project's Stable/Preview/Nightly channel structure
  during the transition.

## Phase 5 — Migration and cutover

- Direct import of existing connection files (covered by the Phase 1 test).
- Windows registry options (original `Config/Settings/Registry/`) do not
  apply on Linux; document a config-file equivalent for enterprise
  deployments.
- Parallel coexistence as a "Preview" channel until RDP + SSH parity.
- Deprecate the C#/WinForms version only after a stable release cycle with
  real-user feedback.

## Open risks to decide early

1. Wayland: no guaranteed native support, XWayland dependency.
2. Panel docking: functionality loss in v1 (product decision).
3. Reflection-based inheritance: needs redesign, not mechanical translation.
4. Scope: this is a full rewrite — hence a separate repository, not a
   branch of the original.
