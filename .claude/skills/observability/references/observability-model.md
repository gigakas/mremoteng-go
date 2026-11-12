# Observability Model

Logs, metrics and traces are not interchangeable. Choosing the wrong signal is why teams have millions of log lines and still cannot answer "is it slow?".

## Which signal answers which question

| Question | Signal | Why |
|---|---|---|
| What happened in this request? | Logs | Full context, per event |
| How many failed? How slow? | Metrics | Cheap aggregation, alerting |
| Where did the time go? | Traces | Per-hop timing and causality |
| Is it up? | Health check + metric | Fast, external |
| Did the deploy change behavior? | All three, grouped by `version` | Compare windows |

Rule of thumb:

- A number you will graph or alert on → **metric**.
- A discrete fact you will search later → **log**.
- A request crossing components → **trace**.
- A question with no bounded labels → logs, not metric labels.

## Correlation

One context threads all three:

```text
X-Request-ID: req_01J8Z3K4          (generated at the edge, returned to the client)
traceparent: 00-4bf92f…-00f067…-01 (W3C trace context, propagated across services)
```

- The edge generates `request_id` and the root span; both travel in request headers.
- Every log event inside the request includes `request_id`, `trace_id`, `span_id`.
- Queue messages carry the context in the envelope; the consumer rebinds it.
- Cron jobs create a root context of their own (`job_id` plus a trace).
- The client-facing error response includes `request_id` so support can find the event.

## Metrics

| Type | Use for | Example |
|---|---|---|
| Counter | Monotonic totals | `http_requests_total{status="500"}` |
| Histogram | Latency and size distributions | `http_request_duration_seconds_bucket` |
| Gauge | Current value | `queue_depth`, `connections_open` |

Rules:

- Percentiles come from histograms, never from averaging averages.
- Labels must be bounded: `route` template (`/orders/{id}`), not the raw URL; `status` class, not every code; never `user_id`.
- Name with unit suffix (`_seconds`, `_bytes`) and follow one convention across services.
- Cardinality has a bill: estimate series = product of label value counts before adding a label.

## Traces

- One trace per request or job; spans for I/O boundaries (DB, HTTP, queue), not for every function.
- Span attributes are the same identifiers as log `attributes` (`order_id`, `provider`).
- Mark errors on the span (`status = error`) and record the exception once.
- Sample by percentage, but keep errors: tail sampling if the backend supports it, otherwise sample at the edge and always trace failures.
- Propagate `traceparent` through every client; a missing header cuts the trace in half.

## Health checks

| Probe | Question | Behavior |
|---|---|---|
| **Liveness** | Is the process alive? | No dependency checks; restart on failure |
| **Readiness** | Can it serve traffic now? | Check critical dependencies with a short timeout |
| **Startup** | Has initialization finished? | For slow boot; avoids liveness killing a warming process |

Rules:

- Readiness must not depend on non-critical services (email, analytics); otherwise one outage takes down a healthy fleet.
- Keep probes internal and unauthenticated; expose only inside the network.
- Return a degraded state explicitly (`{"status":"degraded","cache":"down"}`) instead of a binary lie.
- A health check that always returns 200 is worse than none.

## SLOs and alerts

- **SLI**: a ratio of good events to total (`successful_requests / valid_requests`), measured from metrics.
- **SLO**: a target over a window (`99.9% over 28 days`) with an error budget.
- **Alert on symptoms, not causes**: page when users are affected (error rate, latency), not when CPU is high.
- **Burn rate**: alert when the budget is consumed faster than the window allows; two windows (fast + slow) avoids both noise and late detection.
- One alert per user-visible symptom, with a link to the runbook (`documentation`).

## Dashboards

Minimum per service:

1. Traffic: requests per second by route and status class.
2. Errors: error rate and the top events (from logs, grouped by `event`).
3. Latency: p50/p95/p99 from histograms.
4. Saturation: CPU, memory, queue depth, connection pool.
5. Deploy marker: version changes on the same timeline.

A dashboard without a deploy marker cannot explain a step change.

## Retention and cost

| Data | Retention |
|---|---|
| `debug` (when enabled) | Hours to a few days |
| `info` business events | 7–30 days |
| `warn`/`error` | 30–90 days |
| Traces (sampled) | 7–30 days |
| Metrics | 13 months (downsampled) |

- Log volume is driven by fields, not by lines: a 4 KB payload per event costs more than 10× the events.
- Drop or sample before ingestion when the platform supports it; filtering later still pays for storage.
- Review the top 10 events by volume quarterly; most cost comes from a handful of paths.

## Anti-patterns

| Anti-pattern | Why it fails | Fix |
|---|---|---|
| Parsing logs to build metrics | Fragile and expensive | Emit the metric in code |
| `user_id` as a metric label | Cardinality explosion | Aggregate, keep the ID in logs |
| Alerting on CPU | Symptom-free noise | Alert on error rate and latency |
| Readiness checks that call 10 dependencies | One slow dep removes the fleet | Check critical deps only, with timeouts |
| Traces sampled at 1% with no error retention | The incident is the missing trace | Tail sampling or always trace failures |
| Logs and metrics with different service names | Nothing joins | One `service` identity everywhere |
| Dashboards without version markers | Cannot correlate deploys | Add the deploy annotation |
