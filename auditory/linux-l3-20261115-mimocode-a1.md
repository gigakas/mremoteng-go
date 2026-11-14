# Audit — Linux track, Stage L3

- **Date (UTC)**: 2026-11-15
- **Agent**: mimocode-a1
- **Audited stage**: L3 — Session UX, optional import, credential strategy
- **Commits covered**: `b58c1a1` (2026-11-10, the Linux preview commit that carried
  import/export, the credential vault, quick connect, folder management and folder
  defaults), `fc6cfb6` (2026-11-11, terminal zoom), `f279bb3`, `961c011`,
  `8aa9217` (2026-11-12, zoom and scrollback corrections), `50c4e2b`, `b419ad2`,
  `fca48aa` (2026-11-12, workspace state and the folder/import mutations moved
  into it), `2bdab19` (2026-11-14, tabs restored), `8a58d3d` (2026-11-14, split
  layout restored), `5053838` + `2f9d36c` (2026-11-14, settings dialog), `f489832`
  (2026-11-14, notifications), `1e08976` (2026-11-14, manual order)
- **Files covered**: `internal/transfer/`, `internal/vault/`, `internal/terminal/`,
  `internal/profiles/{store.go,profiles.go}`, and in `internal/ui/`:
  `transfer.go`, `vault.go`, `quickconnect.go`, `folders.go`, `drop.go`,
  `workspace.go`, `session.go`, `terminal.go`, `zoom.go`, `settings_dialog.go`,
  `preferences.go`, `notifications.go`, `order.go`

**Why this audit exists late.** L3's scope was not delivered as one increment. Each
piece landed under a backlog stage (L5, L6, L9, L10) with its own audit, while the
stage row in `blueprint/linux-preview.md` accumulated the pieces; no audit was ever
written against L3 itself. This file closes that gap by auditing the aggregate that
the stage row now claims. It records no code change of its own.

## 1. Code quality

**Import and export (the "optional" half of L3).** `internal/transfer/transfer.go`
(267 lines) owns the mapping and nothing else: it walks the shared
`connection` tree, so XML and CSV share one path, imported rows go through
`profiles.Validate` (`transfer.go:96`), container names are reduced to single path
segments, and imported passwords are counted as dropped rather than stored. The UI
half (`internal/ui/transfer.go`) keeps the testable part separate from the file
dialogs: `importPreview`, `importSummary`, `duplicateCount`, `applyImport` and
`unionFolders` (`transfer.go:62-148`) are pure and covered by
`TestApplyImport_MergeSkipsDuplicatesAndAddsFolders`,
`TestApplyImport_ReplaceDropsExistingConnections`,
`TestImportSummary_ReportsSkippedAndDropped`, plus the workspace-level
`TestWorkspace_ImportMerge_...` and `TestWorkspace_ImportReplace_...`. Nothing is
written before the user picks merge or replace, and the source file is only read:
`docs/linux-preview-usability.md:79-88` makes the unchanged `sha256sum` part of the
acceptance script. Counted tests: 7 in `internal/transfer`, 4 in
`internal/ui/transfer_test.go`.

**Folder defaults.** `profiles.Resolve` (`internal/profiles/profiles.go:388`) is a
pure, non-mutating resolver with a nearest-ancestor rule per field, and
`Profile.Validate` still rejects out-of-range values when the field is not
inherited, so inheritance cannot be used to bypass validation. The dialog lives in
`internal/ui/folders.go:83` (`requestFolderDefaults`) and `:113`
(`saveFolderDefaults`); inherited values are shown as placeholders in the Config
panel instead of a second read-only grid. `internal/profiles` now carries 29 tests.

**Credential vault.** `internal/vault/vault.go` (248 lines) is a versioned,
self-describing file (`version`, `algorithm`, `iterations`, salt, nonce, payload),
PBKDF2-HMAC-SHA256 at 210,000 iterations (`vault.go:27`) with AES-256-GCM
(`algorithm` at `vault.go:29`), atomic temp-file writes at `0600`, and
`ErrWrongPassword`/`ErrLocked` as the two failure modes. The UI (`internal/ui/vault.go`,
128 lines) never stores implicitly: `passwordFor` (`vault.go:92-99`) resolves typed
value, then the stored secret for that profile id, then empty, and storing is an
explicit button (`requestStorePassword`, `vault.go:103`). 8 tests in
`internal/vault`, 2 in `internal/ui/vault_test.go`.

