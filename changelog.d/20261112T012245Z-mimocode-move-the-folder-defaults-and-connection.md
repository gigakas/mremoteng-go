---
timestamp: 2026-11-12T01:22:45Z
agent: mimocode
files:
  - internal/ui/actions.go
  - internal/ui/folders.go
  - internal/ui/workspace.go
---

Move the folder, defaults and connection mutations into the workspace

Phase B2 of the screen extraction. Five folder mutators and one connection mutator repeated the same shape: validate, build the next slices, write the document, assign the slices, refresh the widgets. The workspace now owns the computation and the write for all of them: addFolder, setFolderDefaults, renameFolder, deleteFolder, putConnection and renameProfile, each built on commit or commitWith, which write the candidate document and only then replace the in-memory state. The screen keeps the user-facing validation and every widget effect. Two details kept deliberately: deleteFolder still closes the session tabs before attempting the write, so a rejected write leaves the tabs closed exactly as before, and renameProfile does not go through putConnection because a rename must not add folder entries. actions.go and folders.go lose their inline writes; the only direct store writes left are the import and the repair path, which is phase B3. Validated with ./scripts/check.sh.
