---
timestamp: 2026-11-12T00:43:15Z
agent: mimocode
files:
  - .claude/skills/commit-conventions/SKILL.md
  - .claude/skills/commit-conventions/references/examples.md
  - .claude/skills/commit-conventions/references/history-policy.md
  - .claude/skills/design-patterns/SKILL.md
  - .claude/skills/design-patterns/references/app-recipes.md
  - .claude/skills/design-patterns/references/language-idioms.md
  - .claude/skills/documentation/SKILL.md
  - .claude/skills/documentation/references/docstrings.md
  - .claude/skills/documentation/references/readme.md
  - .claude/skills/documentation/references/runbooks.md
  - .claude/skills/observability/SKILL.md
  - .claude/skills/observability/assets/log-event.schema.json
  - .claude/skills/observability/references/observability-model.md
  - .claude/skills/observability/references/structured-logging.md
  - .claude/skills/refactoring/SKILL.md
  - .claude/skills/refactoring/references/file-responsibility-rules.md
  - .claude/skills/refactoring/references/refactoring-recipes.md
  - .claude/skills/refactoring/references/techniques.md
  - .claude/skills/refactoring/references/tooling.md
  - .claude/skills/solid-principles/SKILL.md
  - .claude/skills/test-driven-development/SKILL.md
  - .claude/skills/testing-patterns/SKILL.md
  - .claude/skills/testing-patterns/references/architecture-tests.md
  - .claude/skills/testing-patterns/references/dart-flutter.md
  - .claude/skills/testing-patterns/references/mutation-testing.md
  - .claude/skills/testing-patterns/references/non-functional-testing.md
  - .claude/skills/testing-patterns/references/test-doubles.md
  - .claude/skills/testing-patterns/references/test-strategy.md
  - internal/terminal/terminal.go
  - internal/ui/terminal.go
---

Sync the repository skills with the current collection and adapt two new ones

The skill collection this repository copied from has moved on: it now ships ten skills, restored refactoring/references/tooling.md, and rewrote the descriptions with When NOT to use and Done when sections. Resynced the six skills the repository already had (refactoring, solid-principles, design-patterns, testing-patterns, test-driven-development, documentation) from the new master, keeping their Repo adjustments sections and translating the collection test-runner commands into this repository gate (./scripts/check.sh, go test, ./scripts/smoke.sh). Added commit-conventions and observability, both adapted rather than copied: commit-conventions documents that this repository uses plain imperative subjects instead of Conventional Commits types, forbids every trailer because AGENTS.md does, and routes verification through the changelog tool instead of a commit-msg hook, which is why the collection check-commit-msg.sh and commit-template.txt are deliberately not installed; observability documents the activity console, the status bar and the diagnostics dialog, and drops the JSONL schema checker because this desktop app has no structured log stream. Kept code-audit and multi-agent-changelog out on purpose: the repository has its own audit and changelog skills, and loading both would give the agent two competing workflows. Verified with ./scripts/check.sh.
