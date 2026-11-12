---
name: observability
description: Use when adding or reviewing logging, defining what a log event contains, correlating logs with traces, or deciding between a log, a metric and a trace. Covers structured JSON events, levels, PII redaction, request and trace IDs, sampling, health checks and alerting.
---

# Observability

You cannot debug what you cannot query. Observability is the ability to answer **new** questions about a running system without shipping code: which requests failed, for whom, how long they took, and what changed.

The three signals answer different questions, and most teams over-invest in one:

| Signal | Answers | Cost shape |
|---|---|---|
| **Logs** | What exactly happened in this event, with context | Per event |
| **Metrics** | How often / how much, aggregated | Per series (cardinality) |
| **Traces** | Where the time went across services | Per span (sampled) |

Structured logging is the cheapest entry point: it makes the first question answerable and gives traces a context to attach to.

## When to use

- Replacing `print`, `console.log` or free-text logs with a structured event.
- Deciding what an event must contain.
- Correlating logs across services, requests or background jobs.
- Reviewing a service that logs too much, too little, or the wrong thing.
- Choosing between a log line, a counter/histogram, or a span.
- Adding health checks or the first alert.

## When NOT to use

- Writing a runbook or on-call procedure — `documentation`.
- Performance budgets and load tests — `testing-patterns` (`non-functional-testing.md`).
- Security incident response — follow the org's incident process; this skill only defines what to record.
- Vendor setup (SaaS log platform configuration) — follow the platform docs; the event schema here is portable.
- Auditing code quality — `code-audit` (it can route findings here).

## The event

One log line = one JSON object = one event. Never a formatted string with the fields glued in.

```json
{
  "timestamp": "2026-11-11T14:03:21.482Z",
  "level": "error",
  "service": "checkout-api",
  "env": "prod",
  "version": "2.14.3",
  "event": "payment.authorization_failed",
  "message": "Payment provider rejected the authorization",
  "request_id": "req_01J8Z3K4",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "duration_ms": 812.4,
  "error": { "type": "ProviderError", "message": "insufficient_funds" },
  "attributes": { "provider": "stripe", "order_id": "ord_1042", "retry": 1 }
}
```

| Field | Required | Rule |
|---|---|---|
| `timestamp` | Yes | ISO 8601 UTC with milliseconds |
| `level` | Yes | `debug`, `info`, `warn`, `error`, `fatal` (lowercase) |
| `service` | Yes | Stable service name, same as in traces and dashboards |
| `event` | Yes | Dot-separated lowercase: `order.created`, `payment.authorization_failed` |
| `message` | Yes | Human sentence, no newlines, ≤ 200 chars; not a substitute for `event` |
| `env` | Recommended | `dev`, `staging`, `prod` |
| `version` | Recommended | Deployed build/version |
| `trace_id`, `span_id` | Recommended | From the active span; 32/16 lowercase hex |
| `request_id` | Recommended | Generated at the edge, returned as `X-Request-ID` |
| `user_id` | Optional | Pseudonymous ID, never email or name |
| `duration_ms` | Optional | For operations worth timing |
| `error.type`, `error.message` | Required with `error` level | Type is a class/code, message is short |
| `error.stack` | Optional | Structured stack, not embedded in `message` |
| `attributes` | Optional | Flat key/value bag for domain data |

**Forbidden** anywhere in the event: passwords, tokens, API keys, `Authorization`/`Cookie` headers, full card numbers, CVV, national IDs, health data, and free-form payloads copied from requests.

Schema and validator: [`assets/log-event.schema.json`](assets/log-event.schema.json) and `<install>/scripts/check-log-event.py`.

## Levels

| Level | Use for | Example |
|---|---|---|
| `debug` | Developer detail, off in production by default | cache hit/miss |
| `info` | Business or lifecycle events worth keeping | `order.created`, `job.completed` |
| `warn` | Degraded but handled | retry after timeout, fallback used |
| `error` | The operation failed; someone should look | payment rejected, unhandled exception |
| `fatal` | The process cannot continue | config invalid at startup |

Rules:

- `error` means **a request or job failed**, not "something odd happened".
- One failure, one log. Do not log in every layer of the call stack.
- Never log expected control flow (validation errors the UI handles) as `error`; use `info` or `warn`.
- `debug` must be off in production unless a flag turns it on for a bounded time.

## Procedure

1. **Name the event** as `object.action` in past tense: `invoice.paid`, not `paying invoice`.
2. **Choose the level** by consequence, not by how the developer felt.
3. **Pick the fields** from the schema: nothing else unless it is domain data you will query.
4. **Add correlation**: `request_id` at the edge, `trace_id`/`span_id` from the active context.
5. **Redact at the boundary**: an allowlist of fields, not a denylist of secrets.
6. **Log once**, at the boundary where the outcome is known, with the error attached.
7. **Validate** the emitted line in a test or CI (see below).
8. **Ship with the query**: a log nobody can search is storage cost.

## Per-language starting points

