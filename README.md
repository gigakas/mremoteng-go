# mRemoteNG Go

A Go implementation of the [mRemoteNG](https://github.com/mRemoteNG/mRemoteNG)
connection manager, built Linux-first. The window follows the original layout:
menu bar, toolbar, connection tree, session tabs, a Config panel and a status
bar — implemented with Go and Fyne, with no .NET or C# dependency.

The window uses a restrained dark palette with a light variant, compact type and
one accent colour for the primary action; the tree has its own header and the
Config panel is titled *Connection details*.

**Status: Linux preview (L1).** You can create, edit, organise, filter and
delete connection profiles and they persist locally. Session tabs exist and
show the saved connection details, but **opening SSH or RDP sessions is not
implemented yet** — that is stage L2 in [`blueprint/linux-preview.md`](blueprint/linux-preview.md).

## What works today

- Narrow (15%), collapsible connection tree with folders (`Production/Web` style
  paths), filtering and expand/collapse (flat icons in the status bar, or the
  View menu), plus a Welcome tab and one tab per opened connection. Folders can be
  created (File > New folder, persisted even while empty), **renamed** (whole
  subtree, with the Folder value of every contained connection rewritten) and
  **deleted** (with confirmation; contained connections and their tabs go too).
  Right-clicking a connection row opens a context menu whose **Move to** submenu
  lists the root and every folder, so moving a connection into a folder — or back
  to the root — needs clicks only. Each row
  starts with a circular SSH/RDP badge; hovering reports the full name and
  endpoint in the status bar. The footer shows `connections / folders`.
- Config panel with name, folder, protocol (SSH/RDP), host, port and username;
  Save and Revert, with Delete attached to the connection name (confirmed). It is
  hidden until you create or select a connection; the settings icon in the status
  bar toggles it at any time.
- Validation that rejects missing or ambiguous values (including `host:port`,
  `ssh://` prefixes and `-`-prefixed fields, which would be unsafe when passed
  to an external client later).
- **Credential vault** (File > Credentials): optional encrypted store for session
  passwords, unlocked with a master password (PBKDF2-SHA256 210k + AES-256-GCM,
  atomic `0600` file). A session uses the typed password, or the stored secret
  when the field is empty; storing is an explicit action on the session bar.
- **Quick connect**: an ad-hoc SSH/RDP/VNC session (sidebar + icon or File menu)
  that never writes a profile; the status bar says it is temporary.
- **Activity log**: every status and error message is timestamped and kept in a
  collapsible console (bounded to 200 entries) with its own status-bar icon and a
  Clear action.
- **Terminal zoom**: Ctrl+= / Ctrl+- / Ctrl+0, or View > Terminal zoom, change the
  console font size (8-28) per tab without touching the rest of the window. The
  mouse wheel keeps scrolling the history: per-event theming made a Ctrl+wheel
  zoom flicker and reset itself.
- **Terminal**: scrollback (2000 lines) with the mouse wheel (dragging past an
  edge auto-scrolls), PageUp/PageDown and the arrows once scrolled (at the bottom
  the arrows keep their shell meaning; Home/End jump in the history), drag to select,
  double-click for a line, right-click copies the selection, and the copy icon
  copies the whole buffer when nothing is selected. Ctrl-C interrupts the remote
  process, Ctrl-V pastes, and the other Ctrl combinations (A, D, E, K, L, R, U,
  W, Z, X…) reach the shell as control bytes even though Fyne does not report
  modifiers. Right-click copies.
- **Diagnostics** (Tools menu): version, store path, counts, vault state and open
  sessions, with a copy-to-clipboard action; closing the app ends every live
  session and embedded window first.
- **Workspace memory**: window size/position, collapsed folders and the profile
  document survive a restart.
- **Folder defaults**: a folder can share a username and port; connections opt in
  with "From folder" toggles in the Config panel, and sessions use the resolved
  values. Stored in the versioned JSON document (schema 2, version 1 migrates on
  load).
- **Import/export**: mRemoteNG XML (2.6/2.7/2.8) and CSV, with a preview showing
  connections, folders, skipped rows and dropped passwords, then Merge or Replace.
  The source file is only read; export writes XML (optionally password-protected)
  or CSV.
- Local storage in `~/.config/mremoteng-go/profiles.json` (respects
  `XDG_CONFIG_HOME`), written atomically with `0600` permissions. A file that
  cannot be read is never overwritten: writes stay disabled until it is fixed.
- Unsaved changes are protected when switching connections, starting a new one
  or closing the window.
- SSH session tab: test the handshake, or Connect for an interactive shell.
  Agent and default key files are tried first, then an optional password used for
  that attempt only (never stored). Host keys come from `~/.ssh/known_hosts`; an
  unknown host shows its SHA-256 fingerprint and needs explicit approval.

## Known limitations

- SSH is **interactive**: a single connection bar (name, endpoint, password,
  Connect/Disconnect and Ctrl-C/Ctrl-D) sits above a pseudo-terminal shell
  rendered by a VT100/xterm emulator, with keyboard input, 256-colour output and
  window resizing. The Config panel hides while a session
  is active (and returns on disconnect) so the terminal gets the full height.
  Tab reaches the shell for completion instead of moving widget focus.
  Ctrl/Alt combinations are not exposed by the Fyne key API yet, so Ctrl-C and
  Ctrl-D have dedicated buttons; Shift-Tab arrives as Tab.
- RDP launches FreeRDP (`xfreerdp3`/`xfreerdp`/`wlfreerdp`) as an external process
  embedded in the session tab through `/parent-window` on a dedicated X11 window
  that follows the tab's geometry, so expanding or contracting the application
  rescales the session. FreeRDP asks for its own window size, so the area also
  resizes that child window: together with `+smart-sizing` the session always
  matches the tab and never spills over the rest of the window. Embedding needs
  the `x11` build tag (used by `make dev`/`scripts/dev.sh`); without an X11/XWayland display the panel falls
  back to a separate window and says so. Certificates default to `/cert:tofu`;
  enable **Ignore certificate** when a server presents a changed or self-signed
  one.
  "Test credentials" uses FreeRDP's auth-only mode, which needs a server that
  supports it (xrdp does not; Connect still works there). "Legacy RDP security" opts into `/sec:rdp` for old servers without
  TLS. Passwords travel through stdin, never argv.
- **VNC**: native RFB client (None and VNC authentication, Raw encoding,
  framebuffer view, click forwarding). Keyboard and clipboard transfer are not
  wired yet.
- **Telnet, RAW and rlogin** run in the terminal view (no remote pty: full-screen
  programs may misrender); **HTTP** opens the address in the system browser
  because embedding a webview in the window is not implemented yet. Serial ports
  are still pending.
- Passwords and keys are not stored at all.
- Folder operations are menu-driven: select the folder (rename/delete) or the
  connection or folder, or use the **right-click context menu** on a row. A
  connection's menu has **Move to** with a submenu listing the root and every
  folder, so moving needs clicks only (no dialog). Folder rows create a
  uniquely named subfolder in one click. Renaming a folder or a connection opens a
  compact single-field dialog with Create/Rename and Cancel. Connections can also be **dragged onto a folder row**
  (dropping elsewhere returns them to the root); there is no drag preview yet.
- Windows and packaging are deferred; RDP will stay an external FreeRDP process
  (never linked), reached through X11/XWayland.

## Build and run

Requires Go 1.23+, a C toolchain and the Fyne Linux dependencies
(`libgl1-mesa-dev`, `xorg-dev`, `libxkbcommon-dev` and friends; Fyne uses cgo
for the window layer).

```bash
make dev                              # rebuild, replace the running instance, start it
./scripts/dev.sh /tmp/try.json        # same with a scratch profile store
make run                              # build and run in the foreground
```

`scripts/dev.sh` stamps the window title and About dialog with the version,
commit and build time, so it is always clear which build is on screen. It also
notifies the desktop and prints the new PID.

## Release and installation

```bash
make release            # dist/mremoteng, desktop files, tarball and sha256
make install            # same, then install into ~/.local (PREFIX overrides)
./scripts/package-deb.sh   # build dist/mremoteng-go_<version>_<arch>.deb
./scripts/release.sh --install --run   # install and start the installed app
```

The release build uses `-tags x11 -trimpath -ldflags "-s -w"` and stamps version,
commit and build time from `VERSION`. Installation copies the binary to
`$PREFIX/bin/mremoteng`, the scalable icon to
`$PREFIX/share/icons/hicolor/scalable/apps/` and a `.desktop` entry whose `Exec`
points at the installed binary, so the application appears in the desktop
application grid with its icon. The window sets a stable `WM_CLASS`
(`mRemoteNG Go`) so the running window matches that entry.

## Development

```bash
./scripts/check.sh    # gofmt check, go vet, all tests (includes the headless UI tests)
./scripts/smoke.sh    # build all binaries, -version check, changelog reproducibility
make changelog        # regenerate CHANGELOG.md from changelog.d/
```

The UI tests use Fyne's software test driver, so they run without a display.
The desktop window itself needs a session; on Wayland it runs through XWayland.

## Layout

- `cmd/mremoteng` — desktop entry point (`-profiles`, `-version`).
- `internal/profiles` — native profile model, validation and JSON store.
- `internal/ui` — Fyne shell: menu, toolbar, tree, tabs, Config panel, status bar,
  SSH test panel.
- `internal/protocol/ssh` — SSH probe and interactive shell: known_hosts
  verification, agent/key/password authentication, PTY request and resize.
- `internal/protocol/rdp` — FreeRDP launcher: argument building, credentials via
  stdin, `/parent-window` support, auth-only probing and an opt-in legacy
  security mode, plus per-session redirection options (clipboard, printers,
  audio, shared drives, dynamic resolution, multi-monitor). `RDP_LOCAL` (see
  `rdp_test.go`) runs a real-server probe.
- `internal/ui/embed.go` — X11 child window that hosts the RDP session and tracks
  the tab geometry (250 ms poll; Fyne has no resize callback).
- `internal/terminal` — VT100/xterm emulator snapshot used by the tab view.
- `internal/connection`, `internal/serialize/{xml,csv}`, `internal/security` —
  mRemoteNG data-core compatibility (model, XML 2.6/2.7/2.8, CSV, encryption).
  These remain as regression coverage and as the base for optional import; the
  native app does not require them.
- `blueprint/` — active delivery plan (`linux-preview.md`) and historical phases.
- `auditory/` — stage-closing audits.
- `changelog.d/` — changelog fragments (source of the generated `CHANGELOG.md`).

## Manual usability

[`docs/linux-preview-usability.md`](docs/linux-preview-usability.md) is the
checklist for trying a build: create, validate, persist, filter, edit, delete,
toggle the panel and corrupt-file safety.
