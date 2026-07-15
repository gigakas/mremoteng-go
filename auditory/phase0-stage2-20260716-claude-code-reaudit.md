# Audit — Phase 0, Stage 2 (re-audit)

- **Date (UTC)**: 2026-07-16
- **Agent**: claude-code
- **Audited stage**: 0.2 Win32 reparenting on Windows
- **Commits covered**: 409193d..HEAD (post-closure fixes: DPI, dialog
  filtering, adopt retry, resize strategies)
- **Why a re-audit**: the first closure (`phase0-stage2-20260716-claude-code.md`)
  was premature — the "validated" mstsc embedding regressed immediately
  after (silent SetParent refusal). Per `auditory/README.md`, audits are
  immutable; this file supersedes the verdict with the real validation.

## 1. Code quality

- The diagnostic-driven rounds left the spike better than the first pass:
  `adoptChild` extracted from `embedSession` (win32.go), find-and-adopt is
  a deadline retry loop, every Win32 call outcome is logged (`SetParent
  ret/errno`, DPI contexts, window classes) — remote debugging over pasted
  logs worked and the pattern is worth repeating in Phase 2.5.
- Win32 error idioms now correct and commented in place: cleared
  last-error before `SetParent`, NULL-return-with-clear-error is success,
  post-call verification via `GetAncestor`.
- Process still test-exempt (spike); validation manual on a real VM.

## 2. Performance

- Unchanged from first audit: 200 ms polling, spike-grade, documented.

## 3. Architecture

- No new dependencies since the first audit. Package boundaries unchanged.
- Material finding for later phases, now recorded in `docs/spike-win32.md`:
  Windows production client is **sdl-freerdp via generic reparenting**
  (no `/parent-window`); mstsc demoted to unreliable fallback (uiAccess/UIPI
  suspect); `smart-sizing` universal fallback, `dynamic-resolution`
  server-dependent opt-in (fails against the xrdp container on both
  platforms).

## 4. Evidence

- `./scripts/check.sh` and `./scripts/smoke.sh`: OK (2026-07-16).
- Human-validated on the VM: sdl-freerdp session embedded and interactive
  (screenshot evidence in session), Notepad control embeds and re-embeds,
  dynamic-resolution disconnect reproduced and documented.
- New tests: none (spike exemption).

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

Carried over and updated:

1. Phase 2.5: per-client argument builder + the full DPI/adopt-retry
   recipe from `docs/spike-win32.md` §"field manual".
2. Phase 4.4: FreeRDP Windows binaries are nightly-only — pin/mirror or
   build in CI.
3. Retest `/dynamic-resolution` against a real Windows RDP host (owner:
   2.5 claimant).

Phase 0 remains one stage short (0.4 go/no-go).
