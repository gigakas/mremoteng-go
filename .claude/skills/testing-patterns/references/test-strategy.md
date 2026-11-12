# Test Strategy

How to decide **what** to test, **how much**, and **when** — without turning coverage into a number game. Use this with `test-driven-development` (the cycle) and `testing-patterns` (how to write each test).

## Core idea: tests are a portfolio

Different tests buy different confidence at different cost. Choose the cheapest test that can catch the failure you care about.

| Type | Scope | Speed | Cost | Confidence | Use for |
|------|-------|-------|------|------------|---------|
| **Unit** | One function/class, dependencies doubled | ms | Low | Logic correctness | Business rules, edge cases, algorithms |
| **Integration** | A few real components (DB, HTTP, queue) | 100 ms–s | Medium | Wiring and contracts | Repositories, adapters, transactions |
| **Contract** | Producer/consumer agreement | ms–s | Medium | Compatibility | APIs, events, shared schemas |
| **E2E** | Whole system through the UI/API | s–min | High | User-visible flow | Critical journeys only |
| **Smoke** | "Does it start and respond?" | s | Low | Deployability | Post-deploy, environment checks |
| **Performance** | Latency/throughput under load | min | High | Non-functional budget | Critical paths with SLOs |
| **Security** | Authn/authz and input handling | ms–s | Medium | Access control | Permissions, tenant isolation |

Default shape: **many unit, some integration, few E2E**. Do not invert it unless the system is mostly glue (then integration tests carry more value).

Performance, security and contract testing have their own playbook: [`non-functional-testing.md`](non-functional-testing.md) — budgets, cadence, tooling and reporting.

## The four required layers

This collection expects four layers. `unit`, `integration`, and `system` are **scopes**; `regression` is a **purpose** that can live in any of the other three. What makes a test a regression test is that it locks a bug fix or existing behavior so it cannot silently break again.

| Layer | Question it answers | Isolation | Data | Speed | Run when |
|-------|--------------------|-----------|------|-------|----------|
| **Unit** | Does this rule/function behave correctly? | No real I/O; doubles at the boundary | Fixtures/factories | ms per test | Every edit and commit |
| **Integration** | Do the real components work together? | Real adapters; disposable DB/queue/HTTP | Seeded/transactional | 100 ms–s | PR / pre-push |
| **System** | Does the whole app work in a production-like environment? | Full app + real services | Seeded environment | s–min | Main, nightly, pre-release |
| **Regression** | Did we break something that already worked? | Same as the test it grew from | Same as the test it grew from | Depends on scope | Every PR + before release |

### What belongs in each layer

- **Unit**: business rules, validation, calculations, state transitions, parsers, pure functions. One behavior per test; no network, clock, filesystem, or database.
- **Integration**: repositories, adapters, transactions, serializers, migrations, message consumers, HTTP handlers with a real router. Use a disposable database or container.
- **System**: the critical user journeys end to end (login → create → pay → confirm), deployment smoke, permissions across the real stack. Keep it small; these are expensive.
- **Regression**: one test per fixed bug, named after the bug/behavior, plus characterization tests for legacy code before refactoring. A regression test must fail on the old code and pass on the fix.

### Directory layout

Prefer explicit directories when the language allows it:

```
tests/
  unit/
  integration/
  system/
  regression/
```

