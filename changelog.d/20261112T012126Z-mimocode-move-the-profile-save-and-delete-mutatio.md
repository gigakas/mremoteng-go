---
timestamp: 2026-11-12T01:21:26Z
agent: mimocode
files:
  - internal/ui/actions.go
  - internal/ui/workspace.go
---

Move the profile save and delete mutations into the workspace

Phase B1 of the screen extraction. save and deleteProfile each computed the next profile slice, wrote the document, assigned the new slices and then refreshed the widgets, all in the same method. The workspace now owns the mutation: commit writes the candidate document and only replaces the in-memory state after the store accepts it, and putProfile and deleteProfile are built on top of commit, with putProfile also adding the profile folder entry. The screen keeps what is genuinely its own: form validation, id generation, selection, tree refresh, tab teardown and the status line. Behaviour is unchanged, including the invariant that a rejected write leaves the workspace and the UI exactly as they were. actions.go loses its two inline writes and one assignment block. Validated with ./scripts/check.sh.
