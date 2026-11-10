# Audit — Linux track, stage L11 (part 1: diagnostics and exit cleanup)

- **Date (UTC)**: 2026-11-09
- **Agent**: opencode
- **Audited stage**: L11 partial — diagnostics and crash-safe session shutdown
- **Scope**: `internal/ui/diagnostics.go`, `sessionPanel.shutdown`, window close path

## 1. Code quality

- `diagnosticsText` is a pure function: version, store path, profile/folder
  counts, vault state and open sessions. It contains no secret, and the vault is
  reported only as locked/unlocked/not configured.
- The dialog copies the same text to the clipboard, so support reports do not
  depend on screenshots.
- `shutdown()` closes shell, RDP, VNC and stream sessions and destroys the
  embedded X11 area, and the window close path calls it before hiding the window.
  RDP uses `rdpDetached` so the watcher goroutine does not touch the UI while the
  app is closing.
- `activeSessionCount` counts panels holding a live session without asking the
  sessions themselves, keeping the diagnostics read-only.

## 2. Performance

- Nothing on a hot path; the counts are O(panels).

## 3. Architecture

- Cleanup lives with the panel that owns the session, so new protocols get it for
  free as long as they set their field.
- Diagnostics deliberately builds its own text instead of dumping the store:
  secrets and endpoints of unrelated connections stay out of support output.

## 4. Evidence

- `./scripts/check.sh`: green.
- New tests: diagnostics content and absence of secrets, shutdown closing stream
  and VNC sessions, destroying the embedded area and leaving zero active sessions.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Verify on a real exit that no `xfreerdp` process or X11 child remains (manual).
- Remaining L11: schema migration test with a pre-L6 file on disk, and a recovery
  action for a corrupt store (currently writes are disabled, which is safe but
  offers no repair path).
