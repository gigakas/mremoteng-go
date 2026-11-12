# Non-Functional Testing

Functional tests answer *"does it work?"*. Non-functional tests answer *"does it hold?"* — under load, under attack, and across independent deploys.

The rule: **one budget per critical path, not a matrix**. A latency number you can defend, an authorization matrix you can execute, and a contract you can break on purpose are worth more than a thousand exploratory checks.

## Start from a budget, not a tool

| NFR | Question | Budget (example) | Measured by |
|---|---|---|---|
| **Latency** | How slow is the critical path at peak? | p95 < 300 ms at 200 rps | Load test with thresholds |
| **Throughput** | How much can it take before degrading? | 500 rps sustained, error rate < 1% | Stress test |
| **Stability** | Does it leak or drift over time? | RSS growth < 5% over 4 h | Soak test |
| **Availability** | What is the error rate in steady state? | < 0.1% 5xx over 24 h | Smoke probes / soak |
| **Access control** | Can a role read what it must not? | 0 cross-tenant reads | Parameterized authz tests |
| **Compatibility** | Can old consumers still call the new producer? | v1 client passes against v2 | Contract tests |
| **Recoverability** | How long to restore after data loss? | RTO < 30 min, RPO < 5 min | Restore drill |
| **Accessibility** | Can the journey be completed without a mouse? | Keyboard-only, 200% zoom | Manual pass + axe |

If you cannot put a number (or a binary assertion) on it, you do not have a test — you have an intention. Write the budget in the ticket before writing the test.

## Performance and load

### Test types

| Type | Shape | Answers |
|---|---|---|
| **Smoke** | 1–5 users, seconds | Does it serve at all under a load tool? |
| **Load** | Expected peak, 10–30 min | Does it meet the latency budget at peak? |
| **Stress** | Ramp beyond peak until failure | Where does it break, and how? |
| **Soak** | Expected load, 2–8 h | Leaks, queue growth, connection exhaustion |
| **Spike** | Sudden burst, short | Does autoscaling/backpressure cope? |

Run **load** and **stress** against a production-like environment on nightly/release. Never assert exact timings in pull-request CI: shared runners are noisy and the test becomes a coin flip.

### Tooling

| Tool | Language | Best for |
|---|---|---|
| **k6** | JavaScript | Scripted scenarios with thresholds in code; good default |
| **locust** | Python | Custom user behavior, teams already in Python |
| **Gatling** | Scala/JVM | High concurrency, JVM shops |
| **vegeta** / **hey** | CLI | Quick fixed-rate HTTP probes |
| **autocannon** | Node | Single-endpoint Node services |
| **JMeter** | GUI/XML | Legacy estates, protocol breadth |

### Rules that make numbers trustworthy

1. **Warm up** and discard the first 30–60 s (JIT, caches, pools).
2. **Report p95/p99**, never the mean; a good average hides the users who waited 4 s.
3. **Keep the generator out of the bottleneck**: watch its CPU and network first.
4. **Match the data class**: 10 rows is not production. Document the difference when you cannot.
5. **Three to five scenarios**, one per critical journey — not one per endpoint.
6. **Correlate** the latency curve with DB, CPU, memory, and queue depth; a number without a cause cannot be fixed.
7. **Thresholds live in the script**, so nightly fails on regression instead of needing a human to read a chart.

```javascript
// k6: the budget is executable
import http from "k6/http";
import { check } from "k6";

export const options = {
  scenarios: { peak: { executor: "constant-arrival-rate", rate: 200, timeUnit: "1s", duration: "10m", preAllocatedVUs: 50 } },
  thresholds: {
    http_req_duration: ["p(95)<300", "p(99)<800"],
    http_req_failed: ["rate<0.01"],
  },
};

export default function () {
  const res = http.get(`${__ENV.BASE_URL}/api/checkout/quote`);
  check(res, { "status 200": (r) => r.status === 200 });
}
```

### Anti-patterns

