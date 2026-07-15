# auditory/ — Stage closing audits

Every time a **stage** of a blueprint phase is finished, the agent that
worked on it produces an audit here evaluating **code quality**,
**performance** and **architecture**. A stage without an audit is not done.

## Naming convention

```
phase<N>-stage<M>-YYYYMMDD-<agent>.md
```

Examples: `phase1-stage2-20260801-claude-code.md`,
`phase2-stage1-20260815-opencode.md`.

## Rules

- Always start from `TEMPLATE.md`; fill in every section.
- Concrete findings with `file:line` — no generic assessments.
- Audits are immutable once committed: if a re-audit is needed, create a new
  file with a new date; never edit the old one.
- The audit is recorded in the shared changelog like any other change.
- If the stage completes its phase, closing includes updating the top-level
  `README.md`.
