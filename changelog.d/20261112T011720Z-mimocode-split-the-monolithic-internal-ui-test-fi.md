---
timestamp: 2026-11-12T01:17:20Z
agent: mimocode
files:
  - internal/ui/actions.go
  - internal/ui/actions_test.go
  - internal/ui/diagnostics.go
  - internal/ui/embed_test.go
  - internal/ui/folders.go
  - internal/ui/folders_test.go
  - internal/ui/helpers_test.go
  - internal/ui/panes.go
  - internal/ui/panes_test.go
  - internal/ui/quickconnect.go
  - internal/ui/rdp_test.go
  - internal/ui/screen_test.go
  - internal/ui/session.go
  - internal/ui/session_test.go
  - internal/ui/shell.go
  - internal/ui/stream_test.go
  - internal/ui/terminal_test.go
  - internal/ui/transfer_test.go
  - internal/ui/tree_test.go
  - internal/ui/ui_test.go
  - internal/ui/vault_test.go
  - internal/ui/vnc_test.go
---

Split the monolithic internal/ui test file by area

internal/ui/ui_test.go had grown to 2393 lines and 117 tests, which made every internal/ui refactor expensive: any change to a helper forced a scan of one huge file, and the seams that the next refactors need were impossible to see. Split it into thirteen files by area: screen, actions, tree, folders, session, terminal, rdp, vnc, stream, transfer, vault, embed, plus helpers_test.go for the shared fakes and constructors. The split is mechanical: a script parsed the top-level declarations, assigned each to a file by name, and computed each new import block from the package qualifiers actually used, with gofmt normalising the groups afterwards. Verified by counting test functions before and after (125 in the package, unchanged) and by the full gate. Largest test file is now 494 lines instead of 2393. No test body, assertion or name changed. Validated with ./scripts/check.sh.
