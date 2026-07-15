# CLAUDE.md

This project's instructions are shared across agents and live in AGENTS.md —
reading them is mandatory:

@AGENTS.md

## Claude Code specifics

- Project skills: `/changelog` (record a change), `/check`
  (gofmt+vet+tests), `/smoke` (smoke test), `/audit` (stage closing audit).
- Active hooks (`.claude/settings.json`): automatic gofmt after every
  Edit/Write of Go files, and a Stop hook that blocks ending the turn when
  there are changes without a changelog fragment — if it blocks you, run the
  `/changelog` skill.
- When using `cmd/changelog`, your agent name is `claude-code`.