| Anti-pattern | Why it fails | Fix |
|---|---|---|
| Load test from a laptop over Wi-Fi | Measures the network, not the service | Run from a fixed host in the same region |
| Asserting exact milliseconds in PR CI | Flaky on shared runners | Nightly with thresholds, trends over time |
| Only the happy path | Peak traffic hits retries, cache misses, cold paths | Include the failure-heavy journey |
| Tuning before measuring | Optimizes the wrong thing | Profile under the realistic scenario first |

## Security testing

### Threat model first

List assets (money, personal data, credentials), actors (anonymous, user, admin, other tenant), and trust boundaries (internet → API, API → DB, job → API). For each boundary, ask what a malicious caller can send. A test suite without a threat model tests what is easy, not what is risky.

### Authorization is testable; treat it as a matrix

- **Role × resource × action**: for every role, assert allowed and denied on the sensitive resources. Denied cases are the valuable ones.
- **Tenant isolation, both directions**: tenant A cannot read B's row by ID, by list query, by export, or by report. Test the ID-guessing path explicitly.
- **"Works as Administrator" is not a pass.** Run the matrix as the least-privileged role; elevate only where the rule requires it.
- **Object-level checks, not only endpoint-level**: the endpoint may be allowed while the object is not.

```python
import pytest

@pytest.mark.parametrize(
    "role,expected",
    [("member", 200), ("viewer", 403), ("other_tenant", 404)],
)
def test_invoice_read_respects_role_and_tenant(client, invoices, role, expected):
    invoice = invoices["tenant_a"]
    client.authenticate(role, tenant=invoices[role].tenant)
    assert client.get(f"/invoices/{invoice.id}").status_code == expected
```

### Authentication and input handling

- Session expiry, refresh-token rotation, token audience/issuer, password-reset single use.
- Boundary values, encoding, and injection surfaces: query, template, header, filename, redirect target.
- Error responses must not leak stack traces, SQL, or internal IDs.
- Rate limits and lockouts: assert they trigger, and that they do not lock out legitimate recovery.

### Scans complement tests, they do not replace them

| Layer | Tool examples | Where |
|---|---|---|
| Secrets | `gitleaks`, `detect-secrets` | Pre-commit + CI |
| Dependencies | `pip-audit`, `npm audit`, `govulncheck`, `cargo audit` | CI |
| SAST | `semgrep`, `bandit`, `gosec`, `eslint-plugin-security` | CI |
| DAST (baseline) | OWASP ZAP baseline scan | Nightly against staging |
| Container/IaC | `trivy`, `checkov` | CI |

A scanner finding is an input to triage, not a coverage metric. The tests that protect you are the ones asserting *behavior*: who can do what, with which object, in which tenant.

### Anti-patterns

| Anti-pattern | Why it fails | Fix |
|---|---|---|
| Testing only as Administrator | Hides the most common vulnerability class | Run the role × object matrix |
| Tenant isolation tested in one direction | B may read A but not the reverse | Assert both directions |
| Treating a clean ZAP scan as "secure" | DAST finds a slice of issues | Threat model + authz tests + dependency scanning |
| Security tests only before release | Fixes arrive too late | Permission tests in every PR |

## Contract testing

Use it when **producer and consumer deploy independently** or share a schema across teams. Skip it when a single deployable owns both sides — an integration test is cheaper.

| Approach | Answers | Tools |
|---|---|---|
| **Consumer-driven** | Does the producer still satisfy what this consumer reads? | Pact, Spring Cloud Contract |
| **Schema validation** | Do requests/responses match the published schema? | Schemathesis, Dredd, `openapi-core` |
| **Diff** | Is this change breaking for existing clients? | `openapi-diff`, `oasdiff`, Buf (protobuf) |

Rules:

1. **Test both directions**: the new producer against old consumers, and the new consumer against the published contract.
2. **Breaking changes need a major version**: additive fields yes; removing, renaming, or changing types no.
3. **Do not encode business rules in contracts** — that is an integration test. Contracts describe shape and compatibility.
4. **Version the contract with the code** and publish it from CI, so consumers test against the real artifact.

