# Audit — Linux track, Stage L10 (notifications panel)

- **Date (UTC)**: 2026-11-14
- **Agent**: mimocode
- **Audited stage**: L10 — session UX, the notifications panel from
  `blueprint/backlog.md`
- **Commits covered**: this commit

## 1. Code quality

The window could only ever show one message: `setProblem` painted the status bar and
the activity log, and a second failure replaced the first before anyone read it. The
notification history (`internal/ui/notifications.go`) keeps what the status bar has
already forgotten.

- `notifications` is a bounded history (100 entries) of `{At, Level, Text}`. Text
  goes through the existing `oneLine`, so a client dumping a multi-line log still
  occupies one row, and the panel height does not jump as messages arrive — the same
  reason the activity console applies it.
- Levels are `noticeError` and `noticeWarning`: `setProblem` records an error and the
  new `setWarning` records a warning. Everything the panels report already funnels
  through `sessionPanel.setProblem` into `screen.setProblem`, so failures from SSH,
  RDP, VNC, transfers and the store all land here without touching each call site.
- `setWarning` was added for the one case that is neither a success nor a hard
  failure: FreeRDP ending after a session had been open (see
  `auditory/linux-l7-options-20261114-mimocode.md`). It keeps the previous inline
  `WarningImportance` styling and now also records a notification.
- The panel mirrors the activity console: monospace, one line per entry, a Clear
  button, and a count in its title. The badge in the status bar shows how many are
  waiting, because an icon alone does not tell the user that a panel exists after the
  first failure.

**The two bottom panels share one slot.** Showing notifications hides the log and the
other way round. Two stacked panels would take the height the terminal needs, which is
the constraint the whole chrome was built around; `TestToggleNotifications_ReplacesTheLog`
pins the exclusion.

**Not persisted, on purpose.** A notification describes this run, like the activity
log. A problem that outlives the window belongs in the log the user can copy, and
writing failures to the profile store would grow a file that exists to hold
connections.

## 2. Performance

Appending to a bounded slice and re-rendering at most 100 monospace lines when the
panel is open. No file I/O, no goroutines, nothing on a hot path: the same cost the
activity log already paid.

## 3. Architecture

No new dependency, no new package: `internal/ui` only. The panel is built like the log
console and follows the same rule about the status bar (an icon-only control explains
itself on hover), and it is documented on the start page's **Buttons** card so the new
icon is discoverable — `welcomeHelp` gained a Notifications row and the test that
guards that card now also requires the word.

## 4. Evidence

- `./scripts/check.sh`: OK.
- New tests: `TestNotifications_Add_KeepsTheNewestWithinTheLimit`,
  `TestNotifications_Render_ListsOneLinePerMessage`,
  `TestSetProblem_RecordsANotification`,
  `TestSetWarning_RecordsAWarningWithoutDangerStyling`,
  `TestNotificationPanel_Clear_EmptiesTheHistoryAndTheBadge`,
  `TestToggleNotifications_ReplacesTheLog`,
  `TestNotificationsHint_CountsTheWaitingProblems`.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Human: open a connection to a dead port, then to another one, and check the badge
  shows two and the panel lists both with their times; clear it and the badge goes.
- L10 still owes manual ordering of the tree.
- Notifications are not clickable; a message cannot jump to the connection it came
  from, which would be the natural next step.
