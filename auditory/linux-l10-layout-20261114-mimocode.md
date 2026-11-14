# Audit — Linux track, Stage L10 (split layout in the session restore)

- **Date (UTC)**: 2026-11-14
- **Agent**: mimocode
- **Audited stage**: L10 — session UX, the pending action left by
  `auditory/linux-l10-sessions-settings-20261114-mimocode.md`: "L10 still owes the
  split layout in the restore"
- **Commits covered**: this commit. The previous L10 audit stays as written.

## 1. Code quality

`profiles.PaneLayout` (`profiles.go`) stores one tab's split tree: `Split` marks a
node with two children, `SideBySide` its orientation, `A`/`B` the subtrees, and a leaf
is the zero value. Only shape is stored because every pane of a tab is the same
connection — storing the profile again per pane would duplicate data and allow the
copies to disagree.

`paneNode` gained `sideBySide`, set when a split is created and copied when a pane
closes and its sibling takes the place. Without it the orientation would have to be
read back out of the Fyne `container.Split` widget, which is the kind of coupling this
package avoids elsewhere.

`splitPane` was split into `addPane` (builds the pane, no connection) plus
`splitPane` (builds and connects). The restore needs the first half: no credential is
stored, so a restored pane must wait for its own Connect. `growPanes` rebuilds a tree
by splitting the existing pane on the left/top and giving the fresh pane the other
side, which is the order the split command produces — so a restored tree is
indistinguishable from one built by hand.

**Refusals instead of surprises.** `PaneLayout.Validate` rejects a split with a
missing side, a leaf that carries children, and more than `maxPanesPerTab` (12)
panes. The store runs it on save, and the restore runs it again on the value it reads,
then ignores an unusable layout rather than failing the tab: a corrupted layout costs
the split, never the session list. `TestStore_Save_RejectsALayoutTheWindowCannotRebuild`
and `TestScreen_RestoreSessions_IgnoresALayoutTheProtocolCannotSplit` cover both ends;
the second also proves a store that claims an RDP tab was divided leaves it single,
because `splittable` refuses RDP and VNC.

**Saving.** `rememberSessions` now also writes the layouts, and splitting or closing a
pane calls it. The "skip an unchanged write" rule grew a `sameLayouts` comparison, so
a split that changes nothing still does not touch the file, and a change of shape does.

## 2. Performance

One small nested object per divided tab. A leaf is `{}` and a tab with a single pane
stores no entry at all, so the common case costs nothing. The restore splits panes
without dialling, so it adds widget construction only.

## 3. Architecture

No new dependency; `internal/profiles` gained a recursive type and one validation
rule, `internal/ui` the save/restore halves. `Document.TabLayouts` is additive and
optional: a file written before it loads with no layouts and therefore no splits
(`TestStore_TabLayouts_RoundTripAndSurviveOtherSaves`), so the schema version stays 2.

The limitation from the previous audit is now closed. What the restore still does not
do, by design: connect a pane (no stored credentials) or remember pane sizes (a split
comes back at the default 50 %).

## 4. Evidence

- `./scripts/check.sh`: OK.
- New tests: `TestStore_TabLayouts_RoundTripAndSurviveOtherSaves`,
  `TestStore_Save_RejectsALayoutTheWindowCannotRebuild`,
  `TestScreen_SplitLayout_IsPersistedAndRestored`,
  `TestScreen_RestoringPanes_KeepsTheTabPointingAtTheRoot`,
  `TestScreen_RestoreSessions_IgnoresALayoutTheProtocolCannotSplit`.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Human: split a terminal to the right, split one of the panes downwards, restart —
  the tab must come back with three panes in the same shape, each waiting on Connect.
- L10 still owes the notifications panel and manual ordering of the tree.
- Pane sizes are not restored; the split returns at 50 %.
