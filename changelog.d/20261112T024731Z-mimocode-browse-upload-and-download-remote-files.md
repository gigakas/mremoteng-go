---
timestamp: 2026-11-12T02:47:31Z
agent: mimocode
files:
  - auditory/linux-sftp-20261112-mimocode.md
  - blueprint/linux-preview.md
  - go.mod
  - go.sum
  - internal/protocol/ssh/files.go
  - internal/protocol/ssh/files_test.go
  - internal/ui/remote_files.go
  - internal/ui/remote_files_test.go
  - internal/ui/shell.go
---

Browse, upload and download remote files over the SSH session

Adds file transfer to an SSH session: Tools > Remote files (SFTP) opens a single-pane browser over one SFTP session, starting in the remote home directory, with a path bar, an Up button and a list that puts directories first. Uploading takes a local path and copies into the current remote directory; downloading takes a local path for the selected file; both report progress in the dialog and the status bar. The protocol layer in internal/protocol/ssh/files.go reuses the existing Target and Options types and the unexported connect helper, so host-key approval, agent and key authentication and the one-off password behave exactly as the shell session does, and the password never reaches argv. New dependency: github.com/pkg/sftp v1.13.12 (BSD-2, pure Go), justified in auditory/linux-sftp-20261112-mimocode.md; implementing SCP by hand was the alternative and it has no in-process server to test against. That upgrade moved golang.org/x/crypto from 0.33.0 to 0.54.0, which is the SSH stack of this application, so it was verified against a real server: TestProbeAndShell_LocalContainer passes against OpenSSH 10.3 (lscr.io/linuxserver/openssh-server) with handshake, fingerprint, password auth and an interactive shell. Tests: eight protocol cases driven by an in-process SFTP server on a loopback socket (listing order and kinds, missing directory, upload and download content plus progress, missing local and remote files, ParentDir at the root, JoinRemote with absolute names) and two formatting cases in the UI. Known gaps, deliberate: no remote delete or rename, no recursion, no resume, no local pane. Validated with ./scripts/check.sh.