| Language | Library | Notes |
|---|---|---|
| Python | `structlog`, or `logging` + `python-json-logger` | Bind request context with `contextvars`; `structlog.contextvars.bind_contextvars` |
| Node/TS | `pino` | `pino-http` for the access log; redact paths configured, not ad hoc |
| Dart/Flutter | `logging`, `logger` | `dart:developer log()` shows in DevTools; configure level and printer once |
| Go | `log/slog` (stdlib) | `slog.NewJSONHandler`; `context` carries the logger |
| Java/Kotlin | SLF4J + Logback/Log4j2 JSON encoder | MDC for `request_id`/`trace_id` |
| C# | Serilog (`Serilog.Formatting.Compact`) | Enrichers for service/env/version |
| Rust | `tracing` + `tracing-subscriber` JSON | Spans give the trace context |
| Ruby | `semantic_logger` | Tagged context instead of string interpolation |
| PHP | Monolog with `JsonFormatter` | Processors for request context |

Framework note: OpenTelemetry's log data model (`timestamp`, `severity_text`, `body`, `attributes`, `trace_id`, `span_id`) is a superset of this schema — mapping the fields above to OTel attributes keeps you portable.

## Enforcing it

There is no JSONL stream to validate in this repository: the activity console
records plain human-readable lines, so a schema checker would have nothing to
read. What replaces it:

- `internal/ui/log.go` bounds the console (entry count and age) so a long session
  cannot grow without limit.
- `internal/ui/diagnostics.go` is the support surface: extend it rather than
  adding a second dump path.
- A test that exercises a failure path asserts the status message the user sees,
  which is the only contract the console promises.

Wire it in CI over a fixture file, and assert in code review that every new `error` path emits exactly one event with a correlation ID.

## Guardrails

- **Redact by allowlist.** A denylist always misses the next secret field.
- **No PII in attributes.** `user_id` is a pseudonym; email, phone, name and address are not logs.
- **Never interpolate untrusted input** into `message` unescaped; log injection is real (newlines and control characters).
- **Do not log full payloads** "just in case". Log identifiers and the decision, not the body.
- **Do not build metrics by parsing logs** when the process can emit a counter or histogram.
- **Keep cardinality bounded**: no unbounded values (`user_id`, `order_id`, raw URLs) as metric labels.
- **One place for configuration**: level, redaction and encoding set once, not per module.
- **Errors at the boundary**: catch, log with context, and return a stable error to the caller.

## Done when

- [ ] Every service emits JSON events with the required fields.
- [ ] `event` names are `object.action` and reused consistently.
- [ ] `request_id` and trace context are present in request-scoped logs.
- [ ] A redaction allowlist exists and is tested with a sample event.
- [ ] Exactly one `error` event per failed operation, with `error.type`/`error.message`.
- [ ] Levels are meaningful; production runs at `info` with `debug` off.
- [ ] The checker passes on a fixture from the test suite.
- [ ] At least one documented query answers "show me the failures for this request_id".

## Common mistakes

| Mistake | Why it fails | Fix |
|---|---|---|
| `print`/`console.log` in production paths | No fields, no levels, no correlation | Structured event with the schema |
| Logging the same error in 3 layers | Noise, duplicate alerts | Log once at the boundary |
| `error` for expected validation | Alerts fire on user typos | `info`/`warn` for handled cases |
| Message contains the whole payload | PII leak and huge cost | Identifiers plus decision |
| `trace_id` only in HTTP logs | Background jobs and queues lose correlation | Propagate the context everywhere |
| One metric label per user | Cardinality explosion, huge bill | Bounded labels (`status`, `route`) |
| Logs without retention policy | Cost grows forever, debugging stale data | Tiered retention by level |

## References

- [`references/structured-logging.md`](references/structured-logging.md) — field-by-field schema, redaction, sampling, migration and query patterns
- [`references/observability-model.md`](references/observability-model.md) — logs vs metrics vs traces, correlation, health checks, SLOs and alerts
- `assets/log-event.schema.json` — machine-readable event schema
- `check-log-event.py` — CI/hook validator for JSONL events

## Repo adjustments (mremoteng-go)

This is a desktop application, not a service, so most of the server-side advice
above does not apply. What the repository actually has:

- **Activity console**: `internal/ui/log.go` keeps a bounded, collapsible log the
  user can read in the tab; `internal/ui/sessionstate.go` and the session panels
  append to it through the screen.
- **Status bar**: `setStatus` reports normal progress and `setProblem` reports a
  failure with danger styling. Both also reach the activity log, so a session
  error is visible twice on purpose: transiently and historically.
- **Diagnostics dialog**: `internal/ui/diagnostics.go` gathers what a support
  request needs. Extend it instead of adding a second diagnostics path.
- **No structured JSON logging, no metrics endpoint, no tracing.** Do not
  introduce a logging library or a metrics stack for a local GUI app; that is a
  new dependency with no consumer.
- **Never log credentials.** Passwords reach FreeRDP through its stdin and the
  vault only on explicit unlock; nothing else may appear in the log, the status
  bar or a diagnostics dump.
- **Messages are user-facing**: English, actionable, and free of internal
  identifiers the user cannot act on. A message that only says "error" is a bug.
- **Changelog fragments are the durable record**, not the activity log: the console
  is cleared on restart, `changelog.d/` is not.

