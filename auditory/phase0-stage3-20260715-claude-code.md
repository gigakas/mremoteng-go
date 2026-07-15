# Audit — Phase 0, Stage 3

- **Date (UTC)**: 2026-07-15
- **Agent**: claude-code
- **Audited stage**: 0.3 Wayland assessment
- **Commits covered**: fc7c71e..HEAD (assessment produced after stage 0.1 closure)

## 1. Code quality

- Documentation-only stage: no code was written or modified. Deliverable is
  `docs/spike-wayland.md`; findings are sourced (FreeRDP discussion #11595,
  Debian wlfreerdp3 manpage, Phoronix on FreeRDP 3.2) and dated, since the
  Wayland ecosystem moves — the doc states its as-of date (2026-07).
- Empirical basis: the full 0.1 checklist executed on GNOME Shell 50.1
  native Wayland (evidence chain in `docs/spike-x11.md`), not a synthetic
  re-run — honest reuse, declared explicitly in the doc.

## 2. Performance

- Not applicable — no code. The only performance-adjacent finding
  (fractional-scaling blur on XWayland) is recorded as cosmetic in
  `docs/spike-wayland.md`.

## 3. Architecture

- No new dependencies, no package changes.
- Key architectural constraint produced for later phases: **the Linux app
  must run as an X11/XWayland client** (never build Fyne with the `wayland`
  tag while embedding is a feature). Consumers: stage 2.5 (RDP), 2.7
  (AnyDesk), 4.1 (build matrix must not add a wayland-tagged variant).
- Deviation from the stage text: KDE Plasma could not be tested (no Plasma
  environment on the machine). Declared in the doc with a concrete
  follow-up owner/stage instead of silently narrowing scope.

## 4. Evidence

- `./scripts/check.sh`: OK (2026-07-15, unchanged code).
- `./scripts/smoke.sh`: OK (2026-07-15, unchanged code).
- New tests in this stage: none — documentation-only stage, nothing to
  test.

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

1. Run the 0.1 checklist on KDE Plasma Wayland when an environment exists
   (owner: human provides VM/machine, claude-code executes; latest: fold
   into 5.3/5.4 preview feedback).
2. Stage 0.4 (go/no-go) must weigh the KDE gap explicitly when presenting
   options to the human (owner: 0.4 claimant).

Phase 0 is **not** complete (0.2 and 0.4 pending); top-level README
untouched.
