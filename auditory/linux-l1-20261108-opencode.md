# Audit — Linux track, stage L1

- **Date (UTC)**: 2026-11-08
- **Agent**: opencode
- **Audited stage**: L1 — mRemoteNG-style profile manager (`blueprint/linux-preview.md`)
- **Scope**: `internal/profiles`, `internal/ui`, `cmd/mremoteng`, build/test wiring

## 1. Code quality

- `internal/profiles/profiles.go:1` separates the model and validation rules
  from storage (`store.go`). Validation rejects `host:port`, scheme prefixes,
  `/`, `@`, control characters and `-`-prefixed values; the last two matter
  because L2 will pass these fields to external clients.
- `internal/profiles/store.go:70` writes through a temporary file created with
  `0600` in the destination directory, `Sync`s it and then renames, so a failed
  write cannot truncate the previous store. Failed loads leave writes disabled
  (`internal/ui/actions.go:17`), so corrupt data is never overwritten.
- `internal/ui/tree.go:29` builds the tree from folder paths and sorts folders
  before connections. Folders are intentionally implicit: an empty folder would
  need a schema addition that this stage does not require.
- `internal/ui/actions.go` holds all state transitions; `withDiscardGuard`
  centralises the unsaved-changes prompt instead of duplicating it in five
  call sites. Functions stay below the 50-line threshold except `save`, which is
  a linear sequence of guarded steps.
- Test coverage: store round trip, permissions, corrupt/unsupported/duplicate
  data, validation table, tree filtering and nested folder keys, selection and
  tab opening, delete with confirmation, write lockout, discard guard, protocol
  port defaults, revert and panel toggling. All UI tests use the software test
  driver and run without a display.

## 2. Performance

- Tree rebuilds allocate one index per filter change (`buildTreeIndex`), which
  is O(n log n) for n profiles; irrelevant at the target scale (hundreds to a
  few thousand connections).
- Saves rewrite the whole JSON file. That keeps atomicity simple; if stores grow
  past a few thousand profiles the next step is a per-connection file and an
  index, not a mutable append log.
- No benchmarks were added: the stage has no measured hot path.

## 3. Architecture

- Native storage is independent of mRemoteNG's XML. `internal/profiles` owns the
  native schema (versioned JSON, `version: 1`); `internal/connection` +
  `internal/serialize/*` remain untouched regression coverage for optional
  import, as recorded in `docs/MIGRATION_PLAN.md`.
- New dependency: `fyne.io/fyne/v2 v2.8.2` (plus its transitive modules). It is
  the toolkit named in the plan; it requires cgo and the X11/GL headers, which
  also means `scripts/check.sh` now compiles cgo code. `go.mod` also gained the
  indirect modules Fyne needs; no other direct dependency was added.
- `internal/ui` is split into shell (layout/menu/tabs/status), tree index,
  properties panel and actions. The `Store` interface keeps the window testable
  without touching the filesystem.
- Stage boundary: sessions are absent by construction — session tabs state that
  they are placeholders, so the UI cannot imply working connectivity.

## 4. Evidence

- `./scripts/check.sh`: passed (gofmt check, `go vet`, all tests).
- `./scripts/smoke.sh`: builds all binaries and `mremoteng -version` passes; the
  changelog reproducibility comparison is reported separately because the
  regenerated `CHANGELOG.md` is unstaged in this working tree.
- Desktop startup: the binary ran for 6 s on this machine's XWayland session
  without errors (`-profiles` scratch store).
- Visual review: the software-driver render was inspected and fixed twice —
  an empty tree (Fyne only walks children when the invisible root reports
  `IsBranch`, fixed in `internal/ui/shell.go`) and nested folder children keyed
  by raw path instead of UID (`internal/ui/tree.go`).
- New tests: `internal/profiles/profiles_test.go` (model, validation, folder
  paths, store round trip, permissions, corrupt/unsupported files) and
  `internal/ui/ui_test.go`.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Human usability run of `docs/linux-preview-usability.md`, including the menu
  bar on a real window manager (menus are driver-drawn and do not appear in the
  software-driver captures).
- Decide whether the Config panel should be resizable; today it is a fixed
  bottom panel with a two-column form.
- L2 (SSH/RDP sessions) starts only after L1 acceptance; it introduces the
  protocol package and the first real connectivity claims.
