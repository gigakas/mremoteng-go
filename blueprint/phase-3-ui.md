# Phase 3 — User interface

**Goal**: usable Fyne application: connection tree, session tabs, options,
theming, external credential sources.
**Depends on**: Phase 1 (model); integrates protocols from Phase 2 as they
land.

**Owned packages**: `internal/ui/`, `internal/credential/`.

## Stages

| # | Stage | Status | Agent |
|---|---|---|---|
| 3.1 | Application shell (window, menu, layout) | done | claude-code |
| 3.2 | Connection tree panel | done | claude-code |
| 3.3 | Session tabs hosting protocol views | done | claude-code |
| 3.4 | Connection properties panel (with inheritance UI) | done | claude-code |
| 3.5 | Options dialog + settings persistence | done | claude-code |
| 3.6 | Theming | done | claude-code |
| 3.7 | External credential repositories | done | claude-code |

### Parallelism & collision notes

- 3.1–3.6 all live in `internal/ui/` and need visual iteration on a
  desktop: they stay with claude-code, one stage at a time (3.1 first; then
  3.2/3.3 in either order; 3.4–3.6 after the shell is stable).
- 3.7 owns `internal/credential/` — REST/CLI clients with no UI, fully
  disjoint from `internal/ui/`: the one stage of this phase that can run in
  parallel on OpenCode at any point. Its UI wiring (picker in the
  properties panel) happens inside 3.4, not in 3.7.
- 3.5's settings persistence backend (config file load/save) is separable
  from its dialog; if delegated, OpenCode does the backend package and
  claude-code wires the dialog.

### Notes

- **2026-07-23 (claude-code) — visual verification is not possible in
  this dev environment.** Before writing any UI code, confirmed Fyne
  itself builds and runs (a probe window gets a real, valid Win32 handle:
  `IsWindowVisible=true`, plausible on-screen coordinates, same
  `SessionId` as the interactive session) but that window never appears
  in a screenshot taken from this session (tried `FYNE_RENDERER=software`
  too — same result). Root cause not confirmed. Raised with the user
  before proceeding; the user chose to continue on headless-only
  verification (`fyne.io/fyne/v2/test`, `scripts/check.sh`/`smoke.sh`)
  rather than pause the phase to chase the screenshot issue. This applies
  to every stage in this phase, not just 3.1 — each stage's audit repeats
  the point rather than assuming it's remembered. **Whoever can actually
  see the app should look at it** before treating any 3.1–3.6 layout as
  final.

- **v1 uses a fixed layout** (tree + tabs). No auto-hide/floating/docking
  equivalent to the original `WeifenLuo` docking — a known, communicated UX
  regression; revisit as v2 if demanded.
- 3.4 must expose per-field inheritance toggles exactly like the original
  property grid (`Inherit<Field>` flags from Phase 1.2).
- 3.5: settings in a plain config file (no Windows registry); document the
  enterprise-deployment equivalent of the original registry policies.
- 3.7 ports the `ExternalConnectors` integrations (AWS, 1Password, Delinea,
  Vault/OpenBao, Passwordstate) — mostly REST/CLI clients, standard library
  HTTP preferred.

## Exit criteria
Stages done with audits; the app manages a real connection file end-to-end
(load, edit with inheritance, connect, save); top-level README updated.

**2026-07-24 (claude-code) — met, with caveats.** All seven stages are
done with audits (`auditory/phase3-stage{1..7}-20260723|20260724-claude-code.md`).
The full chain is genuinely wired in `cmd/mremoteng/main.go`: File > Open
Connections File... (`ui.LoadConnectionsFile`, Phase 1's real
AES-256-GCM/PBKDF2 decryption) populates `ConnectionTree.SetRoot`;
selecting a node shows it in `PropertiesPanel` with working per-field
inheritance toggles (stage 3.4); selecting a connection leaf calls
`protocol.Create` (Phase 2's real backends) and opens a session tab
(stage 3.3); File > Save Connections File As... (`ui.SaveConnectionsFile`)
round-trips back through the same real encryption. This also retroactively
satisfies the runner Phase 2's own wrap-up note said was missing (see
`blueprint/phase-2-protocols.md`) — it's exactly what stage 3.5 built.

Two caveats carried forward honestly rather than glossed over:

1. **No visual verification of any Phase 3 UI was possible in this dev
   environment** (see the phase-wide note above, repeated in every
   stage's own audit) — the wiring is real and headless-tested, but no
   one has looked at the running app.
2. **No single automated test exercises the whole load → select →
   connect → save chain together** — each link is tested in isolation
   (`internal/ui`'s per-widget tests, Phase 2's per-backend tests,
   `internal/ui/connectionsfile_test.go`'s real encrypted round trip).
   Building a chain-level `integration/` test, or getting a real look at
   the running app, are the natural next steps for whoever can act on
   them — Phase 4 (Packaging) doesn't strictly need either first, but
   both would materially increase confidence before calling the UI
   done for real users.

Top-level `README.md` is updated alongside this note, since — unlike
Phase 2's wrap-up — the actual wiring this criterion asks for now exists,
not just the individual pieces.
