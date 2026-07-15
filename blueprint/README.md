# blueprint/ — Operational per-phase plan

Executable detail of the master plan (`docs/MIGRATION_PLAN.md`), split into
phases and stages. This is the source of truth for coordinating agents.

## Files

- `phase-0-spike.md` — window-embedding validation (blocking)
- `phase-1-core.md` — data model, serialization, encryption
- `phase-2-protocols.md` — SSH, Telnet, HTTP, VNC, RDP, WinRM, AnyDesk
- `phase-3-ui.md` — Fyne interface, tree, tabs, theming, external credentials
- `phase-4-packaging.md` — cross-platform builds and distribution
- `phase-5-cutover.md` — user migration and coexistence with the original C# app

## Multi-agent coordination rules

1. **One stage per agent at a time.** Never two agents on the same stage.
2. **Package ownership**: each phase declares which packages it owns. An
   agent working phase N only modifies that phase's packages. Touching
   another phase's packages requires a separate, minimal change justified in
   its changelog fragment.
3. **Stage status**: each phase file has a status table
   (`pending / in progress (<agent>) / done`). When claiming a stage, the
   agent marks it `in progress` with its name **in the same commit** where it
   starts — that is the reservation mechanism.
4. **Stage closing** (non-negotiable):
   - `./scripts/check.sh` and `./scripts/smoke.sh` green,
   - audit in `auditory/` (quality, performance, architecture),
   - changelog fragment,
   - mark the stage `done` in the phase table,
   - if the phase is now complete: update the top-level `README.md`.
5. Phase dependencies are strict: do not start a phase whose blocking
   predecessor has not closed (exceptions documented in each file).
