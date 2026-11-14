---
timestamp: 2026-11-14T17:31:52Z
agent: mimocode
files:
  - internal/protocol/ssh/files.go
  - internal/protocol/ssh/files_test.go
  - internal/ui/filepane.go
  - internal/ui/filepane_test.go
  - internal/ui/remote_files.go
---

Create and delete folders on both sides of the transfer window

The two-pane window could only move data that already existed. Each pane now has New folder and Delete. New folder asks for a name and creates it in that pane's current directory; Delete confirms first and then removes the selected entry, a folder going with everything inside it. The work is split where the rest of the feature already splits: fileSource gained Join, Mkdir and Remove, so localSource uses os.MkdirAll and os.RemoveAll while remoteSource uses two new RemoteFiles methods, MkdirAll and a recursive RemoveAll that lists each directory, deletes its children and then the directory itself. The pane only calls those callbacks; the dialogs and their wording live in the dialog, which is what keeps the pane testable. Two safeguards: the current directory is never a deletion target, and a folder deletion states in the confirmation that it takes the contents. Tests cover the nested local create and remove, the remote recursive remove, the remote nested create, and the refusal plus the confirmed local deletion. Validated with ./scripts/check.sh.
