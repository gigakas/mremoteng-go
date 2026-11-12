---
timestamp: 2026-11-12T01:36:23Z
agent: mimocode
files:
  - internal/terminal/terminal.go
  - internal/terminal/terminal_test.go
---

Record scrollback for output that arrives in a single large block

Found while testing the terminal: a cat of a big file, a build log or any output larger than the screen filled the visible rows but recorded no scrollback, so the user could not scroll back through it. Emulator.Write fed the whole payload and then ran trackScroll, which rebuilds history by aligning the previous screen with the current one; that alignment only searches shifts below the screen height, so a block that scrolled past a full screen was invisible to it. Write now feeds the payload in steps that end at a newline and carry at most half a screen of them, so every step is alignable, and it keeps the erase-scrollback scan over the whole payload rather than the remainder. The step is half a screen and not a full one because at rows-1 only a single row remains to compare and it is usually blank right after a scroll, which made the first attempt still record nothing. Tests were written first: TestEmulator_WriteLargeBlock_RecordsTheWholeScrollback failed before the change and now asserts 198 of 200 lines on a three-row screen, the arithmetic being N-rows+1 because the last lines stay visible. Added three more cases: output that fits records nothing, a huge block caps at historyLimit, and History returns the scrollback without the visible screen. Two findings worth recording: a run of identical lines is not recorded at all, because every screen then looks the same and the tracker reads it as a redraw, and the pre-existing bound test was vacuous for the same reason, asserting zero is at most historyLimit. Written test first, then the fix, validated with ./scripts/check.sh.
