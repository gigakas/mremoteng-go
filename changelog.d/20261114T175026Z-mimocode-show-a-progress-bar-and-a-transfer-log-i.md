---
timestamp: 2026-11-14T17:50:26Z
agent: mimocode
files:
  - internal/ui/remote_files.go
  - internal/ui/remote_files_test.go
---

Show a progress bar and a transfer log in the SFTP window

A large download only had a status sentence, which says nothing about how far along it is or what happened a minute ago. The window now keeps two things under the panes. A progress bar follows the bytes: startProgress resets it and shows it, setProgress moves it against the size of the selection, and finishProgress parks it at full so the finished transfer stays visible. The status line under it reports bytes and percentage. A transfer log lists each start, finish and failure with a timestamp in a monospaced, muted font, keeps the last hundred lines in a scroll of fixed height, and mirrors every entry into the application activity log through the existing logger, so a transfer is recorded even after the window is closed. The helpers treat a nil widget as a no-op rather than dereferencing it, which a probe caught before it could become a crash. Tests cover the progress text with and without a known total, the log cap with its mirroring into the activity log, and the nil-widget path. Validated with ./scripts/check.sh, without Fyne errors in its output.
