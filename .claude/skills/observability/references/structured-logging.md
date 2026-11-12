# Structured Logging

Field-level guidance for the event defined in the skill: what to put in each field, what to keep out, and how to migrate without a flag day.

## Why JSON and not a formatted string

```text
# Unqueryable
2026-11-11 14:03:21 ERROR Payment failed for order ord_1042 (user ana@example.com)
```

```json
{"timestamp":"2026-11-11T14:03:21.482Z","level":"error","service":"checkout-api","event":"payment.authorization_failed","message":"Payment provider rejected the authorization","request_id":"req_01J8Z3K4","attributes":{"order_id":"ord_1042","provider":"stripe"}}
```

The second line can answer "how many failures per provider in the last hour" without a regex, and without exposing the customer's email.

## Field rules

### `timestamp`

- ISO 8601, UTC, milliseconds: `2026-11-11T14:03:21.482Z`.
- Let the logger stamp it, not the business code; it must be the moment of emission.
- If the platform adds ingestion time, keep both and name them differently.

### `level`

| Level | Numeric (OTel) | Meaning |
|---|---|---|
| `debug` | 5 | Detail for a developer; off in production |
| `info` | 9 | Normal business or lifecycle event |
| `warn` | 13 | Degraded but handled; no action tonight |
| `error` | 17 | Operation failed; someone should look |
| `fatal` | 21 | Process cannot continue |

The numeric mapping matters when you send logs to an OTel backend; keep `level` as text in the event and let the exporter map it.

### `event`

`object.action`, lowercase, past tense, stable:

```text
order.created          order.cancelled        invoice.paid
payment.authorization_failed                  job.completed
user.login_failed      cache.invalidated      migration.applied
```

Rules:

- One event name per code path; do not build it dynamically from user input.
- Reuse names across services; a dashboard should not need a synonym table.
- Do not encode IDs or status in the name (`order_1042_created`); those are fields.

### `message`

A sentence for the human reading the log at 3 a.m. It complements `event`, it does not replace it.

- No newlines, tabs or control characters (log injection).
- Do not interpolate untrusted values; put them in `attributes`.
- Keep it stable when possible; the event name is what you query, the message is what you read.

### Correlation fields

| Field | Where it comes from |
|---|---|
| `trace_id`, `span_id` | Active OpenTelemetry span (`traceparent` header, W3C trace context) |
| `request_id` | Generated at the edge (proxy or first handler) and returned as `X-Request-ID` |
| `user_id` | Pseudonymous internal ID after authentication |

Background jobs and queue consumers must copy the context from the message envelope; otherwise the trace stops at the enqueue.

### `error`

```json
"error": {
  "type": "StripeCardError",
  "message": "insufficient_funds",
  "stack": "StripeCardError: insufficient_funds\n  at charge (payments.ts:88)"
}
```

- `type` is a class or stable code — it is what you group by.
- `message` is short and provider/user safe.
- `stack` only where it helps; it is expensive and often noisy in JSON logs.
- Do not duplicate the error into `message`; the viewer renders both.

### `attributes`

Flat scalars only:

```json
"attributes": { "order_id": "ord_1042", "provider": "stripe", "retry": 1, "amount_cents": 4990 }
```

- No nested objects, arrays of objects, or raw request bodies.
- Keys are snake_case and stable; they become queryable fields.
- Avoid high-cardinality values you will never filter on (full URLs with query strings).

## Redaction

**Allowlist, not denylist.** Build the event from known-safe fields:

```python
SAFE = ("order_id", "provider", "retry", "status")
attrs = {k: v for k, v in candidate.items() if k in SAFE}
```

Additional rules:

- Never log `Authorization`, `Cookie`, `Set-Cookie`, API keys, passwords, tokens.
- Never log full card numbers, CVV, national IDs, IBANs, health data.
- Email/phone/name are PII: if support needs to find a user, log `user_id` and look the user up elsewhere.
- Redact error messages coming from third parties: they sometimes echo the request.
- Do not log the same object at `debug` that you redacted at `info`; redaction happens before the level check.

The checker flags forbidden keys recursively, including inside `attributes`.

## Sampling and volume

| Situation | Approach |
|---|---|
| High-volume `info` (per request) | Sample a percentage, but keep 100% of `warn`/`error`/`fatal` |
| `debug` | Off in production; enable per-service with a TTL flag |
| Repeated identical events in a loop | Aggregate (count + first/last) instead of per item |
| Traces | Head sampling ~10%; tail sampling to keep all errors if the backend supports it |

Sampling logs loses the correlation with the unsampled trace: prefer sampling at the trace level and deriving logs from the same decision when possible.

## Migration without a flag day

1. Introduce the JSON logger alongside the old one, writing to a separate sink.
2. Convert the top 5 events that support existing dashboards or alerts.
3. Add the checker to CI over a recorded fixture, so new code cannot regress.
4. Convert remaining call sites; delete the old logger.
5. Remove free-text parsing rules from the log platform.

Do not convert by search-and-replace: the field names and levels need a decision per event.

## Testing logs

```python
def test_payment_failure_emits_one_event(caplog, client):
    with caplog.at_level("INFO"):
        client.post("/pay", json={"card": "declined"})
    events = [r for r in caplog.records if r.msg.get("event") == "payment.authorization_failed"]
    assert len(events) == 1
    assert events[0].msg["level"] == "error"
    assert "card" not in events[0].msg["attributes"]
```

Assert: one event per failure, required fields present, no forbidden keys, correlation ID propagated.

## Queries worth having before shipping

| Question | Query shape |
|---|---|
| Failures for this request | `request_id = "req_…"` |
| Error rate by event | `level = error` group by `event` over time |
| Slow operations | `duration_ms > 1000` group by `event` |
| Provider failures | `event = payment.authorization_failed` group by `attributes.provider` |
| Deploy correlation | `version = "2.14.3"` compared with the previous version |

If none of these can be answered, the schema is decorative.

## Stack examples

```python
# Python: structlog with context vars
import structlog
log = structlog.get_logger()
structlog.contextvars.bind_contextvars(request_id=rid, trace_id=tid)
log.info("order.created", order_id=oid, amount_cents=4990)
log.error("payment.authorization_failed", error={"type": "ProviderError", "message": "insufficient_funds"}, order_id=oid)
```

```ts
// Node: pino with redaction
const log = pino({ redact: { paths: ["req.headers.authorization", "*.password", "*.token"], censor: "[redacted]" } })
log.error({ event: "payment.authorization_failed", order_id, err }, "Payment provider rejected the authorization")
```

```dart
// Dart: logging with a JSON printer
final log = Logger('checkout');
Logger.root.level = Level.INFO;
Logger.root.onRecord.listen(jsonPrinter);
log.severe('Payment provider rejected the authorization',
  'event=payment.authorization_failed order_id=$oid');
// dart:developer log() for DevTools; ship JSON through your sink in production.
```

```go
// Go: slog with context
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "checkout-api")
logger.ErrorContext(ctx, "Payment provider rejected the authorization",
    "event", "payment.authorization_failed", "order_id", oid, "error.type", "ProviderError")
```
