---
timestamp: 2026-11-14T17:09:09Z
agent: mimocode
files:
  - blueprint/linux-preview.md
  - internal/ui/remote_files.go
---

Choose the local file with the platform dialog for SFTP upload and download

Uploading asked the user to type a local path into a text field, and downloading did the same, which made the only real step of the feature the one that felt broken. Upload now opens the platform file chooser and copies the chosen file into the current remote directory under its base name; download opens the platform save chooser pre-filled with the remote file name and writes the remote copy there. Both callbacks are thin: they resolve the path from the chooser URI and hand it to uploadFrom or downloadTo, which keep the empty-path and no-selection guards. The save chooser opens the destination file, so the download truncates and refills it. Validated with ./scripts/check.sh.
