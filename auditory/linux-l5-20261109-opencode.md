# Audit — Linux track, stage L5

- **Date (UTC)**: 2026-11-09
- **Agent**: opencode
- **Audited stage**: L5 — mRemoteNG XML/CSV import and export
- **Scope**: `internal/transfer`, `internal/ui/transfer.go`, File menu wiring

## 1. Code quality

- `internal/transfer/transfer.go` owns the mapping only: it walks the shared
  `connection` tree, so XML and CSV share one code path instead of duplicating
  parsing logic per format.
- Imported rows are validated with `profiles.Validate`; rows that cannot be used
  are counted as unsupported instead of being stored broken
  (`transfer.go:96`). Container names are sanitised into single path segments
  (`sanitizeSegment`), so a `/` or `..` in a source folder cannot escape the tree.
- Imported passwords are **never** carried over: they are counted and reported
  (`Preview.Dropped`) because the native store has no credential storage yet.
- IDs are regenerated for every imported connection; the source ID is not trusted
  and cannot collide with existing profiles.
- UI logic is split from file dialogs: `importPreview`, `duplicateCount`,
  `applyImport`, `unionFolders` and `importSummary` are pure functions of their
  inputs and are unit-tested; the pickers are thin wrappers.
- Merge/replace is explicit in the preview dialog; nothing is written before the
  user chooses, and the imported file is only read.

## 2. Performance

- Parsing is linear in file size; the preview holds the imported profiles in
  memory (mRemoteNG files are small, low thousands of rows at most).
- Merge builds one key map for duplicate detection, O(n + m).

## 3. Architecture

- `internal/transfer` depends on `internal/connection`, `internal/profiles` and
  both serializers; no new external dependency was added.
- The native document remains the single source of truth; import produces native
  profiles rather than making the UI adapter-specific.
- Export rebuilds a `connection` tree from native profiles/folders, which keeps
  XML/CSV output identical to the serializers already covered by the compatibility
  corpus tests.
- Known limitation, recorded in the backlog: inheritance flags from the source
  file are flattened to effective values, which is why L6 (folder defaults) comes
  next.

## 4. Evidence

- `./scripts/check.sh`: green (gofmt check, vet, all tests).
- New tests: `internal/transfer/transfer_test.go` (folder mapping, password
  drop/count, unsupported protocol, protected file needs password, XML and CSV
  round trips, segment sanitising) and UI tests for merge (duplicate skip, folder
  union), replace, summaries and folder union.
- Manual check pending: pick a real mRemoteNG file through File > Import and
  export it back. The fixtures used by the tests are generated with the Go
  serializer, so source compatibility still relies on the corpus suite.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Human check with a real mRemoteNG export (XML and CSV) through the UI.
- `internal/serialize/csv` writes credentials columns empty; confirm on import
  round trip with a real C# CSV that no column is lost.
- Next planned stage: L6 (folder defaults and inheritance) per
  `blueprint/backlog.md`.
