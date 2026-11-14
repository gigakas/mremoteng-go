---
timestamp: 2026-11-14T20:50:11Z
agent: mimocode
files:
  - auditory/linux-l10-sessions-settings-20261114-mimocode.md
  - blueprint/backlog.md
  - blueprint/linux-preview.md
---

Record the L10 audit for session restore and settings

The two L10 items the backlog listed as pending are delivered and audited together: the workspace stores which tabs were open and which one was selected, and File > Settings edits the colour variant, the protocol and port a new connection starts with, and whether the last tabs come back. The audit records what the restore deliberately does not do (connect, or rebuild a split layout), why ephemeral quick-connect tabs are never stored, why an unchanged tab list is not written, and that Fyne has no light/dark setting so the variant is pinned inside the theme. Blueprint and backlog move to the delivered state. Validated with ./scripts/check.sh and ./scripts/smoke.sh.
