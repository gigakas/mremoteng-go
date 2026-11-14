---
timestamp: 2026-11-14T17:57:43Z
agent: mimocode
files:
  - internal/protocol/ssh/files.go
  - internal/protocol/ssh/files_test.go
  - internal/ui/remote_files.go
  - internal/ui/remote_files_test.go
---

Stop a running transfer from the SFTP window

A large copy could only be waited out. Upload and Download now take a context, and the copy reads through a cancelableReader that checks it before each block, so a cancel lands within one buffer instead of at the end of the file; directory copies check between entries too. The window grew a Stop button beside the progress bar: it is disabled while idle, enables for the duration of a transfer, and clears the cancel function as it fires so a second press cannot affect anything. A cancelled copy is reported as cancelled rather than failed, in the status line, in the window log with a warning level and in the activity log, and both panes are re-listed because a partial file may already exist. Cancellation tests upload and download four megabytes and cancel from the progress callback, asserting context.Canceled in both directions, plus the two window cases: stopping with nothing running leaves the status untouched, and stopping a running copy calls the cancel function, clears it and says it is stopping. Validated with ./scripts/check.sh, without Fyne errors in its output.
