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