**Quick connect.** `internal/ui/quickconnect.go` (89 lines) builds a profile in
memory and reuses `profiles.Validate` and `openEphemeral`, so an ad-hoc session
cannot skip the rules a saved one obeys and takes the same panel path.
`TestQuickConnect_OpensTemporaryTabWithoutSaving` is the property that matters:
zero store writes.

**Folder management.** Folders are explicit in the document (`Document.Folders`),
so an empty folder survives a restart. Rename rewrites the whole subtree
(`workspace.go:232`) and delete is recursive with a counted confirmation
(`workspace.go:256`); the row context menu is built at `internal/ui/menu.go:26`,
`:59`, and drag-and-drop lands in `internal/ui/drop.go:20`.

**Terminal zoom, scrollback and copy.** Zoom is a terminal-only theme override
(`internal/ui/zoom.go`, 32 lines) driven by three canvas shortcuts
(`shell.go:496-501`), so the rest of the window keeps its size. Scrollback is
bounded at 2000 lines (`internal/terminal/terminal.go:42`), browsed by wheel and
PageUp (`internal/ui/terminal.go:546`), and copy takes the selection when there is
one and the whole buffer otherwise (`session.go:853-870`). 16 tests in
`internal/terminal`, 20 in `internal/ui/terminal_test.go`.

**Session restore with its split layout.** The document gained five optional
fields (`OpenTabs`, `ActiveTab`, `TabLayouts`, plus the earlier `Collapsed` and
`Order`; `internal/profiles/store.go:18-37`, 8 fields in total) and
`workspace.setSessions` (`workspace.go:127`) writes them through the same atomic
save as everything else. Three guards keep the write honest and are each pinned by
a test: ephemeral tabs are never recorded, an unchanged tab list is not written
(`shell.go:413`, `sameSessions` + `sameLayouts`), and the restore suppresses the
save while it replays. `profiles.PaneLayout` stores only shape, because every pane
of a tab is the same connection; `maxPanesPerTab = 12`
(`internal/profiles/profiles.go:253`) and `PaneLayout.Validate` (`:219`) reject a
tree the window cannot rebuild, and the restore ignores a bad layout rather than
failing the tab.

**Settings dialog and notifications.** `internal/ui/preferences.go` is the typed
view of Fyne's preference store (7 methods) and `settings_dialog.go` keeps the
widgets apart from `applySettings` (`:35`) and `parseSettingsPort` (`:60`), so a
refused value leaves the stored settings untouched. The notification history
(`internal/ui/notifications.go`, 154 lines) is bounded at 100 entries
(`notificationLimit`, `:40`), fed by the existing `setProblem` funnel plus the new
`setWarning`, and shares the bottom slot with the activity log by design.

**Manual order.** `Document.Order` is a flat list of tree node identifiers and
`moveInOrder` (`internal/ui/order.go:8`) swaps a row with its previous or next
sibling of its own kind, writing the whole displayed list back. The filtered case
is the subtle one and is handled: the write-back walks the unfiltered tree
(`orderedKeysForWrite`, `order.go:81`) while the move targets the previous visible
sibling, pinned by `TestMoveInOrder_WhileFiltered_KeepsTheHiddenNodesInPlace`.

**Debt accepted from incremental delivery.** Three concrete items:

- The single-field dialog is written twice: `showTextDialog`
  (`internal/ui/folders.go:66`) and `showSecretDialog`
  (`internal/ui/transfer.go:215`) are the same function body, differing only in
  `entry.Password` and the entry construction. One helper taking a `*widget.Entry`
  would cover both; about 20 lines are duplicated because the folder piece and the
  import piece landed in different increments.
- `internal/ui/terminal.go:59` still documents `fontSize` as "Ctrl+scroll zoom",
  although wheel zoom was removed in `961c011` and `shell.go:493-495` now explains
  why (scroll events carry no modifiers and the X query misreported on
  Wayland/XWayland). The comment outlived the behaviour it describes.
