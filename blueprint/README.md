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

## Agent roster and task distribution

Two agent families work on this repo:

- **`claude-code`** — interactive agent with the user nearby. Default owner
  for stages needing desktop/GUI validation, architectural decisions or
  human sign-off.
- **`opencode`** — headless agent for well-specified, test-driven stages.
  Several OpenCode agents may run in parallel using suffixed names
  (`opencode-2`, `opencode-3`, …); each name claims its own stage and is
  used as `-agent` in its changelog fragments.

Each phase's status table has an **Agent** column with the suggested owner:

| Value | Meaning |
|---|---|
| `claude-code` | Needs interaction, visual checks or decisions — keep in Claude Code. |
| `opencode` | Well-specified and delegable to an OpenCode agent. |
| `any` | First free agent takes it. |
| `human + <agent>` | The agent prepares evidence; the human decides/provides input. |

Rules:

1. **Delegating a stage to OpenCode always requires asking the user
   first.** Whatever the Agent column says, no stage is claimed by an
   `opencode*` agent without explicit user confirmation — the column is a
   suggestion, the user is the dispatcher.
2. The suggestion is not a lock: the only reservation mechanism remains
   marking the stage `in progress (<agent>)` in the status table.
3. When reassigning a stage, update the Agent column in the same commit
   that claims it.

## Multi-agent coordination rules

1. **One stage per agent at a time.** Never two agents on the same stage.
2. **Package ownership**: each phase declares which packages it owns. An
   agent working phase N only modifies that phase's packages. Touching
   another phase's packages requires a separate, minimal change justified in
   its changelog fragment.
3. **Intra-phase parallelism**: two agents may work the same phase only
   when their stages touch disjoint packages/files — each phase file lists
   this under "Parallelism & collision notes". If a stage needs to touch a
   file owned by another stage (e.g. a shared factory), it does so as a
   separate, minimal commit justified in its changelog fragment.
4. **Stage status**: each phase file has a status table
   (`pending / in progress (<agent>) / done`). When claiming a stage, the
   agent marks it `in progress` with its name **in the same commit** where it
   starts — that is the reservation mechanism.
5. **Stage closing** (non-negotiable):
   - `./scripts/check.sh` and `./scripts/smoke.sh` green,
   - audit in `auditory/` (quality, performance, architecture),
   - changelog fragment,
   - mark the stage `done` in the phase table,
   - if the phase is now complete: update the top-level `README.md`.
6. Phase dependencies are strict: do not start a phase whose blocking
   predecessor has not closed (exceptions documented in each file).
