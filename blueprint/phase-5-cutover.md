# Phase 5 — Migration and cutover

**Goal**: real users move from the C# app to the Go port without losing
data, with a coexistence period.
**Depends on**: Phases 1–4.

**Owned files**: `docs/migration-guide.md`, import tooling under
`cmd/mremoteng` flags.

## Stages

| # | Stage | Status | Agent |
|---|---|---|---|
| 5.1 | Direct import of existing connection files | pending | claude-code |
| 5.2 | Settings migration guide (registry → config file) | pending | opencode |
| 5.3 | Preview channel release in parallel with the C# app | pending | human + claude-code |
| 5.4 | Feedback cycle and parity gaps triage | pending | human + claude-code |
| 5.5 | Deprecation plan for the C#/WinForms version | pending | human |

### Parallelism & collision notes

- 5.1 touches `cmd/mremoteng` flags plus the Phase 1 serializers
  (read-only imports); 5.2 owns `docs/migration-guide.md` — disjoint, so
  5.1 and 5.2 can run in parallel.
- 5.2 is documentation against a known source (`Config/Settings/Registry/`
  in the C# repo) — a good OpenCode candidate.
- 5.3–5.5 are release/governance work driven by the human with agent
  support; sequential by nature.

### Notes

- 5.1 is mostly covered by the Phase 1.7 corpus; this stage adds the user
  facing flow (open old file → works, no migration step).
- 5.2: the original Windows registry policies
  (`Config/Settings/Registry/` in the C# repo) get a documented config-file
  equivalent for enterprise deployments.
- 5.3: ship as "Preview" until RDP + SSH parity covers the majority of real
  usage; the C# app remains the stable channel meanwhile.
- 5.5 is a human/governance decision — prepare the evidence (adoption,
  open parity gaps), do not decide unilaterally.

## Exit criteria
Stages done with audits; one full stable release cycle of the Go port with
real-user feedback; top-level README updated to reflect the new status quo.
