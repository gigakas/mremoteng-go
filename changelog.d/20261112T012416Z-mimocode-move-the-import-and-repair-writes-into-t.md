---
timestamp: 2026-11-12T01:24:16Z
agent: mimocode
files:
  - internal/ui/repair.go
  - internal/ui/transfer.go
  - internal/ui/workspace.go
---

Move the import and repair writes into the workspace

Phase B3 and the end of the write extraction. applyImport built the merged or replaced document inline, wrote it, assigned the slices and then refreshed the tree; repairStore wrote an empty document directly on the store and re-enabled writes by hand. The workspace now owns both: importInto keeps the exact merge rule the preview counts (lowercased name plus folder as the key), appends the incoming folders through unionFolders, commits only if the write succeeds and returns the added and skipped counts the status line reports, and replace mode still starts from an empty document so every incoming profile is added. reset writes the empty document and re-enables writes, which is the only exit from the read-only state. The screen keeps the confirmation, the backup rename and the status messages. With this change every store read and write in internal/ui lives in workspace.go: five call sites instead of scattered ones across actions, folders, transfer and repair. Behaviour is unchanged, including the import summary and the repair flow that reloads after writing. Validated with ./scripts/check.sh.
