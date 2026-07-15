# Phase 0 — Embedding validation spike (2–3 weeks)

**Goal**: prove that the "external process + window reparenting" approach
works reliably before committing to the rest of the project.
**Blocking**: no RDP protocol work starts until this phase closes.

**Owned packages**: `internal/spike/` (throwaway code, deleted when the
phase closes), `docs/spike-*.md` (findings).

## Stages

| # | Stage | Status | Agent |
|---|---|---|---|
| 0.1 | X11 reparenting on Linux | done | claude-code |
| 0.2 | Win32 reparenting on Windows | in progress (claude-code) | claude-code |
| 0.3 | Wayland assessment | done | claude-code |
| 0.4 | Documented go/no-go decision | pending | human + claude-code |

### Parallelism & collision notes

No parallelism: the spike is sequential (0.2/0.3 reuse the 0.1 prototype,
0.4 needs all findings) and every stage requires a desktop session and
visual validation — one agent at a time on the whole phase.

### 0.1 X11 reparenting on Linux
- Fyne window with an empty container panel.
- Launch `xfreerdp` against a test host (a container running xrdp is fine).
- Reparent its window into the panel via `github.com/BurntSushi/xgb`
  (pure-Go X11, no cgo — never touch libfreerdp: GPLv2 restriction).
- Validate: resizing the panel resizes the session, keyboard focus enters
  and leaves correctly, process exit is detected and the panel cleaned up.

### 0.2 Win32 reparenting on Windows
- Same prototype with `SetParent`/`SetWindowLong` from `golang.org/x/sys/windows`.
- Test against `xfreerdp.exe` (official FreeRDP build) and `mstsc` as fallback.
- Validate the same three points as 0.1.

### 0.3 Wayland assessment
- Run the 0.1 prototype on GNOME and KDE under Wayland (via XWayland).
- Document in `docs/spike-wayland.md`: what works, what degrades, and the
  state of `xdg-foreign` support in xfreerdp as of that date.

### 0.4 Go/no-go decision
- Write `docs/spike-result.md` with the recommendation and its evidence.
- If reparenting is unreliable on modern GNOME/KDE, the continue/rethink
  decision belongs to the human — present options, do not decide.

## Exit criteria
Stages 0.1–0.4 done, each with its audit; `docs/spike-result.md` approved by
the human; top-level README updated with the outcome.
