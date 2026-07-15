# Phase 3 — User interface

**Goal**: usable Fyne application: connection tree, session tabs, options,
theming, external credential sources.
**Depends on**: Phase 1 (model); integrates protocols from Phase 2 as they
land.

**Owned packages**: `internal/ui/`, `internal/credential/`.

## Stages

| # | Stage | Status |
|---|---|---|
| 3.1 | Application shell (window, menu, layout) | pending |
| 3.2 | Connection tree panel | pending |
| 3.3 | Session tabs hosting protocol views | pending |
| 3.4 | Connection properties panel (with inheritance UI) | pending |
| 3.5 | Options dialog + settings persistence | pending |
| 3.6 | Theming | pending |
| 3.7 | External credential repositories | pending |

### Notes

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
