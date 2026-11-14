---
timestamp: 2026-11-14T17:17:45Z
agent: mimocode
files:
  - auditory/linux-sftp-dualpane-20261114-mimocode.md
  - internal/ui/filepane.go
  - internal/ui/filepane_test.go
  - internal/ui/remote_files.go
  - internal/ui/remote_files_test.go
---

Split the SFTP window into local and remote panes

The transfer window had one pane and two path prompts. It now opens as a two-pane browser: the local filesystem on the left and the remote host on the right, each with its own path bar, Up button and list that puts directories first. Clicking a directory enters it and clicking a file selects it; Upload to remote copies the file selected on the left into the remote directory shown on the right and Download to local does the reverse, keeping the file name, reporting progress in the dialog and the status bar, and re-listing both panes afterwards. Both sides are one widget driven by a new fileSource interface with a localSource over os and a remoteSource over the existing RemoteFiles, so the rows, the sorting and the status line exist once and a third source would only need four methods. localSource and the pane's row rendering and selection rules are covered by tests, including the out-of-range and foreign-object cases that crashed the first version. No new dependency and no protocol change. Recorded in auditory/linux-sftp-dualpane-20261114-mimocode.md, which supersedes the layout part of the earlier audit while leaving it untouched. Validated with ./scripts/check.sh.
