---
timestamp: 2026-07-22T20:51:26Z
agent: opencode
files:
  - .claude/skills/changelog/SKILL.md
  - .opencode/command/changelog.md
  - .opencode/skills/changelog/SKILL.md
  - cmd/changelog/main.go
  - internal/changelog/changelog.go
  - internal/changelog/changelog_test.go
---

Require detailed descriptions in the changelog bitacora

Added a Description field to the changelog Entry struct, parsed from the fragment body after a blank-line separator (first paragraph = summary, rest = description). Render outputs the description as an indented paragraph between the summary and the file list. The CLI now accepts -description and warns when it is omitted so the bitacora stays complete. Added four unit tests covering parsing with/without description and both render layouts. Created .opencode/skills/changelog/SKILL.md mandating detailed descriptions for all future changes.