When the language convention keeps tests next to the code (Go, Rust, Java, C#), use tags/categories instead of moving files:

| Language | Mechanism | Example |
|----------|-----------|---------|
| Python | directories or pytest markers | `@pytest.mark.integration` |
| TypeScript/JS | directories or Vitest/Jest projects | `vitest run tests/integration` |
| Go | build tags | `//go:build integration` + `go test -tags=integration ./...` |
| Rust | integration test files | `tests/integration.rs` → `cargo test --test integration` |
| Java | JUnit tags | `@Tag("integration")` |
| C# | category traits | `[Category("Integration")]` |
| Ruby | directories or RSpec tags | `spec/integration/` |
| PHP | PHPUnit testsuites | `<testsuite name="integration">` |
| Dart/Flutter | tags (`@Tags(["integration"])`) or directories | `dart test --tags integration` / `flutter test test/integration` |

The runner below understands both directory-based and tag-based setups; override the command with `TEST_<LAYER>_CMD` when your project uses a different convention.

### Runner

The repository maps the layers onto Go tooling:

```bash
# from the repository root
./scripts/check.sh                      # formatting check -> vet -> go test ./... (all layers)
go test ./internal/...                  # unit only
go test ./integration/                  # integration: C# compatibility corpus
./scripts/smoke.sh                      # system: build, skeleton startup, changelog check
go test ./integration/ -run '^TestCSharpCorpus_' -count=1 -v   # regression
go test ./internal/serialize/xml/ -run '^TestName$' -count=1   # one test
```

Extra args after `--` apply to a **single-layer** run only; multi-layer runs reject them so a filter cannot silently skip a layer.

Environment overrides:

| Variable | Purpose |
|----------|---------|
| `TEST_UNIT_CMD` | Exact command for the unit layer |
| `TEST_INTEGRATION_CMD` | Exact command for the integration layer |
| `TEST_SYSTEM_CMD` | Exact command for the system layer |
| `TEST_REGRESSION_CMD` | Exact command for the regression layer |
| `TEST_TIMEOUT` | Per-layer timeout in seconds (uses `timeout` when available) |
| `TEST_STRICT=1` | A missing layer is a failure instead of a skip |
| `PROJECT_ROOT` | Override project root detection |
| `PYTHON_BIN` | Python interpreter (default: `python3`, then `python`) |

The runner is **fail-open for missing layers**: if a layer has no command or directory, it prints `SKIP` and continues. Use `TEST_STRICT=1` in CI once all four layers exist.

### Cadence

| Moment | Layers |
|--------|--------|
| While editing | Unit (focused) |
| Before commit | Unit + regression |
| Pull request | Unit + integration + regression |
| Main branch | All four |
| Nightly | All four + performance/security if configured |
| Before release | All four + system on a production-like environment |

### Regression suite rules

1. Every bug fix adds a regression test that **fails before** the fix and passes after.
2. Name it after the behavior, not the ticket: `test_rejects_duplicate_invoice_number`.
3. Keep the regression suite fast; move slow end-to-end cases to the system layer.
4. Never delete a regression test without replacing the behavior coverage it provided.
5. Characterization tests for legacy code count as regression tests: they pin current behavior before a refactor.

## Type of change → test to write

| Change | Start with | Also consider | Avoid |
|--------|------------|---------------|-------|
| **New feature** | Unit tests for the core rule | One integration test for wiring; one E2E happy path if UI | Testing every UI permutation |
| **Bug fix** | A regression test that reproduces the bug **first** | Integration test if the bug crossed a boundary | Fixing without a failing test |
| **Refactor** | Existing tests must stay green | Characterization tests if coverage is missing | Adding new behavior in the same change |
| **API change** | Contract test for the new shape | Integration test for the endpoint | Snapshotting the whole response blindly |
| **DB migration** | Migration up/down on a disposable DB | Data backfill test | Running migrations against shared dev data |
| **UI change** | Component test for behavior | One E2E for the critical journey | E2E for styling or copy |
| **Config/infra** | Smoke test that the service starts | Health-check assertion | Unit-testing YAML |
| **Performance-sensitive** | Benchmark with a threshold | Load test on the critical path | Asserting exact timings in CI |
| **Security-sensitive** | Permission/auth tests per role | Tenant-isolation test | Testing only as Administrator |
| **Data import/export** | Round-trip test with a fixture file | Large-file boundary test | Real production data |

## Coverage policy

Coverage is a **diagnostic**, not a target.

- Use it to find **untested risk**, not to reward lines executed.
- Prioritize: money, auth/permissions, data loss, state transitions, validation boundaries, error paths.
- A module at 60% coverage with all critical branches tested is healthier than 95% with tautological tests.
- Do not add a coverage gate that can be gamed by asserting getters.
- If you set a gate, set it **per critical package**, not repo-wide.
- New code should not lower the coverage of its package; that is the actionable rule.

## CI quality gates

Keep the pull-request gate fast; move slow checks to main/nightly.

| Gate | PR | Main/nightly |
|------|----|--------------|
| Format/lint | Yes | Yes |
| Type check | Yes | Yes |
| Unit tests | Yes | Yes |
| Integration tests | If fast (< 2 min) | Yes |
| Contract tests | If fast | Yes |
| E2E critical path | No | Yes |
| Full E2E matrix | No | Nightly |
| Performance/load | No | Nightly/release |
| Migration up/down | No | Yes |

Rules:

- A flaky test is a **bug in the test**; do not "fix" it with retries.
- Do not merge with a red suite; quarantine only with an owner and an expiry date.
- Test output must name the failing behavior, not just the line.

## Definition of Done (testing)

A change is done when:

- [ ] The new behavior has at least one test that would fail without the change.
- [ ] The test asserts behavior, not implementation details.
- [ ] Error and boundary paths are covered where they matter.
- [ ] Tests are deterministic (no real clock, network, randomness, or order dependence).
- [ ] External systems are doubled at the boundary, not deep inside.
- [ ] Test names describe the expected behavior.
- [ ] The fast suite still runs in a reasonable time.
- [ ] If a refactor was involved, tests stayed green throughout.
- [ ] If coverage was missing in the touched area, characterization tests were added.

## When NOT to test

- **Generated code** (protobuf, OpenAPI clients, ORM models) — test the generator or the integration seam, not the output.
- **Third-party libraries** — test your usage, not their behavior.
- **Trivial glue** (one-line forwarding, getters/setters) — unless it encodes a rule.
- **Throwaway prototypes** — but mark them as such and do not ship them.
- **Pure configuration** — use a smoke test instead of unit tests.

The exception: if the "trivial" seam is where a real failure happens (auth, money, data loss), test the seam.

## Test data

- Prefer **factories/builders** over shared fixture dumps; each test declares only the data it needs.
- Use **unique identifiers** (UUID suffix) when the store is shared.
- Clean up or roll back; never depend on test execution order.
- Keep one canonical fixture for stable reference data; do not mutate it.
- For Frappe, see `frappe-testing` and its `fixtures.md`.

## Naming and layout

| Language | Test file | Test name style |
|----------|-----------|-----------------|
| Python | `test_<module>.py` | `test_<behavior>` |
| TypeScript/JS | `<module>.test.ts` / `.spec.ts` | `it("returns zero for a new account")` |
| Go | `<module>_test.go` | `TestXxx_Behavior` |
| Rust | inline `#[cfg(test)]` or `tests/` | `it_returns_zero_for_new_account` |
| Java | `XxxTest.java` | `shouldReturnZeroForNewAccount` |
| C# | `XxxTests.cs` | `Returns_zero_for_new_account` |
| Ruby | `*_spec.rb` / `*_test.rb` | `it "returns zero for a new account"` |
| PHP | `*Test.php` | `testReturnsZeroForNewAccount` |
| Dart/Flutter | `<module>_test.dart` | `test('returns zero for a new account', ...)` |

Follow the project's existing convention first; consistency beats this table.

Dart/Flutter specifics — widget, golden, generated-code and device-bound integration tests: [`dart-flutter.md`](dart-flutter.md).

## Runners per language

| Language | Fast run | Full run | Coverage |
|----------|----------|----------|----------|
| Python | `pytest -q` | `pytest` | `pytest --cov=<pkg>` |
| TypeScript/JS | `npx vitest run` / `npx jest` | same | `--coverage` |
| Go | `go test ./...` | `go test -race ./...` | `go test -cover ./...` |
| Rust | `cargo test` | `cargo test --all` | `cargo llvm-cov` |
| Java | `mvn -q test` | `mvn verify` | JaCoCo |
| C# | `dotnet test` | `dotnet test --no-build` | coverlet |
| Ruby | `bundle exec rspec` | `bundle exec rspec` | SimpleCov |
| PHP | `vendor/bin/phpunit` | `vendor/bin/phpunit` | PHPUnit coverage |
| Dart/Flutter | `dart test` / `flutter test` | same | `flutter test --coverage` (lcov) |
| Frappe | `bench --site <site> run-tests --module <module>` | `bench --site <site> run-tests --app <app>` | — |

Run the **focused** test while developing; run the **full** suite before committing.

## Strategy checklist

- [ ] I know which failure this change can introduce.
- [ ] I chose the cheapest test type that catches it.
- [ ] The test fails for the right reason before the fix.
- [ ] The test is deterministic and isolated.
- [ ] Critical error/boundary paths are covered.
- [ ] I am not chasing a coverage number.
- [ ] The fast suite remains fast.

## References

- `test-driven-development` — red/green/refactor cycle and characterization tests
- `testing-patterns` — AAA, FIRST, test doubles, test smells
- `testing-patterns/references/test-doubles.md` — dummy, stub, spy, mock, fake
- `testing-patterns/references/test-smells.md` — flaky, fragile, slow, tautological tests
- `refactoring` — characterization tests as the safety net before refactoring
- `frappe-testing` — Frappe-specific commands, fixtures, permissions, Cypress
- `testing-patterns/references/non-functional-testing.md` — load, security, contract, accessibility and reliability tests
- `testing-patterns/references/dart-flutter.md` — Dart/Flutter layers, widget/golden/integration tests, coverage and CI
