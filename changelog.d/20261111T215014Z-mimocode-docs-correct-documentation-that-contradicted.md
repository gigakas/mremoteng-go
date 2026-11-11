---
timestamp: 2026-11-11T21:50:14Z
agent: mimocode-docs
files:
  - AGENTS.md
  - blueprint/linux-preview.md
  - docs/linux-preview-usability.md
  - .claude/skills/refactoring/SKILL.md
  - .claude/skills/solid-principles/SKILL.md
  - .claude/skills/design-patterns/SKILL.md
  - .claude/skills/testing-patterns/SKILL.md
  - .claude/skills/test-driven-development/SKILL.md
  - .claude/skills/documentation/SKILL.md
---

Correct documentation that contradicted the implemented code

blueprint/linux-preview.md: L2 now names the real gaps (modifier-key input, embedded webview, serial) instead of the already delivered RDP embedding and scrollback, and the table marks L2 and L3 in progress and L4 partially done, each only where the code proves it; L1 stays open for the pending human run, the L2 ownership bullet adds internal/terminal and the post-L4 packages, the moved-connection path no longer points at a File menu item that does not exist, and the terminal history, zoom and copy behaviour is documented with its limits.
AGENTS.md: the toolchain floor is Go 1.25+ because go.mod declares go 1.25.0, so the old 1.23+ line sent readers to a version the module cannot build with.
docs/linux-preview-usability.md: scenarios 11-18 cover folders (create, subfolder, subtree rename, recursive delete with its count, drag and Move to), the import/export round trip on scratch files, quick connect, terminal zoom, scrollback and copy, and the master-password vault, each with an observable pass condition; the -profiles preparation block and scenarios 1-10 are kept, and the script stays a pending human checklist that keeps L1 open instead of claiming the old session out-of-scope preview.
.claude/skills/{refactoring,solid-principles,design-patterns,testing-patterns,test-driven-development,documentation}/SKILL.md: the repo-adjustment sections repeat the toolchain fact as Go 1.25+ with the go.mod declaration, so the six generic engineering skills stop restating a floor the module no longer supports.
