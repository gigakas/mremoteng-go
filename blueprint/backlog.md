# Backlog — planned improvements after L4

Ordered plan for the Linux application, derived from the gap analysis against
mRemoteNG. Each stage follows the normal closing rules: `scripts/check.sh`,
`scripts/smoke.sh`, audit in `auditory/`, changelog fragment, status update in
`blueprint/linux-preview.md`, and human acceptance where the stage changes what
the user sees.

## Principles kept from the existing plan

- Linux first; Windows and packaging of other platforms stay in L12.
- RDP/AnyDesk remain external processes (never linked).
- Credentials are never stored silently; any persistence is an explicit,
  documented, encrypted design.
- A feature is only "done" with automated tests plus a recorded manual check.

## Current state (baseline for planning)

Implemented: native JSON profile store with folders, tree with badges, tabbed
shell, SSH (known_hosts, agent/keys/password, PTY + terminal emulator, Tab
completion, resize), RDP (external FreeRDP embedded via `/parent-window`,
smart-sizing, certificate policy, legacy security), search/filter, context menu,
drag to move, compact dialogs, themes, window geometry memory, app icon, local
installer and tarball release.

## Stage L5 — mRemoteNG file import and export — **done (2026-11-09)**, audit `auditory/linux-l5-20261109-opencode.md`

- **Goal**: bring existing user data in without touching the source file.
- **Scope**: File > Import (XML 2.6/2.7/2.8 and CSV) and File > Export; reuse
  `internal/serialize/{xml,csv}`; show a preview (counts, folders, conflicts);
  choose "merge" or "replace"; never overwrite the imported file; decrypt with a
  password prompt when needed and keep the password out of logs.
- **Acceptance**: importing the compatibility corpus produces the expected tree;
  conflicts are reported, not silently resolved; export round-trips through the
  Go serializers; tests cover merge/replace/cancel and decryption failure.
- **Risks**: mapping inheritance-bearing XML into the flat native store; document
  the mapping (values copied, inheritance flags preserved only as effective
  values) or deliver L6 first.
- **Depends on**: nothing.

## Stage L6 — Folder-level defaults and inheritance — **done (2026-11-09)**, audit `auditory/linux-l6-20261109-opencode.md` (username + port)

- **Goal**: stop repeating username/domain/port per connection.
- **Scope**: choose between (a) folder records with default values plus per-field
  inherit toggles in the Config panel, or (b) importing the full
  `InheritanceFlags` model from `internal/connection`. Show effective values in
  the form (placeholder) and persist the choice; document the decision in the
  audit.
- **Acceptance**: multi-level inheritance resolves correctly in the UI and in
  what the session receives; toggles persist; tests mirror
  `internal/connection/inheritance_test.go` cases; no regression for flat
  profiles.
- **Depends on**: L5 for imported data to be meaningful; decide the data model first.

## Stage L7 — RDP parity — **partially done (2026-11-09)**: redirection/display flags and options popup; RD Gateway and options persistence pending. Audit `auditory/linux-l7-20261109-opencode.md`.

- **Goal**: make embedded RDP usable for daily work.
- **Scope**: clipboard redirection, drive/printer/audio redirection, fullscreen
  and "send special keys", opt-in `/dynamic-resolution` per host, multi-monitor
  as an explicit option, RD Gateway fields wired to FreeRDP flags, and a readable
  failure report (certificate, credentials, transport) in the status bar.
- **Acceptance**: each flag has a unit test on the argument builder and one manual
  session check against a real host; no silent downgrades; documented server
  compatibility notes (xrdp vs Windows).
- **Depends on**: L7 needs a reachable RDP host for acceptance; embedding already
  works.

## Stage L8 — Additional protocols — **partially done (2026-11-09)**: VNC client + view (`auditory/linux-l8-vnc-20261109-opencode.md`) and Telnet/RAW/rlogin + HTTP handoff (`auditory/linux-l8-streams-20261109-opencode.md`); embedded webview and serial pending.

