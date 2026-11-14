# Audit — Linux track, Stage L10 (session restore and settings dialog)

- **Date (UTC)**: 2026-11-14
- **Agent**: mimocode
- **Audited stage**: L10 — session UX and workspace state, the two items the backlog
  listed as pending: session persistence and the settings dialog
- **Commits covered**: `2bdab19` (tabs restored), `5053838` (settings dialog),
  `2f9d36c` (unused settings package removed)

## 1. Code quality

**Restoring a workspace without storing a secret.** The profile document gains
`open_tabs` and `active_tab` (`internal/profiles/store.go`) and `workspace` gains
`setSessions`, which writes them through the same atomic save as everything else.
`screen.rememberSessions` runs on open, close and select and records the tabs in
visual order, so the restore puts them back where they were. Restored tabs are
**not connected**: the application stores no password, so each tab comes back with
Connect enabled and the user decides. That is the honest shape of "session
persistence and reconnect" in a build that never keeps credentials — the alternative
would be a password cache, which the project rules forbid.

Three guards keep the write honest:

- **Ephemeral tabs are never recorded.** A quick-connect profile is not in the store,
  so restoring its tab would show a connection that cannot come back
  (`rememberSessions` skips ids `profileByID` does not know).
- **An unchanged list is not written.** The feature runs on every tab click, so the
  comparison against `ws.sessions` avoids a file write per click. Without it, quick
  connect stopped being "nothing saved" — which is what
  `TestQuickConnect_OpensTemporaryTabWithoutSaving` asserts.
- **The restore does not write what it reads.** `restoring` suppresses the save while
  the tabs are replayed, then the pruned list is written once.

A fourth property is worth naming because it is easy to break later: the tab list
travels inside the same document as the profiles, so **every** other save carries it.
`TestScreen_OtherSaves_KeepTheOpenTabs` exists because a rename rewriting the
document to an empty tab list would look like the feature never worked.

**Storage validation was deliberately skipped for the tabs.** They are ids, and an
unknown id is dropped on restore with a test; rejecting a whole document because a
tab points at a deleted connection would take the profiles down with it.

**Settings.** `internal/ui/preferences.go` is the typed view of the Fyne preference
store (theme variant, default protocol, default port, reopen tabs), with the
defaults in one place and a nil store answering with them so a headless test needs
no app. `settings_dialog.go` keeps the widgets apart from `settingsValues`, so
`applySettings` and `parseSettingsPort` are testable without a window; a refused
value leaves the stored settings untouched.

**Fyne has no light/dark setting.** The variant comes from the desktop, so the
choice is honoured inside the theme: `SoberThemeVariant` pins the variant and
`soberTheme.Color` ignores the one the framework passes. This is a special case and
it is documented as one — the palette is the only place the variant is read, so
pinning it there is complete for our colours, but it would not change a widget that
queried the framework's variant directly.

**Two tests changed for the right reason.** Opening a workspace tab now writes the
project document, so `TestSave_...` expects at least one write rather than exactly
one, and the declined-delete test compares before and after instead of expecting
zero. Both were counting writes to prove "nothing was saved"; the new write is the
tab list, not the edit under test.

**Dead package removed.** `internal/settings` had no importer: window geometry
already lived in Fyne's preference store, and the theme choice and new-connection
defaults now live there too. Keeping a second, unwired mechanism for the same data
is how two sources of truth start, so package and tests are gone and the decision it
documented (settings in a plain config file, no Windows registry) still holds,
because Fyne's store is one.

## 2. Performance

Tab changes write a small JSON file (a profile store of a few hundred connections is
tens of kilobytes), and only when the list actually changes. The restore opens tabs
without dialling anything, so startup cost is one file read plus one widget tree per
tab. No polling added.

## 3. Architecture

No new dependency. `internal/profiles` gained two optional fields — a document
written before them loads with no tabs, tested — and `internal/ui` gained the
preference layer and the dialog. The protocol packages are untouched. Deleting
`internal/settings` removes a package that no stage had wired.

Limitation, recorded rather than hidden: a restored tab is one pane. The split
layout inside a tab is not persisted, so a tab that had two panes comes back as the
connection alone. Persisting the tree would mean a nested document shape and a
restore path that rebuilds pane nodes; it is a bigger change than the value it adds
today, and the tab is one click from being split again.

## 4. Evidence

- `./scripts/check.sh`: OK.
- New tests: `TestStore_OpenTabs_RoundTripAndSurviveOtherSaves`,
  `TestScreen_OpenAndCloseTabs_ArePersisted`,
  `TestScreen_RestoreSessions_ReopensTheTabsUnconnected`,
  `TestScreen_RestoreSessions_SkipsConnectionsThatAreGone`,
  `TestScreen_RestoreSessions_PreferenceOff_OpensNoTabs`,
  `TestScreen_SelectedTab_IsRememberedAcrossRestarts`,
  `TestScreen_OtherSaves_KeepTheOpenTabs`,
  `TestAppPreferences_FreshInstall_HasTheDocumentedDefaults`,
  `TestApplySettings_StoresAndValidatesTheValues`,
  `TestParseSettingsPort_EmptyMeansTheProtocolDefault`,
  `TestNewConnectionProfile_UsesTheStoredDefaults`,
  `TestSoberThemeVariant_PinsTheColours`,
  `TestSoberThemeForPreferences_ReadsTheStoredChoice`.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Human: open three connections, close one, restart the app — the two tabs must be
  back, unconnected, with the last selected one in front. Then File > Settings,
  pick dark, Apply (the window repaints), restart (it is still dark).
- L10 still owes the split layout in the restore, the notifications panel and manual
  ordering from `blueprint/backlog.md`; the manual usability script in
  `docs/linux-preview-usability.md` still needs the run that closes L1/L4.
