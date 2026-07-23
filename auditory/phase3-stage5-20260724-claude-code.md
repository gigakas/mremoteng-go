# Audit — Phase 3, Stage 3.5

- **Date (UTC)**: 2026-07-24
- **Agent**: claude-code
- **Audited stage**: Options dialog + settings persistence
- **Commits covered**: uncommitted at audit time; see the stage 3.5
  implementation and closing-audit changelog fragments created alongside
  this file

## 1. Code quality

- `internal/settings/settings.go` (new package): `Settings` is a small
  JSON-serializable struct (window size, last connections file, theme
  name). `Load` treats a missing file as `Default()` with no error (first
  run has nothing to load — not an error condition), but a *malformed*
  existing file still returns an error rather than silently falling back,
  so a corrupt settings file is visible instead of quietly discarded.
  `Save` creates the parent directory (`MkdirAll`) since
  `os.UserConfigDir()`'s `mremoteng-go` subdirectory won't exist on first
  run. Doc comment explicitly states the scope gap this package leaves
  open: no machine-wide/enterprise config equivalent to the original
  app's registry policies — user-level JSON only.
- `internal/ui/connectionsfile.go` (new): `LoadConnectionsFile`/
  `SaveConnectionsFile` are thin wrappers over Phase 1's
  `internal/serialize/xml` (`Deserialize`/`Serialize`), adding only
  `os.ReadFile`/`os.WriteFile` and error wrapping. This is the piece that
  actually satisfies Phase 2/3's shared "demo config file" exit
  criterion — no earlier stage wired the deserializer to anything the UI
  could reach.
