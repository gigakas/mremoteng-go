# Audit — Phase 0, Stage 1

- **Date (UTC)**: 2026-07-15
- **Agent**: claude-code
- **Audited stage**: 0.1 X11 reparenting on Linux
- **Commits covered**: ad31676..HEAD (spike scaffold through HiDPI offset fix)

## 1. Code quality

- Spike code is **throwaway by charter** (`internal/spike/README.md` states
  the exemption from the unit-test rule); validation was manual against the
  checklist, performed by the human on 2026-07-15. Acceptable for Phase 0
  only — no spike pattern may be copied into `internal/protocol/` without
  tests.
- `internal/spike/x11reparent/main.go:1` documents tag guarding, usage and
  the GPLv2 constraint; every error path updates the UI status and kills the
  child process (`fail` closure, main.go:118) — no silent failures, no
  zombies (single reaper goroutine, main.go:106).
- `internal/spike/x11reparent/x11.go:60` (`findChildWindow`) takes the last
  child from QueryTree without filtering; fine for the spike (GLFW creates
  no children) but a production embedder must match the child by PID.
- Debt, intentional: `-mode reparent` does not survive xfreerdp's window
  re-creation (documented in `docs/spike-x11.md`); needed later for AnyDesk
  (stage 2.7), not for closing 0.1.

## 2. Performance

- Not a performance stage. Window discovery polls at 200 ms (x11.go:75,
  x11.go:110) with hard deadlines — bounded work, irrelevant CPU. The event
  loop blocks on `WaitForEvent` (x11.go:160), zero idle cost. No
  measurements taken; nothing here ships.

## 3. Architecture

- Package boundaries respected: all code under `internal/spike/`
  (phase-owned), zero imports from other phases' packages.
- New dependencies, justification required by AGENTS.md:
  - `fyne.io/fyne/v2` — the UI toolkit the whole migration plan is built
    on (docs/MIGRATION_PLAN.md); first use, inevitable.
  - `github.com/BurntSushi/xgb` — pure-Go X11 wire protocol, the only
    no-cgo way to do reparenting/geometry; named explicitly by the
    blueprint (blueprint/phase-0-spike.md, stage 0.1).
  - Both guarded behind the `spike` build tag, so default builds and
    `check.sh` need no C toolchain today. **Phase 3 will make Fyne a hard
    dependency of `cmd/mremoteng`** — CI (4.1/4.2) must install the C build
    deps then.
- License constraint honored: xfreerdp is driven strictly as an external
  process (exec + X11), never linked.
- Deviation from the stage text: the blueprint said "reparent via xgb"; the
  spike found naive reparenting unreliable (window re-creation) and
  validated `/parent-window` as primary with xgb kept for geometry/events
  and as the future generic fallback. This is a finding, not scope creep —
  recorded in docs/spike-x11.md and it informs stage 2.5/2.7 design.

## 4. Evidence

- `./scripts/check.sh`: OK (gofmt + go vet + go test, 2026-07-15).
- `./scripts/smoke.sh`: OK (binaries build, mremoteng starts, changelog
  reproducible).
- New tests in this stage: none — throwaway spike code exempted per
  `internal/spike/README.md`; validation was the manual checklist, passed
  by the human (session embedded, resize+scaling, focus in/out, exit
  cleanup).

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

1. Stage 0.3 must fold in the finding that this validation already ran
   under XWayland/GNOME-Wayland (owner: claude-code, stage 0.3).
2. Phase 2.5 design must default to `/smart-sizing` with per-host opt-in to
   `/dynamic-resolution` (owner: whoever claims 2.5; source:
   docs/spike-x11.md).
3. Phase 4.2 CI must not use `linuxserver/rdesktop` as RDP test host
   (unreliable s6 init) — provision xrdp explicitly (owner: 4.2 claimant).
4. Generic re-embed-on-recreation logic for AnyDesk stays open until stage
   2.7 (owner: 2.7 claimant).

Phase 0 is **not** complete (0.2–0.4 pending); top-level README untouched.
