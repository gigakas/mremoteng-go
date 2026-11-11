---
timestamp: 2026-11-11T22:24:19Z
agent: mimocode
files:
  - internal/ui/actions.go
  - internal/ui/diagnostics.go
  - internal/ui/dialogs.go
  - internal/ui/folders.go
  - internal/ui/quickconnect.go
  - internal/ui/session.go
  - internal/ui/shell.go
  - internal/ui/transfer.go
  - internal/ui/ui_test.go
  - internal/ui/vault.go
---

Extract the modal dialog and writable guard duplication in internal/ui

Nine request handlers each built and showed a modal popup with the same three-line sequence, and nine others repeated the same read-only guard with only the verb changed. internal/ui/dialogs.go now owns showModal, which wraps the content in a fixed-size grid, pads it, attaches it to the window canvas and shows it; callers keep the returned popup so their buttons can hide it. actions.go now owns requireWritable, which keeps the exact user-facing sentence, the verb and the status-bar problem severity while returning a boolean the handler branches on. No behaviour change: every dialog keeps its size and every guard keeps its wording. Unlike the previous extractions this one barely moves the line count (18 call sites collapse to two definitions); the gain is that the canvas, the padding and the read-only explanation now have a single owner instead of eighteen. Added TestRequireWritable_ReadOnlyStore_ReportsProblemAndBlocks and TestRequireWritable_WritableStore_AllowsAndKeepsStatus, which the old inline form made impossible to cover. Validated with ./scripts/check.sh.
