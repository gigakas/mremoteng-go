# Audit — Linux preview, remote file transfer (SFTP)

- **Date (UTC)**: 2026-11-12
- **Agent**: mimocode
- **Audited stage**: Remote file transfer over an SSH session: browse a remote
  directory, upload by path, download by path.
- **Commits covered**: the commit that adds `internal/protocol/ssh/files.go`,
  `internal/ui/remote_files.go` and the `github.com/pkg/sftp` dependency.

## 1. Code quality

- The protocol layer is split so the testable part needs no SSH:
  `listDir`, `uploadFile` and `downloadFile` take a `*sftp.Client`, and the
  exported `RemoteFiles` methods only add the connection. That is why eight tests
  run in process against a real SFTP server on a loopback socket instead of a
  fake.
- `RemoteFiles` owns one SSH and one SFTP connection for the lifetime of the
  browser dialog. Re-authenticating on every directory change would cost a full
  handshake per click, which is the difference between usable and not on a remote
  host.
- Errors are wrapped with the operation and the path
  (`transfer: write <path>: <cause>`), because the failure the user sees is
  usually "which file", not "which package".
- `listDir` drops the `.` and `..` entries the server may report and resolves one
  level of symlink so a link to a directory is navigable. Sorting puts
  directories first, then files, each by name.
- Known gap, deliberate: there is no resume, no recursion and no remote delete or
  rename. The stage scope is one file in each direction by path.
- The UI keeps formatting in two pure helpers (`formatSize`,
  `formatModified`) that are unit tested; the dialog itself is not tested because
  Fyne dialogs cannot be driven headlessly in this repository.

## 2. Performance

- Transfers stream through `io.Copy` with a 32 KiB default buffer, so memory use
  does not follow file size.
- Progress is reported once per write buffer. On a slow link that is a handful of
  `fyne.Do` calls per second; on a fast localhost transfer it can be thousands,
  which only updates a label. If it ever shows up as jank, throttle in
  `progressWriter` — not measured here.
- Listing is one `ReadDir` per navigation, plus one `Stat` per symlink (rare).

## 3. Architecture

- Boundary respected: `internal/protocol/ssh` gains a file, the UI gains a file,
  and nothing else moves. The existing `Target`/`Options` types and the unexported
  `connect` helper are reused, so host-key verification, agent/key/password
  authentication and timeouts behave exactly as the shell session does. The
  password still travels in memory only and never in `argv`.
- **New dependency: `github.com/pkg/sftp` v1.13.12** (BSD-2-Clause, pure Go).
  Justification: the standard library has no SFTP or SCP client, and the
  alternatives are worse — implementing the SCP wire protocol is ~300 lines of
  legacy quirks with no in-process server to test against, and invoking the
  system `scp`/`sftp` would put the password in `argv`, which `AGENTS.md`
  forbids and which this repository already avoids for FreeRDP.
- **Side effect that needed verification: `golang.org/x/crypto` moved from
  v0.33.0 to v0.54.0** (with `x/net`, `x/sys` and `x/text`). That is the package
  that implements the SSH handshake and known_hosts for this application, so the
  upgrade was verified against a real server rather than only against fakes:
  `TestProbeAndShell_LocalContainer` against `lscr.io/linuxserver/openssh-server`
  (OpenSSH 10.3) passes — handshake, fingerprint, password authentication and an
  interactive shell. The alternative, pinning `x/crypto` back, would have kept the
  older SSH stack to avoid a change that is almost certainly a security
  improvement.
- The `RemoteFiles` type is a second SFTP consumer waiting to happen: a
  dual-pane browser or a recursive copy would extend it, not replace it.

## 4. Evidence

- `./scripts/check.sh`: `check OK: gofmt + go vet + go test`
- `./scripts/smoke.sh`: `smoke OK`
- Local SSH container: `SSH_LOCAL=127.0.0.1:2222 … go test ./internal/protocol/ssh/ -run LocalContainer` → PASS against OpenSSH 10.3
- New tests:
  - `internal/protocol/ssh/files_test.go`: listing order and kinds, missing
    directory, upload content and progress, download content and progress, missing
    local file, missing remote file, `ParentDir` at the root, `JoinRemote` with
    absolute names and `..`.
  - `internal/ui/remote_files_test.go`: size formatting across units and the
    unknown date case.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions
- [ ] Stage NOT closed

## 6. Pending actions

- Manual acceptance against a real server: open an SSH session, run Tools >
  Remote files (SFTP), navigate, upload a file and download it back. Automated
  tests cover the protocol but not the dialog on screen.
- Not implemented on purpose, available if asked: remote delete/rename, recursive
  copy, resume, and the dual-pane (local + remote) form.
