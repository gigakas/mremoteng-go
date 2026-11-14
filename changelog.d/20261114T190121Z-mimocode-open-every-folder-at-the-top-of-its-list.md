---
timestamp: 2026-11-14T19:01:21Z
agent: mimocode
files:
  - internal/ui/filepane.go
  - internal/ui/filepane_test.go
---

Open every folder at the top of its listing

Reported as a folder with few files appearing scrolled past the top, so the list had to be scrolled up before its contents could be seen. The pane rebuilt its rows on every navigation but the scroll container kept the previous folder's offset, and a short listing inherited a long one's position. The pane keeps the scroll now and returns it to the top whenever the rows are rebuilt, which covers navigation and re-ordering. TestFilePane_Navigate_ResetsTheScrollToTheTop fills a folder with forty entries, scrolls down, opens a folder with one file and fails unless the offset is back at zero. Validated with ./scripts/check.sh, without Fyne errors in its output.
