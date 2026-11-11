# Linux preview — active delivery plan

Authorized by the user on 2026-11-08: prioritize standalone Linux functionality;
C# compatibility is optional and does not block this track. This table overrides
historical phase dependencies. Existing completed stages and audits are retained.

| Stage | Scope | Status | Owner |
|---|---|---|---|
| L1 | mRemoteNG-style profile manager (tree, tabs, Config panel) | in progress (opencode; human usability pending) | opencode |
| L2 | SSH/RDP session lifecycle and real connections | in progress (opencode; SSH terminal, embedded RDP and VNC done; modifier-key input, embedded webview and serial pending) | opencode |
| L3 | Session UX, optional import, credential strategy | in progress (import/export, credential vault, quick connect, folders, terminal zoom, scrollback/copy done; see the matching backlog stages, no L3 audit yet) | opencode |
| L4 | Linux packaging and human usability acceptance | partially done (`.deb` via `scripts/package-deb.sh`; rpm/Flatpak and human acceptance pending) | human + any |

## Ownership and coordination

- Folders are stored explicitly (`folders` in the native document), so an empty
  folder created with File > New folder survives restarts; connections join one
  through the Folder field or a connection row's right-click Move to submenu.
  Rename rewrites the whole subtree and every contained connection's Folder
  value; delete is
  recursive and confirms the number of affected connections. Rows expose a
  right-click context menu (folder: create subfolder in one click, rename,
  delete; connection: rename, open, Move to submenu listing the root and every
  folder, delete). Rename dialogs are compact fixed-size popups with a single
  text field; an on-screen keyboard was prototyped and removed because it was
  not needed. A
  connection can also be dragged onto a folder row to move it there, or dropped
  elsewhere to return it to the root. The drag has no ghost or drop highlight
  yet.
- The connection tree is deliberately narrow (15% split) and collapsible from a
  flat icon in the status bar (the second icon toggles the Config panel); there
  is no toolbar row, and the session tab keeps a single connection bar so the
  terminal owns the space. Rows start with a circular SSH/RDP badge and report
  the full name and endpoint in the status bar on hover (Fyne's tree consumes row
  hover, so a floating popup is not used). The Config panel is hidden by default
  and revealed by New/selection or the settings icon.
- L1 owns `internal/profiles/`, `internal/ui/`, `cmd/mremoteng/`, build/test
  integration and the active-plan documentation. No existing core package changes.
- Independent L1 units may be delegated within disjoint files: profile storage,
  desktop UI, and entrypoint/build wiring. The lead integrates and audits L1.
- L2 owns `internal/protocol/{ssh,rdp,vnc,telnet}` and `internal/terminal/`; UI wiring in
  `internal/ui/` lands as a scoped change. Packages added after L4
  (`internal/vault`, `internal/transfer`) belong to their backlog stage.
- Other phase work must not overlap the active track. Stage closure still needs
  checks, smoke, audit and changelog; manual acceptance is recorded separately.
- Delegation of this Linux implementation is authorized by the current user
  request; no need to re-ask for each bounded implementation unit.

## L2 partial delivery

SSH is implemented in `internal/protocol/ssh` and exposed in the session tab:

- Connectivity test: handshake, known_hosts verification, agent/key/password
  auth and a session-channel check, without running commands.
- Interactive shell: PTY request, shell start, stdin/stdout streaming and
  `window-change` on resize. Output is rendered by `internal/terminal` (VT100/
  xterm emulation, 256 colours, cursor) inside the tab.
- Terminal history: `internal/terminal` keeps up to 2000 scrolled lines; the
  wheel and PageUp/arrow keys browse them while the view is scrolled up, and a
  mouse selection is copied to the clipboard by right-click (there is no
  Ctrl-Shift-C, because key events carry no modifier information). Zoom is per
  terminal via Ctrl+= / Ctrl+- / Ctrl+0 or View > Terminal zoom; the wheel always
  browses the scrollback. Wheel zoom was tried but removed: Fyne scroll events
  carry no modifiers, and asking the X server for the Ctrl state misreported it
  on Wayland/XWayland, so every wheel event zoomed instead of scrolling.
- Unknown hosts require explicit approval of the shown SHA-256 fingerprint; the
  password lives only in the widget for that attempt.

RDP is implemented in `internal/protocol/rdp`: FreeRDP is launched as an
external process (`xfreerdp3`/`xfreerdp`/`wlfreerdp`), the password is written to
its stdin (`/from-stdin:force`), host certificates use `/cert:tofu`, and
"Test credentials" runs `+auth-only` so a login can be verified without a
desktop, and the "Legacy RDP security" checkbox passes `/sec:rdp` for servers
without TLS (explicit, never automatic). The session is embedded in the tab:
`internal/ui/embed.go` creates an X11 child window inside the Fyne window, keeps
it aligned with the tab area (250 ms geometry poll, since Fyne has no resize
callback), applies the display scale (X11 pixels vs Fyne points) and clamps the
geometry to the visible tab bounds. It also resizes the window FreeRDP creates
inside that area, because the client requests its own 1024x768 regardless: with
`+smart-sizing` this is what makes the session fit the tab and follow window
resizes instead of spilling over the sidebar, Config panel and status bar. Without an X11/XWayland display the
panel falls back to a separate window and reports it.

Verified end to end against `scottyhardy/docker-remote-desktop` (xrdp 0.9.24)
with a full session: FreeRDP stayed connected for the whole test window and the
session opened. Auth-only is not supported by that server, so "Test credentials"
reports failure while Connect works; the status message says so. The `RDP_LOCAL`
test documents how to repeat the auth-only check against a compatible host.

Open: modifier-key input (Fyne key events carry no modifier information, so
Ctrl-C/Ctrl-D are buttons and Shift-Tab is indistinguishable from Tab; only the
control combinations registered in `internal/ui/shell.go` reach the shell), the
embedded webview (HTTP profiles open in the system browser, see
`internal/ui/stream.go`), and serial ports. Tab is delivered to the shell via
`fyne.Tabbable` so completion works.

## Audit naming for this track

Audits use `auditory/linux-l<stage>-<YYYYMMDD>-<agent>.md`, because the
historical `phase<N>-stage<M>` scheme does not describe the Linux track.

## L1 evidence

- Automated: profile store tests (round trip, permissions, corrupt/unsupported
  files, validation) and headless UI tests (save, validation errors, store
  failures, filtering, tree selection, delete confirmation, discard guard,
  write lockout after a load failure).
- Manual: `docs/linux-preview-usability.md` is pending human execution; the
  stage stays open until that evidence exists. The script now also covers what
  shipped after the first draft (folders, import/export, quick connect, terminal
  zoom, scrollback and copy, credential vault), so one run accepts the product
  as it stands.

## Next stages

`blueprint/backlog.md` holds the ordered plan after L4 (import/export,
inheritance, RDP parity, extra protocols, credentials, session UX, reliability
and packaging) and records which of those stages are already partially
delivered. Read it before proposing work outside this table.

See `docs/MIGRATION_PLAN.md` for acceptance criteria and deferred work.
