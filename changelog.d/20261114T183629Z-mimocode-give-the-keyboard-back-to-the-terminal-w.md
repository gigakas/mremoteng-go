---
timestamp: 2026-11-14T18:36:29Z
agent: mimocode
files:
  - internal/ui/remote_files.go
  - internal/ui/remote_files_test.go
---

Give the keyboard back to the terminal when the transfer window closes

Reported as the console needing to be unfocused and refocused before it accepted input again. The window is a modal popup, and a modal popup takes the canvas focus when it opens and does not hand it back when it hides, so after closing the browser the keystrokes went nowhere until the terminal was clicked again. The window now remembers the terminal that owned the keyboard before it opened and focuses it again on close, clearing the reference so it happens once. TestRemoteBrowser_Close_ReturnsTheKeyboardToTheTerminal reproduces the sequence: focus the terminal, take it away as the popup does, close, and assert the terminal owns the canvas focus again. Validated with ./scripts/check.sh, without Fyne errors in its output.