```bash
# Schema conformance against a running service
schemathesis run openapi.json --base-url http://localhost:8000 --checks all
```

## Accessibility (minimum)

The five checks that catch most regressions, per critical journey:

1. **Keyboard-only**: complete the flow with Tab/Shift+Tab/Enter/Escape; no traps, no invisible focus.
2. **Names**: every input, button, and icon has an accessible name; labels are programmatically associated.
3. **Contrast**: 4.5:1 for text, 3:1 for large text and UI boundaries.
4. **Zoom/reflow**: 200% zoom and 320 px width without horizontal scrolling or overlap.
5. **Errors**: announced, associated with the field, and not conveyed by color alone.

Tooling (`axe-core`, `pa11y`, Lighthouse) catches roughly a third of issues; the keyboard pass is not optional. A dedicated browser/usability playbook belongs in the web pack; until it exists, keep these five in the critical-journey checklist.

## Reliability and recovery

- **Timeouts and retries**: assert bounded retries, backoff, and that a retry does not duplicate a side effect (idempotency key on the double-submit).
- **Circuit breaking**: kill the dependency in staging; the service degrades with a clear error instead of hanging.
- **Restore drill**: restore a backup into a clean environment and measure the time. An untested backup is a hope, not a control plane.
- **One drill per quarter per critical service** is enough; more is theatre.

## Cadence

| Moment | Non-functional checks |
|---|---|
| Pull request | Authz matrix (fast), contract validation, a11y lint |
| Nightly | Load/stress with thresholds, DAST baseline, dependency scans |
| Release | Soak, restore drill, accessibility keyboard pass |
| Quarterly | Stress-to-failure, chaos-lite drill, review of budgets |

## Reporting

One line per NFR, in the audit pass or a dedicated `docs/quality/nfr-YYYY-MM-DD.md`:

```markdown
| NFR | Budget | Measured | Verdict | Artifact |
|-----|--------|----------|---------|----------|
| Checkout p95 | < 300 ms @ 200 rps | 268 ms | PASS | `artifacts/k6-checkout.json` |
| Tenant isolation | 0 cross-tenant reads | 0 (48 cases) | PASS | `reports/authz-2026-11-11.xml` |
| API compatibility | v1 client on v2 | 1 breaking diff | FAIL | `reports/openapi-diff.txt` |
```

A verdict without an artifact is an opinion.

## Definition of Done (non-functional)

- [ ] Every critical path has at least one budget with a number or a binary assertion.
- [ ] The load scenario models the real journey, not a single endpoint.
- [ ] Thresholds live in the script and fail a nightly run.
- [ ] The authz matrix covers roles, objects, and tenants in both directions.
- [ ] Scans are wired to CI and findings have an owner and a date.
- [ ] Consumers and producers are tested against the published contract.
- [ ] Keyboard-only completion and 200% zoom pass for the critical journey.
- [ ] Restore was executed at least once in the last quarter.

## Anti-patterns

| Anti-pattern | Why it fails | Fix |
|---|---|---|
| "We'll load-test before launch" | One data point, no baseline, no regression detection | Nightly from the first release |
| Budgets with no owner | Numbers nobody defends | Name an owner per budget |
| Security as a pre-release gate only | Late, expensive, incomplete | Authz tests per PR, scans continuously |
| Contract tests that mirror the producer | Always green, never protects consumers | Test from the consumer's expectations |
| Accessibility done by an external audit only | Regresses the next sprint | Keyboard pass in the critical journey |
| Non-functional tests in the functional suite | Slow, flaky, ignored | Separate cadence, separate environment |

## References

- [`test-strategy.md`](test-strategy.md) — functional layers, coverage policy, CI gates
- [`mutation-testing.md`](mutation-testing.md) — whether the functional suite actually catches changes
- [`architecture-tests.md`](architecture-tests.md) — executable structural rules (also a non-functional control)
- `code-audit` — records NFR gaps as findings with importance
- `frappe-testing` — Frappe-specific permissions and CI
