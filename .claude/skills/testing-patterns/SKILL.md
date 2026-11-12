---
name: testing-patterns
description: Use when writing tests or diagnosing flaky, slow, or brittle ones. Covers Arrange-Act-Assert, FIRST, test doubles (fakes, stubs, mocks), test smells, coverage policy, mutation, architecture and non-functional tests (load, access control, contracts). Language-agnostic.
---

# Testing Patterns

How to structure tests and keep them fast, isolated, and meaningful — in any language.

## When to use

- Writing new tests and want them clean from the start
- Tests are flaky, slow, or brittle and need diagnosis
- Deciding how to isolate a unit from its dependencies (test doubles)
- Reviewing test quality and coverage
- Designing load, access-control, contract or accessibility test budgets

## When NOT to use

- Deciding *what* to test or *how much* — that is `references/test-strategy.md` (this file is *how* to write each test).
- Writing the first test of a feature — that is `test-driven-development` (the red/green/refactor cycle).
- Performance tuning, security remediation, or accessibility fixes — the playbook covers how to *test* budgets, access control and compatibility, not how to optimize or fix what the tests find.
- A broken CI pipeline — fix the pipeline, not the test.

## Test structure: Arrange / Act / Assert

Keep every test in three clearly separated phases (also called Given / When / Then):

```
# Arrange — set up fixtures and dependencies
# Act     — perform the single action under test
# Assert  — verify the outcome
```

Only one logical action per test. If you need multiple "acts", you probably need multiple tests.

## FIRST principles

A good test is:

| Letter | Principle | Meaning |
|--------|-----------|---------|
| F | **Fast** | Runs in milliseconds so the whole suite stays quick |
| I | **Independent** | No reliance on order or other tests' side effects |
| R | **Repeatable** | Same result every run (no time/network/randomness) |
| S | **Self-validating** | Passes/fails on its own — no manual inspection |
| T | **Timely** | Written close to the code (ideally before, via TDD) |

## Test doubles

When a unit depends on something slow, external, or hard to trigger, replace the dependency with a *test double*. Choose the right kind — see [references/test-doubles.md](references/test-doubles.md).

Quick taxonomy:

| Double | What it does | Typical use |
|--------|--------------|-------------|
| **Dummy** | Passed but never used | Fill a required parameter |
| **Stub** | Returns a canned value | Provide a fixed input |
| **Spy** | Records calls, returns real result | Assert "was it called?" |
| **Mock** | Returns canned value + verifies expectations | Assert "called with these args?" |
| **Fake** | Working but simplified implementation | In-memory DB, stub server |

Prefer stubs/fakes for *state* verification and mocks for *interaction* verification. Don't mock types you don't own (e.g., third-party libraries) — wrap them instead.

## Test smells

Common signs of problem tests — see [references/test-smells.md](references/test-smells.md) for the full list. The most damaging:

- **Flaky** — passes and fails unpredictably (time, order, network, concurrency).
- **Fragile / overspecified** — breaks when production code changes for reasons unrelated to the test's intent.
- **Slow** — drags the suite down and discourages running tests.
- **Tautological / "test the test"** — asserts what the setup already guaranteed.
- **Obscure** — hard to tell what's being tested and why.

## Coverage: what to measure

- Coverage tells you what is **not** tested; it does **not** tell you the tests are good.
- Aim for meaningful behavioral coverage of logic and branches, not a magic number.
- Prioritize high-risk paths (validation, money/security, edge cases) over trivial getters.

## Non-functional testing

Functional tests protect behavior; budgets protect the product. Keep one budget per critical path and make it executable:

| Question | Test | Cadence |
|---|---|---|
| p95 latency at peak | Load test with thresholds in the script | Nightly |
| Can tenant A read tenant B's data? | Parameterized authz matrix (roles × objects × tenants) | Pull request |
| Do old consumers still work? | Contract test or schema validation | Pull request |
| Does it survive 4 hours at peak? | Soak test | Release |
| Can we restore after data loss? | Restore drill with measured time | Quarterly |

Full playbook: [references/non-functional-testing.md](references/non-functional-testing.md) — load/stress/soak, threat-modelled authorization, Pact and schemathesis, the accessibility minimum, reliability drills, cadence and reporting.

## References

- [references/test-strategy.md](references/test-strategy.md) — how to choose test types, the four required layers, coverage policy, CI gates, runners and Definition of Done
- [references/test-doubles.md](references/test-doubles.md) — the five doubles, when to use them, per-language tooling
- [references/test-smells.md](references/test-smells.md) — catalogue of test anti-patterns and fixes
- [references/architecture-tests.md](references/architecture-tests.md) — executable dependency/layer rules per language
- [references/mutation-testing.md](references/mutation-testing.md) — measuring whether tests actually catch changes
- [references/non-functional-testing.md](references/non-functional-testing.md) — performance budgets, security/authz matrices, contract testing, accessibility minimum, reliability drills
- [references/dart-flutter.md](references/dart-flutter.md) — Dart/Flutter: widget, golden and integration tests, mocks, generated code, coverage

## Guardrails

- **Isolate tests** — a test must not depend on another test's data or order.
- **Never hit real external systems** — network, clock, filesystem, DB (unless it's an integration test).
- **Mock the boundary, not the internals** — replace external I/O, not your own small classes.
- **One behavior per test** — more than a few assertions usually means split the test.
- **Keep asserts meaningful** — assert outcomes, not that a line executed.

## Common mistakes

| Mistake | Why It Fails | Fix |
|---------|--------------|-----|
| Mocking the wrong layer | Tests lock in implementation details | Mock at the external boundary only |
| Overusing mocks | Tests pass while real code is broken | Prefer stubs/fakes; add integration tests |
| Asserting setup, not behavior | Tautological tests give false confidence | Assert a real outcome of the action |
| Non-deterministic tests | Flaky CI, lost trust in the suite | Inject clock/randomness; avoid real time/network |
| Tests sharing mutable state | Order-dependent failures | Fresh fixtures per test |

## Done when

- [ ] The test names the behavior and asserts an outcome, not a call.
- [ ] It is deterministic: no real clock, network, randomness, or order dependence.
- [ ] Dependencies are doubled at the boundary, not deep inside.
- [ ] It fails when the behavior breaks (checked, not assumed).
- [ ] The suite it joined is still fast enough to run on every edit.

## Repo adjustments (mremoteng-go)

- **Toolchain**: Go 1.25+ (`go.mod` declares `go 1.25.0`).

The four layers map onto what this repository actually runs:

| Layer | Here |
|---|---|
| Unit | `go test ./internal/...` |
| Integration | `go test ./integration/` (C# compatibility corpus) |
| System | `./scripts/smoke.sh` (build, skeleton startup, changelog check) |
| Regression | `-run '^TestCSharpCorpus_'`, plus a failing test first on every bug fix |

- **Gate**: `./scripts/check.sh` is mandatory before finishing. It does not format — `gofmt -w <files>`.
- **Naming**: `TestFunction_Scenario_ExpectedResult`. New packages need unit tests.
- **UI tests stay headless**: Fyne's software driver only. Never open a real window, call `Canvas().Capture()` in a long-lived test, or require `$DISPLAY`.
- **Fixtures**: acceptance fixtures come from the C# app, never from the Go writer.
- **Discarded errors** need a justification comment; a bare `_ =` will not survive review.
- **Changelog**: see the `changelog` skill.
