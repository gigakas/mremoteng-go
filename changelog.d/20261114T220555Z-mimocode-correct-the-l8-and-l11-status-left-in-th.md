---
timestamp: 2026-11-14T22:05:55Z
agent: mimocode
files:
  - blueprint/linux-preview.md
  - blueprint/backlog.md
---

Correct the L8 and L11 status left in the plan

Two planning statements had drifted from the code. blueprint/linux-preview.md still listed the embedded webview and serial ports as open after both were delivered today, leaving modifier-key input as the only item from that list. blueprint/backlog.md still called the store repair path pending although Tools > Repair connection file has moved a damaged store aside and started empty since the repair flow landed. Both are corrected and the L11 line now names what is actually left: a migration test, the copy-diagnostics action and proof that no X11 child or FreeRDP process survives a crash. Documentation only, validated with ./scripts/check.sh.
