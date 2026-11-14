---
timestamp: 2026-11-14T19:44:45Z
agent: mimocode
files:
  - internal/protocol/ssh/files.go
  - internal/protocol/ssh/files_test.go
  - internal/ui/filepane.go
  - internal/ui/filepane_test.go
  - internal/ui/remote_files.go
  - blueprint/linux-preview.md
---

Rename entries in both transfer panes

The panes could create and delete but not rename, so fixing a typo meant downloading, deleting and uploading again. Each pane has a Rename button now, editing one selected entry in a single-field dialog; a multiple selection is reported instead of half-renamed, and a name that is already in use is refused on both sides rather than replacing the file. Rename goes through two paths: os.Rename locally and RemoteFiles.Rename over the open SFTP session, whose renameRemote core stats the target first because SFTP servers disagree on whether a rename replaces an existing entry. TestRename_* exercise the move, the refused target and the missing source over the in-process SFTP server; TestLocalSource_Rename_* and TestFilePane_ActionRow_OffersCreateRenameAndDelete cover the local pane and keep the button on the row. Validated with ./scripts/check.sh.