- `-profiles` redirects the profile store only; the vault stays at
  `$XDG_CONFIG_HOME/mremoteng-go/secrets.json`
  (`docs/linux-preview-usability.md:26-28`). A test run therefore touches the real
  credentials file, which the script has to work around by moving it aside.

## 2. Performance

Every buffer L3 added is bounded by a named constant rather than by use: 2000
scrollback lines (`internal/terminal/terminal.go:42`), 100 notifications
(`internal/ui/notifications.go:40`), 12 panes per tab
(`internal/profiles/profiles.go:253`), one small nested layout object per divided
tab only.

I/O follows one rule: never write unless the value changed. `shell.go:413` skips
the save when neither the tab list nor the layouts differ, which is what keeps tab
clicks from reaching the disk; the earlier collapsed-folder work reached the same
rule from the other direction after a first attempt measured 3 saves per tree
action. Key derivation is the only expensive computation and runs once per unlock
(210,000 PBKDF2 iterations, `internal/vault/vault.go:126-128`), never per session.
Import is linear in file size with one duplicate-detection map; a reorder rebuilds
the tree index once.

The one cost that grows with data rather than with activity: `setSessions`
(`workspace.go:127`) rewrites the whole document, profiles included, on a tab
change. For a few hundred connections that is tens of kilobytes per click, which is
why the unchanged-write guard carries the design; a store large enough to make that
visible would argue for splitting the workspace fields out of the profile document.

No benchmarks were run for this audit, and I am not claiming measured numbers: the
argument above rests on bounded buffers and one write per user action, both readable
off the code. The piece audits made the same claim without measurement.

## 3. Architecture

L3 owns no package of its own; it is the stage in which `internal/ui` learned to be
an application rather than a profile editor. The two packages it introduced,
`internal/transfer` and `internal/vault`, are attributed by
`blueprint/linux-preview.md:52-54` to their backlog stages, and both keep the
boundary the plan asks for: neither imports Fyne, `internal/transfer` depends only
on `internal/connection`, `internal/profiles` and both serializers, and
`internal/vault` depends on nothing else in the tree.

**No stored passwords, by design.** `grep -rn Password internal/profiles/*.go`
(non-test) returns nothing: the profile document has no credential field, and the
vault is a separate, user-unlocked file keyed by profile id. That is what makes a
restored tab come back unconnected rather than reconnecting silently, and it is the
reason the split-layout restore stores shape only.

**`internal/credential` exists with no consumer.** The package is a Go port of the
C# ExternalConnectors: 5 provider subpackages (`vault`, `delinea`, `passwordstate`,
`onepassword`, `awsec2`), 15 Go files and 35 tests, and zero importers outside
itself (verified by grep over the whole tree). Its own doc comment
(`internal/credential/credential.go:14-31`) records why: the UI has no picker, and
the C# convention for which existing field holds a Delinea/Passwordstate/1Password
secret id is not recoverable from this repository, so every client takes the
identifier as an explicit parameter instead of guessing. The decision is deferred,
not forgotten, and nothing in the session path depends on it. The package's fate was
being decided while this audit was written: see the observation in section 4.

**No new dependency.** `golang.org/x/crypto` was already required by
`internal/security/kdf.go`, so the vault added no module; no cgo was introduced by
any L3 piece (RDP stays an external process).

**Fyne sets the ceiling on input.** Key events carry no modifier information
(`internal/ui/terminal.go:356`, `:522`), so Ctrl combinations reach a shell only
through the shortcuts registered in `internal/ui/shell.go:480-502`, and wheel events
carry no modifiers either, which is why zoom is keys-only and the wheel always
browses scrollback. This is the same constraint that leaves modifier-key input open
in L2 (`blueprint/linux-preview.md:116-119`); L3 designed its UX around it instead
of hiding it, which is the right call but worth remembering when a user asks for
Ctrl-Shift-C.

**Schema stability.** All L3 state is optional and additive (`Defaults`,
`Collapsed`, `OpenTabs`, `ActiveTab`, `TabLayouts`, `Order`), so the document stays
version 2 and a file written before any of it loads unchanged. Older fields are
never repurposed.

## 4. Evidence

