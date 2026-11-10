# Audit — Linux track, stage L10 (part 2: workspace state)

- **Date (UTC)**: 2026-11-09
- **Agent**: opencode
- **Audited stage**: L10 partial — persisted collapsed folders
- **Scope**: `internal/profiles` document, tree callbacks in `internal/ui`

## 1. Code quality

- Collapsed folders are stored as paths next to folders/defaults, validated with
  the same `FolderSegments` rules, and serialised only when non-empty.
- `branchChanged` records user toggles; the state is written with the next
  document save and on window close (`closeWindow` → `persistWorkspace`), so tree
  interaction never causes a write storm (a first attempt persisted on every
  branch event and was removed after tests showed 3 saves per action).
- `applyCollapsed` runs after `OpenAllBranches` on load, so the default expanded
  tree is then trimmed to the saved state.
- All save sites now go through `document()`/`documentWithDefaults()`, which keeps
  defaults and collapsed state from being dropped by individual features.

## 2. Performance

- One extra string slice per save; no per-interaction I/O.

## 3. Architecture

- Workspace state lives in the same versioned document (optional field), so no
  schema bump was needed and older files load unchanged.
- Tree paths, not Fyne uids, are persisted, which keeps the file independent of
  the widget implementation.

## 4. Evidence

- `./scripts/check.sh`: green.
- New tests: restore of a collapsed folder on load, toggle tracking, and
  persistence through a save/load round trip (fake store now carries collapsed
  paths).
- The earlier "3 saves" expectation failure is what drove the write-on-close
  design.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Manual check: collapse a folder, close and reopen the app.
- Remaining L10: terminal scrollback/selection-copy, session persistence and
  reconnect, notifications panel, settings dialog, manual ordering.
