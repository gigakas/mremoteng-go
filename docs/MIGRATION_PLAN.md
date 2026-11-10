# Linux-first standalone application plan

## Product decision

The user authorized a standalone Linux-first direction on 2026-11-08.
C#/.NET, Windows availability and complete mRemoteNG parity are not prerequisites
for development or Linux preview acceptance. Existing Go serializers, security
code and compatibility fixtures remain supported regression coverage; do not
claim their incomplete C# corpus is complete. No license change is implied.

## Architecture

- Go + Fyne desktop application, fixed connection list/editor layout initially.
- Native Linux profiles use a versioned JSON file in the user configuration
  directory, atomic replacement and owner-only permissions. The first increment
  stores endpoint metadata only, never passwords, tokens or private keys.
- Existing XML/CSV serializers remain optional interoperability adapters, not
  the native storage contract. Import must not overwrite source files.
- SSH and RDP come next. RDP remains an external FreeRDP process; never link
  libfreerdp. Embedding uses X11/XWayland, not native Wayland (see spike findings).
- Windows, other protocols, full inheritance editing, vaults, docking and
  distribution packages are deferred until Linux workflows are validated.

## Delivery sequence and acceptance

The active stage table is in `blueprint/linux-preview.md` and supersedes the
ordering and release gates in the historical phase files.

1. **L1 — Local profile manager:** runnable desktop with the mRemoteNG layout
   (menu, toolbar, connection tree with folders, session tabs, Config panel,
   status bar), Welcome tab, create/edit/delete/filter of SSH and RDP profiles,
   input validation, durable local storage, visible errors and no pretend
   Connect action. Test model, persistence and UI actions; provide a Linux
   build and manual usability steps (`docs/linux-preview-usability.md`).
2. **L2 — Real sessions:** protocol lifecycle, SSH host-key verification and
   external FreeRDP, cancellation/exit/error handling. No passwords on command
   lines, silent certificate bypass or execution of imported shell commands.
   Local test servers plus explicit human-authorized remote endpoints are required
   before claiming end-to-end connection success. Partial delivery: the SSH probe
   (handshake, known_hosts verification, agent/key/password auth, session channel)
   is implemented and tested against an in-process server; the interactive
   terminal and RDP embedding remain open.
3. **L3 — Usable preview:** optional XML import, encrypted credential strategy,
   session tabs, shortcuts, unsaved-change handling and accessibility review.
4. **L4 — Linux release candidate:** repeatable packaging, upgrade/recovery tests,
   X11 and GNOME/KDE XWayland coverage, human usability acceptance and limitations.
5. **Later:** Windows and additional protocols based on actual demand.

## Verification and honest status

Run `scripts/check.sh` and `scripts/smoke.sh` for each delivery. Keep automated
headless UI tests distinct from real desktop usability and remote-host tests.
A preview may be delivered with documented manual validation pending; do not
mark its human acceptance done until evidence exists. Existing C# corpus tests
remain in the suite but are no longer a release gate for native Linux profiles.
