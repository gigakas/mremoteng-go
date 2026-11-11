---
timestamp: 2026-11-11T21:24:29Z
agent: mimocode
files:
  - .claude/skills/design-patterns/SKILL.md
  - .claude/skills/design-patterns/references/app-recipes.md
  - .claude/skills/design-patterns/references/behavioral-patterns.md
  - .claude/skills/design-patterns/references/creational-patterns.md
  - .claude/skills/design-patterns/references/language-idioms.md
  - .claude/skills/design-patterns/references/pattern-relations.md
  - .claude/skills/design-patterns/references/structural-patterns.md
  - .claude/skills/documentation/SKILL.md
  - .claude/skills/documentation/references/adr.md
  - .claude/skills/documentation/references/api-docs.md
  - .claude/skills/documentation/references/diagrams.md
  - .claude/skills/documentation/references/docstrings.md
  - .claude/skills/documentation/references/readme.md
  - .claude/skills/documentation/references/runbooks.md
  - .claude/skills/documentation/references/user-guides.md
  - .claude/skills/refactoring/SKILL.md
  - .claude/skills/refactoring/references/code-smells.md
  - .claude/skills/refactoring/references/file-responsibility-rules.md
  - .claude/skills/refactoring/references/refactoring-recipes.md
  - .claude/skills/refactoring/references/techniques.md
  - .claude/skills/solid-principles/SKILL.md
  - .claude/skills/test-driven-development/SKILL.md
  - .claude/skills/testing-patterns/SKILL.md
  - .claude/skills/testing-patterns/references/architecture-tests.md
  - .claude/skills/testing-patterns/references/mutation-testing.md
  - .claude/skills/testing-patterns/references/test-doubles.md
  - .claude/skills/testing-patterns/references/test-smells.md
  - .claude/skills/testing-patterns/references/test-strategy.md
  - .mimocode/mimocode.jsonc
---

Add the generic engineering skills adapted to this repository

Copy the language-agnostic skills (refactoring, solid-principles, design-patterns, testing-patterns, test-driven-development, documentation) from scripts/ClaudeCode-OpenCode/generic-skills into .claude/skills so Claude Code and MiMoCode share one tree, and add .mimocode/mimocode.jsonc with skills.paths pointing at it.

Adaptation done in the copies, not upstream: every generic runner invocation (./.claude/scripts/test.sh, ./.opencode/scripts/test.sh) was replaced with the repository gates (./scripts/check.sh, go test <package>, ./scripts/smoke.sh) and the four test layers were mapped onto go test ./internal/..., go test ./integration/, scripts/smoke.sh and the C# corpus. Each SKILL.md gained a 'Repo adjustments (mremoteng-go)' section with Go 1.23+, the check.sh gate (and its no-format caveat), the 50-line size rule, TestFunction_Scenario_ExpectedResult naming, headless Fyne tests, the fixture-provenance rule, the external-process boundary for RDP/AnyDesk, the auditory/ audit conventions and the fragment-only changelog workflow. The documentation skill now points at this repository's changelog and audit skills instead of the generic multi-agent-changelog and code-audit, which were deliberately not copied because they write CHANGELOG.md by hand and docs/audit/, both of which contradict AGENTS.md. The dangling references/tooling.md link (absent from the source collection) was replaced with the actual tooling of this repository.

Validated with ./scripts/check.sh.