- `./scripts/check.sh`: **not re-run for this audit.** The audit records delivered
  code and changes none of it, so a run would report the tree, not this file. The
  piece audits record green: L5 and L6 (2026-11-09), L9 (2026-11-09), L10 quick
  connect/workspace/copy (2026-11-09), and the five 2026-11-14 L10 audits
  (sessions-settings, layout, notifications, order — `check.sh` OK, `smoke.sh` OK
  for order).
- `./scripts/smoke.sh`: **not re-run**, same reason.
- Focused re-verification performed now:
  `go test ./internal/vault ./internal/transfer ./internal/credential/... -count=1`
  → all 8 packages `ok`. Test inventory counted with `go test -list`: 267 tests in
  `internal/ui` across 32 test files, 29 in `internal/profiles`, 16 in
  `internal/terminal`, 8 in `internal/vault`, 7 in `internal/transfer`, 35 across
  the 6 `internal/credential` packages. Per feature file: `workspace_test.go` 13,
  `folders_test.go` 13, `order_test.go` 8, `notifications_test.go` 7,
  `settings_dialog_test.go` 6, `sessions_test.go` 6, `transfer_test.go` 4,
  `zoom_test.go` 2, `vault_test.go` 2.
- New tests in this stage: **none** — this audit adds no code. The test evidence is
  the sum of the piece audits listed above.
- Traceability caveat worth recording: the L5/L6/L9/L10 audits are dated
  2026-11-09, but no commit in the repository carries that date. Their code landed
  in `b58c1a1` on 2026-11-10, so the 2026-11-09 dates are working dates, not commit
  dates. Anyone reconstructing the timeline from `git log` should not read them as
  the commit date.
- **Observed, not audited.** While this audit was being written, the working tree
  gained an uncommitted, staged deletion of `internal/credential/` (15 files, 1655
  lines) plus uncommitted edits to `internal/protocol/serial`, `internal/ui/{panes,
  serial,shell}.go` and `docs/linux-preview-usability.md`. Those belong to another
  increment and are not part of the commits covered here, so every count in this file
  describes `ad1eadb`. If that deletion lands, pending action 1 is settled in the
  "delete" direction and only the plan text needs to follow.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

Rationale: every item the stage row claims is present in the tree and covered by
tests at the level the piece audits recorded, so the scope is delivered; but L3 has
no acceptance test of its own, the credential strategy still stops at the vault, and
the human pass that would confirm the UX is unexecuted.

## 6. Pending actions

- **External credential providers — undecided (owner: whoever picks L9 back up).**
  `internal/credential` has 5 provider clients and 35 passing tests and no importer.
  Deciding means either wiring a picker plus a resolution step at connect time, or
  deleting the package. Step 3 of this plan was taken: the provider packages were
  deleted in the same wave this audit was written in, because nothing imported them and
  no real provider is reachable here to verify a wiring against. The textual follow-up
  was done too — the provider sentence in `blueprint/backlog.md` and the pointer from
  the L3 row are gone.
- **No L3-owned test suite (owner: next change in `internal/ui`).** The coverage is
  the sum of per-feature files; there is no single headless test that imports a
  corpus file, sets folder defaults, opens two tabs with a split, reloads the store
  and asserts tree, tabs and layout together. That seam is exactly where the pieces
  were joined late, so it is the one place a regression could hide.
- **Human usability pass (owner: human).** `docs/linux-preview-usability.md` still
  has no recorded run. It already covers the L3 surface — folders (scenarios 11-13),
  import/export round trip (14), zoom/scrollback/copy and the vault — so one
  execution closes this item together with the L1 and L4 acceptance that depend on
  it.
- **Merge the two one-field dialogs** (`internal/ui/folders.go:66` and
  `internal/ui/transfer.go:215`) when either file is next touched; keep it a
  refactor-only commit.
- **Fix the stale wheel-zoom comment** at `internal/ui/terminal.go:59`.
- **Teach `-profiles` to redirect the vault path** so a manual or scripted run cannot
  touch the real `secrets.json`.
- Limitation carried forward, not a defect: pane sizes are not restored (a split
  returns at 50 %), notifications are not clickable, and reordering is menu-only.
- No top-level `README.md` update is due: L3 does not complete a phase.
