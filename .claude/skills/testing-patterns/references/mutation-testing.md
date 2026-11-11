# Mutation Testing

Mutation testing measures whether tests actually **detect changes** in behavior. A tool introduces small faults (mutants) into the code and runs the tests. A mutant that survives means the tests did not notice the change.

- **Killed**: at least one test failed → the test suite constrains that behavior.
- **Survived**: all tests passed → the behavior is untested or weakly asserted.
- **Equivalent**: the mutant behaves identically to the original; ignore it.

Coverage tells you what code ran. Mutation testing tells you whether the tests would catch a bug.

## When to use it

Use it on **critical modules**, not the whole repository:

- money, billing, taxes, discounts;
- authentication, authorization, tenant isolation;
- state machines and workflow transitions;
- parsers, validators, calculations;
- security-sensitive data handling.

Do not run it on glue code, generated code, or UI layout.

## Tools

| Language | Tool |
|----------|------|
| Python | `mutmut`, `cosmic-ray` |
| TypeScript/JS | Stryker (`@stryker-mutator/core`) |
| Java | PIT (`pitest`) |
| C# | Stryker.NET |
| Rust | `cargo-mutants` |
| Go | `go-mutesting`, `gremlins` |
| Ruby | `mutant` |
| PHP | Infection |

## Workflow

1. **Baseline**: run mutation testing on the target module. Record the mutation score and the surviving mutants.
2. **Triage survivors**:
   - untested branch → add a test;
   - weak assertion → assert the outcome, not that a call happened;
   - equivalent mutant → ignore with a comment;
   - dead code → delete it.
3. **Strengthen** the tests; re-run until the score is acceptable.
4. **Gate**: set a threshold only for the critical module, and start below the current score to avoid blocking.
5. **Cadence**: nightly for the critical module; PR-only on changed files when the tool supports incremental runs.

## Example

Original:

```python
def discount(amount: int, is_member: bool) -> int:
    if is_member:
        return amount * 10 // 100
    return 0
```

Mutant: `amount * 10 // 100` → `amount * 10 // 101`.

If tests only assert `discount(100, True) > 0`, the mutant survives. A test that asserts `discount(100, True) == 10` kills it.

## Reading survivors

| Survivor | Likely cause | Fix |
|----------|--------------|-----|
| Boundary changed (`>` vs `>=`) | Boundary not tested | Add the boundary test |
| Constant changed | Weak assertion or no assertion | Assert the exact outcome |
| Branch removed | Branch not covered | Add a test for that branch |
| Return value changed | Test asserts only "no exception" | Assert the value |
| Equivalent | Same behavior | Document and ignore |

## CI integration

- **Nightly**: full mutation run on critical modules.
- **PR**: incremental run on changed files if supported; report, do not block at first.
- **Release**: require a minimum score for critical modules.
- **Report**: store the score and survivors as an artifact; link it from the audit.

## Costs

- Mutation testing is **slow** (many test runs).
- It needs a fast, deterministic test suite; flaky tests produce false survivors.
- It is not a replacement for coverage or code review.
- Start small; expand only where the risk justifies the cost.

## Guardrails

- **Do not chase 100%** — equivalent mutants and low-value code make it impossible.
- **Do not run it on the whole repo** — focus on critical modules.
- **Fix the test, not the mutant** — the goal is stronger tests.
- **Do not hide survivors** — document equivalent ones with a reason.
- **Keep the suite deterministic** — flaky tests invalidate the results.
- **Pair with TDD** — a mutation score is a quality check, not a substitute for test-first.

## Checklist

- [ ] Critical modules identified.
- [ ] Tool installed and configured.
- [ ] Baseline score recorded.
- [ ] Survivors triaged.
- [ ] Tests strengthened.
- [ ] Equivalent mutants documented.
- [ ] Runs nightly or on changed files.
- [ ] Threshold set for critical modules only.
- [ ] Results linked from `docs/audit/`.
