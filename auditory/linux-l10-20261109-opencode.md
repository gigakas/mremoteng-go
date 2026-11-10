# Audit — Linux track, stage L10 (part 1: Quick connect)

- **Date (UTC)**: 2026-11-09
- **Agent**: opencode
- **Audited stage**: L10 partial — Quick connect and temporary sessions
- **Scope**: `internal/ui/quickconnect.go`, sidebar header and File menu

## 1. Code quality

- Quick connect builds a profile in memory and reuses `profiles.Validate`, so an
  ad-hoc session cannot bypass the same rules as a saved one.
- Temporary sessions never touch the store: the tests assert zero saves and an
  empty profile list, which is the property that matters for this feature.
- The tab is opened through `openEphemeral`, which shares the panel construction
  with saved connections; no second code path for session behaviour.

## 2. Performance

- No effect: one panel and one tab per quick session, closed like any other.

## 3. Architecture

- Quick sessions are deliberately not added to the tree (there is no profile
  record to select later); the status bar says they are not saved and points at
  "New connection" for persistence.
- Reusing `sessionPanel` means VNC/RDP/SSH options, embedding and cleanup all
  behave identically for temporary sessions.

## 4. Evidence

- `./scripts/check.sh`: green.
- New tests: temporary tab creation with the expected profile, no store writes,
  and invalid host/port rejection with an explanatory status.
- The button lives in the sidebar header (add icon) and in File > Quick connect.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Remaining L10: terminal scrollback and selection/copy, session persistence and
  reconnect, notifications panel, settings dialog, persisted expanded state and
  manual ordering.
- Quick sessions disappear on restart; persisting them is part of the workspace
  state work above.
