# API Documentation

API docs are a contract. Prefer **contract-first**: define the schema once, generate the reference, and keep hand-written examples minimal and tested.

## Contract-first workflow

1. Define the contract: OpenAPI 3.x (HTTP), JSON Schema (events/config), Protobuf (gRPC), GraphQL SDL.
2. Generate the reference and client types from the contract.
3. Add hand-written guides for auth, errors, pagination, versioning, and idempotency.
4. Validate the contract in CI; fail on breaking changes.
5. Publish the reference next to the versioned API.

Do not hand-maintain a reference that duplicates the schema. It will drift.

## Endpoint documentation template

```markdown
## POST /v1/orders

Create an order.

### Auth
Bearer token with scope `orders:write`.

### Request

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `customer_id` | string (uuid) | yes | Customer identifier |
| `items` | array<Item> | yes | At least one item |
| `coupon` | string | no | Coupon code |

```json
{
  "customer_id": "9f1c...",
  "items": [{"sku": "ABC", "quantity": 2}]
}
```

### Response `201`

```json
{
  "id": "ord_123",
  "status": "pending",
  "total": {"amount": "42.00", "currency": "EUR"}
}
```

### Errors

| Status | Code | When |
|--------|------|------|
| `400` | `invalid_request` | Missing/invalid field |
| `409` | `duplicate_order` | Idempotency key reused with different payload |
| `422` | `insufficient_stock` | Item unavailable |

### Idempotency

Send `Idempotency-Key: <uuid>`; repeated requests with the same key return the
original response for 24h.
```

## What every API reference needs

- **Authentication and authorization**: how to authenticate, scopes/roles, examples.
- **Base URL and versioning**: `/v1`, deprecation policy.
- **Request/response examples**: realistic, not `foo`/`bar`.
- **Error model**: status codes, machine-readable codes, retryability.
- **Pagination**: cursor vs offset, limits, next/prev.
- **Filtering/sorting**: parameter names and allowed values.
- **Rate limits**: limits, headers, backoff.
- **Idempotency**: keys and semantics for writes.
- **Webhooks/events**: payloads, retries, signatures.
- **Changelog**: breaking vs non-breaking changes.

## Per-language generators

| Language | Source of truth | Docs |
|----------|-----------------|------|
| Python | OpenAPI / type hints | FastAPI `/docs`, Sphinx, pdoc, mkdocstrings |
| TypeScript | OpenAPI / TSDoc | TypeDoc, OpenAPI-generated reference |
| Java | OpenAPI / Javadoc | Springdoc, Javadoc |
| C# | OpenAPI / XML comments | Swashbuckle, DocFX |
| Go | OpenAPI / doc comments | `go doc`, swaggo |
| Rust | Rustdoc / OpenAPI | `cargo doc`, utoipa |

## Docstrings in the API layer

- Document the **public contract**: params, return, errors, side effects.
- Include one runnable example for non-trivial functions.
- Link to the relevant guide/ADR.
- Do not restate types that the signature already expresses.
- Keep examples tested; an outdated example is a bug.

See `docstrings.md` for per-language conventions.

## Breaking changes

A change is breaking if it:

- removes/renames a field, endpoint, or enum value;
- changes a type or format;
- makes an optional field required;
- changes error codes/semantics;
- changes auth/scopes.

Document breaking changes in `CHANGELOG.md`, bump the version, and provide a migration note. Non-breaking additions can ship in a minor release.

## Checklist

- [ ] Contract is the single source of truth.
- [ ] Reference is generated or validated in CI.
- [ ] Auth, errors, pagination, rate limits documented.
- [ ] Examples are realistic and tested.
- [ ] Breaking changes have a migration note.
- [ ] Versioning/deprecation policy is explicit.
- [ ] Docs are linked from the README.
- [ ] No secrets or real tokens in examples.

## Anti-patterns

| Anti-pattern | Problem | Fix |
|---|---|---|
| Hand-written reference | Drifts from the API | Generate from OpenAPI/schema |
| Only happy-path examples | Consumers hit errors blindly | Document errors and retries |
| `foo`/`bar` payloads | Not actionable | Use realistic data |
| No auth docs | Consumers cannot start | Document auth first |
| No versioning | Breaking changes surprise clients | Version and deprecate explicitly |
| Untested examples | Examples break silently | Test snippets in CI |
