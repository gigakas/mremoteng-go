# Audit — Linux preview, two-pane remote file transfer

- **Date (UTC)**: 2026-11-14
- **Agent**: mimocode
- **Audited stage**: The SFTP transfer window grown from one pane to two: local
  and remote side by side, with upload and download between them.
- **Commits covered**: the commit that adds `internal/ui/filepane.go` and rewrites
  `internal/ui/remote_files.go`.
- **Supersedes**: `auditory/linux-sftp-20261112-mimocode.md` for the window
  layout. That audit stays untouched; its protocol findings still hold.

## 1. Code quality

- The two sides are one widget. `fileSource` (Title, Initial, List, Parent) has a
  `localSource` over `os` and a `remoteSource` over `RemoteFiles`, so the pane,
  its rows, its sorting and its status line exist once. Adding a third source —
  a second connection, or a local mount — means implementing four methods.
- The row widget is a typed `fileRow` carrying its icon and labels. This is the
  fix for the crash reported against the first version: `container.NewBorder`
  keeps its nil slots, so reaching into `Objects` by index panicked on the first
  empty slot.
- Selection and navigation are deliberately the same gesture: clicking a
  directory enters it, clicking a file selects it. There is no recursive copy, so
  a directory is never a transfer item, and `selection()` returns false for one.
  That asymmetry is documented in the method comment rather than hidden.
- Both panes re-list after a transfer, so the copied file appears without a manual
  refresh; the cost is one `ReadDir` per side per transfer.
- Known gap: the panes do not share a scroll position or a filter, and there is
  no keyboard shortcut for the transfer buttons.

## 2. Performance

- One `ReadDir` per navigation and one `Stat` per local entry. A directory with
  thousands of entries builds thousands of `fileEntry` values, which is what the
  previous version did too; the list virtualises the widgets, not the slice.
- Transfers still stream through `io.Copy`; the progress callback per buffer is
  unchanged from the previous audit.

## 3. Architecture

- No new dependency and no protocol change: the transfer layer added in the
  previous commit is reused as is, including `JoinRemote` and `ParentDir` for the
  remote side. Local paths use `filepath` and remote paths use `path`, which keeps
  the two spellings from leaking into each other.
- `remoteBrowser` is now a coordinator: it owns the two panes, the buttons and the
  progress line, and holds no directory state of its own. Directory state lives in
  `filePane`, which is what makes the panes testable without a dialog.

## 4. Evidence

- `./scripts/check.sh`: `check OK: gofmt + go vet + go test`
- `./scripts/smoke.sh`: `smoke OK`
- Local SSH container (OpenSSH 10.3),
  `SSH_LOCAL=127.0.0.1:2222 … go test ./internal/protocol/ssh/ -run LocalContainer`:
  PASS, including the upload/download round trip added by the previous commit.
- New tests: local listing order and kinds, `Parent` at the root, row rendering
  for a directory and a file with an out-of-range index and a foreign object, and
  the selection rules that keep a directory out of a transfer.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions
- [ ] Stage NOT closed

## 6. Pending actions

- Manual acceptance: open a session, transfer a file in each direction and confirm
  both panes refresh.
- Available on request: recursive directory copies, a transfer queue with cancel,
  resume, remote delete/rename, and a keyboard shortcut for each direction.
