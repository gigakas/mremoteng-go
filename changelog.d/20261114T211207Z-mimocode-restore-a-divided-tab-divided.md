---
timestamp: 2026-11-14T21:12:07Z
agent: mimocode
files:
  - internal/profiles/profiles.go
  - internal/profiles/store.go
  - internal/profiles/profiles_test.go
  - internal/ui/panes.go
  - internal/ui/panes_test.go
  - internal/ui/workspace.go
  - internal/ui/shell.go
  - internal/ui/helpers_test.go
  - internal/ui/screen_test.go
  - auditory/linux-l10-layout-20261114-mimocode.md
  - blueprint/backlog.md
  - blueprint/linux-preview.md
---

Restore a divided tab divided

Reopening a workspace brought back every tab as a single pane, so a tab that had two or three sessions came back as one; the layout was the half of session persistence the first version left out. The profile document gains tab_layouts, a split tree per divided connection, written whenever a pane is split or closed and rebuilt at startup. Panes are restored without connecting, like the tabs themselves: no credential is stored, so each pane waits for its own Connect. A stored shape is validated on save and again on restore, and one this build cannot rebuild — a split missing a side, a tree over twelve panes, or an RDP tab that claims it was divided — costs the split, never the tab. Pane sizes are not stored, so a split comes back at half. TestScreen_SplitLayout_IsPersistedAndRestored, TestScreen_RestoringPanes_KeepsTheTabPointingAtTheRoot and the store round trip cover the shape, the tab bookkeeping after a restore and the refusal cases. Validated with ./scripts/check.sh.