- **Goal**: cover what mRemoteNG users actually connect to after RDP/SSH.
- **Scope, in risk order**: VNC (existing Go client, fill auth/encoding gaps),
  HTTP/HTTPS via native webview (the only allowed cgo wrapper), Telnet/RAW/rlogin
  on top of the existing terminal widget, serial as a lower-priority extra.
- **Acceptance**: per protocol, a lifecycle test against a local server plus a
  manual session; VNC and webview get their own audit entries for dependency and
  cgo decisions.
- **Depends on**: L5/L6 not required; can run in parallel with L7 by one agent per
  protocol package.

## Stage L9 — Credentials and secrets — **partially done (2026-11-09)**: encrypted vault with master password (audit `auditory/linux-l9-20261109-opencode.md`); external providers pending.

- **Goal**: optional convenience without weakening the current safety.
- **Scope**: encrypted credential store unlocked with a master password (document
  the KDF and file format), per-connection opt-in, external providers
  (Vault/OpenBao, 1Password, Delinea) behind a small interface, and a clear
  fallback when a provider is unavailable.
- **Acceptance**: credentials never appear in argv, logs or the plain JSON store;
  wrong master password leaves data intact; provider clients tested against a
  mock; threat model section in the audit.
- **Depends on**: L6 (fields to fill) and a decision on whether to reuse the
  mRemoteNG XML encryption or a native scheme.

## Stage L10 — Session UX and workspace state — **partially done (2026-11-09)**: Quick connect (`auditory/linux-l10-20261109-opencode.md`) and persisted collapsed folders (`auditory/linux-l10-workspace-20261109-opencode.md`); visible-screen copy and repair path done (`auditory/linux-l10-copy-20261109-opencode.md`); terminal zoom, scrollback and copy done; session persistence, settings dialog and manual ordering pending.

- **Scope**: terminal scrollback and selection/copy, session persistence and
  reconnect, quick-connect bar, notifications panel, settings dialog (default
  protocol/ports, theme choice), persisted expanded state and manual ordering.
- **Acceptance**: workspace state restores after restart; scrollback is bounded;
  settings validation tested; manual usability script extended.
- **Depends on**: L7/L8 for meaningful reconnect behaviour.

## Stage L11 — Reliability and diagnostics — **partially done (2026-11-09)**: diagnostics dialog and session shutdown on exit (audit `auditory/linux-l11-20261109-opencode.md`); store repair path pending.

- **Scope**: structured logging with a "copy diagnostics" action, guarded store
  migration path (schema v2), crash-safe session cleanup (X11 children, child
  processes), and a recovery flow for a corrupt store.
- **Acceptance**: simulated corrupt/unreadable files produce actionable errors;
  no orphan X11 windows or FreeRDP processes after crash/close; tests for the
  migration and cleanup paths.
- **Depends on**: L5 for schema changes.

## Stage L12 — Packaging and other platforms — **partially done (2026-11-09)**: `.deb` via `scripts/package-deb.sh` (audit `auditory/linux-l12-20261109-opencode.md`); rpm/Flatpak and Windows pending.

- **Scope**: `.deb`/`.rpm`/AppImage or Flatpak with the icon and `.desktop`
  entry, update notes, then the Windows port (Win32 `SetParent`, FreeRDP/MSRDC
  runtime dependency) as a separate track.
- **Acceptance**: clean install/uninstall on a fresh VM, desktop entry with icon,
  documented runtime dependencies (`xfreerdp3`), no root-only paths.
- **Depends on**: L7-L10 stable enough to package.

## Suggested execution order

1. L5 (import/export) — highest value per effort, reuses tested serializers.
2. L6 (inheritance) — makes imported data and daily editing practical.
3. L7 (RDP parity) — closes the biggest gap for daily use.
4. L10 (session UX) — small items with high perceived quality.
5. L8 (protocols), L9 (credentials) — larger, can be parallelised per package.
6. L11 (reliability), L12 (packaging/Windows).

## Explicitly out of scope

- Docking/floating panels (documented v1 layout decision) unless users ask.
- Linking FreeRDP or shipping GPL clients inside the binary.
- Storing passwords without an explicit, encrypted, user-unlocked mechanism.
