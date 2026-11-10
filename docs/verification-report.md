# Verification report — Linux preview 0.4.0

Date: 2026-11-09. Scope: what is verified automatically, what was verified against
real services on this machine, and what still needs a human.

## Automatic (run with `./scripts/check.sh`)

- Profile model: validation, folder paths, inheritance resolution, store schema v2
  with v1 migration, atomic writes and owner-only permissions.
- UI (Fyne software driver): tree/menu/tab behaviour, folder create/rename/delete/
  move, drag target hit-testing, quick connect, import merge/replace, vault
  create/store/use/lock, collapsed-folder restore, diagnostics text, repair flow,
  terminal screen extraction, session shutdown.
- Import/export: XML 2.6/2.7/2.8 and CSV mapping, password dropping, round trips.
- SSH: host key verification, auth methods, PTY shell, resize (in-process server).
- RDP: argument builder (redirection, certificate, gateway, legacy security),
  password over stdin, auth-only probe harness.
- VNC: RFB handshake, None and VNC auth, Raw framebuffer decode, pointer/key
  forwarding (fake server).
- Telnet/RAW/rlogin: IAC negotiation, escaped IAC, rlogin preamble, byte pass
  through.
- Vault: PBKDF2+AES-GCM round trip, tamper detection, permissions, lock semantics.

## Verified against real services here

- SSH: `lscr.io/linuxserver/openssh-server` container — handshake, auth, PTY shell
  and command output (`echo container-ok`) succeeded.
- RDP: `scottyhardy/docker-remote-desktop` (xrdp 0.9.24) — full FreeRDP session
  stayed connected; embedded X11 child window tracked the tab geometry and
  rescaled with the window (child and area sizes matched exactly).
- X11 embedding: window class, geometry clamping and display scale verified with a
  diagnostic build; software-driver captures reviewed for layout.

## Needs a human (cannot be automated here)

1. Visual check of the real window: menu bar, taskbar icon, dialogs and the
   embedded RDP session on your desktop.
2. RDP parity against a real Windows host: clipboard, drives, printers, audio,
   RD Gateway.
3. VNC and Telnet/rlogin against real servers.
4. `sudo dpkg -i dist/mremoteng-go_0.4.0~linux~preview_amd64.deb` to confirm the
   packaged entry, icon and uninstall.
5. Usability walkthrough per `docs/linux-preview-usability.md`, including the new
   folder, import and credential-vault flows.

## Known gaps at this commit

- No terminal scrollback (only the visible screen can be copied).
- Session state is not restored after a restart (only collapsed folders are).
- HTTP opens the system browser; an embedded webview is not implemented.
- Serial ports are not implemented; RD Gateway flags exist without UI fields.
- Flatpak/rpm and the Windows port are not built.
