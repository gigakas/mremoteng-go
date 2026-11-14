---
timestamp: 2026-11-14T16:43:57Z
agent: mimocode
files:
  - internal/protocol/ssh/files_test.go
  - internal/ui/remote_files.go
---

Ask for host key approval before an SFTP transfer and never fail silently

Reported as no dialog appearing when opening Tools > Remote files. Two causes: the SFTP connection verified the host key like any other, but an unknown key only wrote a problem to the status bar, so the menu looked dead; and the menu required the terminal to hold the keyboard, which a click on the tree removes. The transfer now resolves its session from the focused pane or, failing that, the selected tab, and an UnknownHostKeyError shows the same approval dialog the terminal uses: the fingerprint, and a retry with the key accepted for that transfer. Added TestRemoteFiles_LocalContainer, an end-to-end case against a real server gated on SSH_LOCAL: it authenticates, resolves the working directory, lists it, uploads a file and downloads it back, comparing the content. Against lscr.io/linuxserver/openssh-server with OpenSSH 10.3 both this and the existing probe-and-shell case pass. Validated with ./scripts/check.sh.
