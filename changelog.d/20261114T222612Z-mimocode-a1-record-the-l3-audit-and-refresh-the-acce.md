---
timestamp: 2026-11-14T22:26:12Z
agent: mimocode-a1
files:
  - auditory/linux-l3-20261115-mimocode-a1.md
  - docs/linux-preview-usability.md
---

Record the L3 audit and refresh the acceptance runbook

L3 was the one stage of the active plan with no audit, because its scope was delivered in pieces by later increments. The audit writes that down: what L3 owns (import/export, folder defaults, the credential vault, quick connect, folder management, terminal zoom and scrollback, restored session tabs with their layout, the settings dialog, the notifications panel and manual tree ordering), where each piece lives, and the gaps a reader should know about - no L3-owned end-to-end test, the two one-field dialogs that should merge, the stale wheel-zoom comment and the fact that -profiles does not redirect the vault. The usability runbook gained scenarios 19-25 for everything that landed after it was written (settings, session restore, notifications, manual order, the embedded web view, serial as optional, and the RDP chords, full screen and per-profile options) plus a known-limits list, and two of its claims were corrected the same day: a divider position is now remembered and serial connections do offer a handshake. Validated with ./scripts/check.sh.
