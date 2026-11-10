---
timestamp: 2026-07-22T21:22:48Z
agent: claude-code
files:
  - .claude/skills/changelog/SKILL.md
  - .opencode/command/changelog.md
  - internal/protocol/factory.go
  - internal/protocol/protocol.go
  - internal/protocol/protocol_test.go
---

Also require -description in the OpenCode changelog command

Following up on the earlier skill fix: .opencode/command/changelog.md (the older command file, distinct from OpenCode's own new .opencode/skills/changelog/SKILL.md) still told agents to write a single-line summary explaining what+why, contradicting the tool's actual -description flag. Rewrote its example command and rules to require -description explicitly, mirroring both .claude/skills/changelog/SKILL.md and .opencode/skills/changelog/SKILL.md, and added a note that $ARGUMENTS (the raw slash-command input) should be turned into -summary/-description rather than passed through verbatim.
