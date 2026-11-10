---
timestamp: 2026-07-22T21:04:24Z
agent: claude-code
files:
  - .claude/skills/changelog/SKILL.md
---

Align changelog skill docs with the new -description flag

OpenCode added a -description flag to cmd/changelog (Entry.Description, parsed as the second paragraph of the fragment body, rendered as an indented block under the summary bullet) in commit f5285d7, landed concurrently with my own doc-only attempt at the same requirement (require explanatory changelog entries). Updated .claude/skills/changelog/SKILL.md to document -description instead of asking for a what+why explanation crammed into the one-line -summary, so the skill matches the actual tool contract and doesn't contradict OpenCode's .opencode/skills/changelog/SKILL.md.
