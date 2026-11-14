---
timestamp: 2026-11-14T18:39:19Z
agent: mimocode
files:
  - internal/ui/diagnostics.go
  - internal/ui/dialogs.go
  - internal/ui/folders.go
  - internal/ui/quickconnect.go
  - internal/ui/remote_files.go
  - internal/ui/remote_files_test.go
  - internal/ui/session.go
  - internal/ui/shell.go
  - internal/ui/transfer.go
  - internal/ui/vault.go
---

Return the keyboard to the previous widget after any modal dialog closes

The transfer window needed to be clicked again before the terminal accepted input, and the cause was general: a Fyne modal popup takes the canvas focus when it opens and never gives it back. showModal now returns a modal wrapper that remembers which object had the keyboard before the dialog opened and focuses it again in Hide, so every window built on it returns the keyboard where it was: new connection, quick connect, diagnostics, vault, import, export, folder and connection dialogs, the About window and the transfer browser. The About window was still building its popup inline and now goes through showModal as well, which leaves widget.NewModalPopUp in exactly one place. The transfer browser lost its own focus bookkeeping because the wrapper covers it. TestModal_Hide_ReturnsTheKeyboardToThePreviousObject focuses a terminal, shows a dialog, moves the focus away as the popup does and hides it, asserting the terminal owns the keyboard again and the remembered focus was cleared. Validated with ./scripts/check.sh, without Fyne errors in its output.
