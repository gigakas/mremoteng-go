# Audit — Phase 0, Stage 2

- **Date (UTC)**: 2026-07-16
- **Agent**: claude-code
- **Audited stage**: 0.2 Win32 reparenting on Windows
- **Commits covered**: 5bd5735..HEAD (per-OS refactor through mstsc smart sizing)

## 1. Code quality

- The stage forced a healthy refactor: `internal/spike/reparent/` now has a
  shared UI (`main.go`) over a `sessionEmbedder` interface with per-OS
  implementations (`x11.go`, `win32.go`) — the same shape
  `internal/protocol/` will need in Phase 2, discovered early.
- `win32.go` uses raw `NewLazySystemDLL("user32.dll")` procs because
  `golang.org/x/sys/windows` does not wrap user32 window management. Error
  handling follows the Win32 idiom (ret==0 + GetLastError), with the one
  subtle case (`SetWindowLongPtrW` legitimately returning 0) commented at
  win32.go:88.
- `clientArgs` (main.go) is pure except for the mstsc temp-file branch,
  which logs and degrades gracefully if the write fails.
- Spike remains test-exempt (throwaway, `internal/spike/README.md`);
  validation was the manual checklist on a real Windows VM, passed by the
  human on 2026-07-16.

## 2. Performance

- Win32 variant polls at 200 ms for resize-follow and death detection
  (win32.go:151) instead of event hooks — bounded, negligible CPU,
  explicitly noted as spike-grade in `docs/spike-win32.md`. Not shippable
  as-is; fine for the validation purpose.

## 3. Architecture

- Package boundaries respected: everything under `internal/spike/`.
- New dependency: `golang.org/x/sys` (windows syscalls) — already an
  indirect dependency, now direct; standard-library-adjacent, zero risk.
- Cross-compilation from Linux with mingw (`CC=x86_64-w64-mingw32-gcc`)
  verified — de-risks the Phase 4.1 build matrix early.
- Deviation from the stage text: validation used `mstsc` (the blueprint's
  named fallback) instead of `xfreerdp.exe`, because FreeRDP stopped
  publishing Windows binaries in GitHub releases. The generic SetParent
  mechanism this stage exists to validate is client-agnostic, so the
  substitution does not weaken the conclusion; the wfreerdp gap is recorded
  in `docs/spike-win32.md` and feeds stages 0.4 and 4.4.

## 4. Evidence

- `./scripts/check.sh`: OK (2026-07-16).
- `./scripts/smoke.sh`: OK (2026-07-16).
- Linux build (`-tags spike`) and Windows cross-build: OK.
- New tests in this stage: none — spike exemption (see §1); manual
  checklist passed (embed, resize+scaling via .rdp smart sizing, focus
  in/out, exit cleanup).

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

1. Validate a FreeRDP client (`wfreerdp.exe` nightly) on Windows when
   convenient — confirms `/parent-window` there and unblocks the 4.4
   packaging decision (owner: 2.5/4.4 claimant; source: docs/spike-win32.md).
2. Phase 2.5 must design a per-client argument builder (mstsc `.rdp` file
   vs FreeRDP slash options) — sketch exists in the spike's `clientArgs`
   (owner: 2.5 claimant).

Phase 0 is **not** complete (0.4 pending); top-level README untouched.
