# Audit — Phase <N>, Stage <M>

- **Date (UTC)**: YYYY-MM-DD
- **Agent**: claude-code | opencode | human
- **Audited stage**: <stage title as in blueprint/phase-N-*.md>
- **Commits covered**: <range or hashes>

## 1. Code quality

<Concrete findings with file:line. Duplication, complexity, error handling,
naming, test coverage of the new code, technical debt introduced and its
justification.>

## 2. Performance

<Unnecessary allocations, I/O on hot paths, algorithmic costs. If measured:
benchmark command and numbers. If not applicable, explain why.>

## 3. Architecture

<Package boundary compliance (internal/*), new dependencies and their
justification, deviations from the blueprint and why, impact on future
phases.>

## 4. Evidence

- `./scripts/check.sh`: <result>
- `./scripts/smoke.sh`: <result>
- New tests in this stage: <list, or "none" with justification>

## 5. Verdict

- [ ] Stage closed unconditionally
- [ ] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

<Concrete list with suggested owner, or "none". If the phase is now
complete: confirm the top-level README.md was updated.>