- `internal/ui/optionsdialog.go` (new): `ShowOptionsDialog` builds a
  `dialog.ShowForm` with a Theme `Select` (values fixed to
  `system`/`light`/`dark` — stage 3.6 is what will actually consume
  `Settings.Theme` to apply a palette; this dialog only collects and
  persists the value ahead of that) and a Last Connections File `Entry`.
  On confirm it copies `*s`, applies the edited fields, and calls `onSave`
  with the copy — the caller (main.go) owns actually persisting it, same
  division of responsibility `ConnectionTree.OnSelect` already uses
  (`internal/ui` widgets don't do their own disk I/O).
- `internal/ui/tree.go`: added `ConnectionTree.Root()` and `SetRoot()`,
  needed once `cmd/mremoteng` can load a *different* file into an
  already-running tree widget. `SetRoot` always calls `Reload()`
  unconditionally, unlike an in-place add under an existing container
  (see `Reload`'s doc comment from stage 3.2) — a full root swap always
  invalidates the whole `nodes` index, there's no cheaper path.
- `internal/ui/shell.go`: replaced the stage-3.1 placeholder
  `"New Connections File"` item with `OnOpenConnectionsFile`/
  `OnSaveConnectionsFile`/`OnOptions` callback fields, called from three
  new File-menu items if set (nil-safe no-ops otherwise, covered by
  `TestNewShell_FileMenuItemsAreNoOpsWhenCallbacksUnset`). Kept
  `"New Connection"` as an explicit still-unwired placeholder — creating a
  brand-new tree node has no UI anywhere yet, and inventing one wasn't
  this stage's scope (settings/file persistence, not tree editing).
- `cmd/mremoteng/main.go`: gained real logic (settings load/save wiring,
  file-open/save dialogs, a password-prompt helper) rather than staying
  a pure call-through. This mirrors the existing precedent in the same
  file (`tree.OnSelect`'s protocol-creation logic, stage 3.1) — the
  binary is still a thin *assembly* layer (no new types, no algorithms of
  its own), just one with more wiring now that there's more to wire.
  `promptConnectionsFilePassword` is the one net-new non-trivial function
  there; kept unexported and single-purpose rather than moved into
  `internal/ui`, since it's pure UI glue with no state or testable
  behavior beyond "doesn't panic" (already covered indirectly by
  `optionsdialog_test.go`'s equivalent smoke test for the sibling
  dialog).
- No duplication, no function over ~50 lines, no discarded errors —
  `saveSettings`'s error is logged (`log.Printf`), not discarded; a failed
  settings write shouldn't crash the app mid-session the way a failed
  settings *load* at startup does (`log.Fatalf`), since by that point the
  user has unsaved session state (open tabs) that fatal-ing would lose.

## 2. Performance

Not applicable: settings load/save is a few hundred bytes of JSON, on
startup/shutdown/dialog-confirm only — never a hot path.

## 3. Architecture

- `internal/settings` is new, self-contained, and does not depend on
  `internal/ui` or `internal/connection` — importable by both the CLI
  binary and (unchanged) by `internal/ui` for `ShowOptionsDialog`'s
  parameter type. No import cycle risk.
- `internal/ui` still doesn't import `internal/protocol` for its own
  sake — `connectionsfile.go` only reaches into
  `internal/serialize/xml` and `internal/connection` (both already
  in-scope dependencies via `tree.go`/`properties.go`).
- No impact on closed Phase 1/2 packages' public contracts:
  `internal/serialize/xml.Deserialize`/`Serialize` are called exactly as
  documented, not modified.

## 4. Evidence — same visual-verification limitation as 3.1-3.4

- `./scripts/check.sh`: **green**.
- `./scripts/smoke.sh`: **green**.
- **No visual verification was possible or attempted**, the same
  phase-wide limitation recorded in every prior stage's audit and in
  `blueprint/phase-3-ui.md`'s note. The Options form's field layout and
  the two new File-menu entries are exactly the kind of thing a real
  screen would immediately validate or correct; neither was possible
  here.
- New/changed tests:
  - `internal/settings/settings_test.go` (new, 4 tests): missing-file
    default, save-then-load round trip, malformed-file error,
    `DefaultPath` sanity.
  - `internal/ui/connectionsfile_test.go` (new, 3 tests): a **real
    end-to-end round trip through Phase 1's actual AES-256-GCM/PBKDF2
    encryption** (root → folder → connection with hostname/username/
    password, saved, reloaded, every field including the password
    verified to survive), a wrong-password error case, and a
    missing-file error case.
  - `internal/ui/optionsdialog_test.go` (new, 1 test): a deliberately
    narrow smoke test (builds and shows the form against a real
    `*settings.Settings` without panicking) — documented inline as to why
    a full click-through-the-form interaction test wasn't attempted
    (would mean reaching into Fyne's own `dialog` package internals for
    uncertain benefit, since `ShowOptionsDialog` itself is a thin wrapper
    around Fyne's own tested `dialog.ShowForm`).
  - `internal/ui/tree_test.go`: +1 test,
    `TestConnectionTree_SetRoot_SwapsRootAndReindexes` — confirms both
    that the new root's nodes become reachable and that the *old* root's
    nodes stop being reachable (not just additively indexed).
  - `internal/ui/shell_test.go`: +2 tests — the three new File-menu items
    call their respective callbacks, and are no-ops (don't panic) when
    the callbacks are left unset.

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- **No visual confirmation of the Options dialog or File-menu wiring** —
  repeating the phase-wide note.
- **Window position isn't persisted, only size** — `Settings` has no
  X/Y fields; Fyne's `fyne.Window` doesn't expose a portable
  position-query API the same way it exposes `Canvas().Size()`, so this
  was left out rather than built on uncertain platform-specific ground.
- **No "recent files" list** — only the single `LastConnectionsFile` is
  tracked, not opened automatically at startup either (a first version of
  this stage considered auto-loading it, but decided against silently
  prompting for a password on every launch before the user has done
  anything — left for whoever can iterate on this with real usage rather
  than guessed at blind).
- **The password prompt has no "remember for this session" option** —
  every open/save round-trips through the form once, even
  saving-immediately-after-loading the same file. Matches how the
  original app's own per-operation prompts work when a master password
  isn't cached, but this Go port hasn't built the master-password-cache
  feature at all yet, so there's nothing to opt into.
- **No enum-aware or validated Theme selection beyond the fixed
  `system`/`light`/`dark` list** — stage 3.6 is what will determine
  whether that's actually the right value set once a real palette exists
  to map them to.
- Commit the working tree — not done without explicit request.
