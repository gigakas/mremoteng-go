---
timestamp: 2026-11-14T18:17:21Z
agent: mimocode
files:
  - internal/ui/remote_files.go
  - internal/ui/remote_files_test.go
  - internal/ui/terminal.go
  - internal/ui/terminal_fix_test.go
---

Stop the transfer window from blocking the console and long selections freezing

Two reports with the same shape: work that belongs off the UI thread running on it. Closing the transfer window called Close on the SSH session synchronously, which writes a disconnect and waits for the socket, so the console stayed unresponsive for as long as that took on a remote host; the session is now closed in its own goroutine after the pointer is dropped, the running transfer is cancelled first, the copy closures capture the session they started with so a close mid-transfer cannot nil it, and closing twice is a no-op. Selecting text in the terminal had the grid rebuilt on every pointer move, and the character cell measured with the font on every event; a long selection therefore froze the window. The cell is now measured once per text size and cached, invalidated by comparing the font size, and the drag repaint is throttled to forty milliseconds while the selection itself still updates per event, with DragEnd always painting so the final highlight is never lost. Tests: the cached cell follows a zoom, a drag burst repaints only once until the interval passes, releasing always paints, and closing a window that never opened a session is idempotent. Validated with ./scripts/check.sh, without Fyne errors in its output.
