# Commit Message Examples

Good and bad messages for the situations that actually occur. The subjects are imperative, lowercase and describe the effect.

## Features

```text
feat(checkout): accept guest orders
feat(api): paginate the invoices endpoint
feat(settings): remember the last selected workspace
```

Bad:

```text
feat: added guest checkout          # past tense, vague
feat: checkout                      # no effect described
feature(checkout): guest orders     # type not in the list
```

## Fixes

```text
fix(invoices): reject duplicate invoice numbers
fix(auth): keep the session alive after token refresh
fix(export): include the header row in empty exports
```

Body when the cause is not obvious:

```text
fix(auth): keep the session alive after token refresh

The refresh call replaced the session cookie before the response body
was read, so the next request arrived without credentials. Preserve the
old cookie until the response completes.

Closes: #412
```

## Performance

```text
perf(reports): load invoice lines in one query

Replace the per-row get_doc with a single filtered fetch. p95 on the
monthly report drops from 4.2 s to 380 ms on production-sized data.
```

## Refactors (no behavior change)

```text
refactor(pricing): extract discount rules into their own module
refactor(api)!: rename `order_id` to `invoice_id` in responses
```

The second one is only `refactor` if it is not user-visible; renaming a public field is `feat` or a breaking change. Choose by effect on consumers, not by effort.

## Tests, docs, build, CI

```text
test(checkout): cover the expired-card path
docs(readme): document the local test command
build(deps): bump playwright to 1.49.0
ci: cache node_modules between jobs
style: format with prettier
chore: remove the unused seed script
```

## Breaking changes

```text
feat(api)!: require an idempotency key on POST /orders

BREAKING CHANGE: requests without `idempotency_key` now return 422.
Clients must send a unique key per logical order and retry with the
same key on network failures.
```

The `!` and the footer are both required; the footer is what tools read.

## Reverts

```text
revert: feat(api): require an idempotency key on POST /orders

This reverts commit 3f2a1c9. The key validation rejected legitimate
retries from the mobile client, which sends a new UUID per attempt.
```

`git revert` generates a `Revert "..."` header; either keep it (the checker allows it) or rewrite it to the `revert:` form and reference the original commit.

## Agent attribution (shared repositories)

```text
fix(checkout): clear the cart after a failed payment

Agent: OC
Co-authored-by: Ana <ana@example.com>
```

- `Agent: OC` (opencode) or `Agent: AC` (Claude Code).
- One trailer per commit; a human co-author is `Co-authored-by`.
- The changelog fragment uses the same identifier in its folder (`changelog.d/OC/`).

## Squash and merge titles

When the PR is squash-merged, the PR title becomes the commit subject and the description becomes the body. Keep the title in the same format:

```text
feat(reports): export invoices as CSV
```

Not:

```text
[REPORTS] CSV export stuff (#812)
```

## Merge and fixup commits

These are exempt from the format and pass the checker untouched:

```text
Merge branch 'main' into feature/invoice-export
fixup! feat(reports): export invoices as CSV
squash! feat(reports): export invoices as CSV
```

## Multi-line body rules

- Blank line between subject and body, and between body and footers.
- Wrap at 72 characters.
- Bullets are fine; lines starting with `#` are comments in `git commit` and are stripped — use `-` instead.
- Do not restate the diff ("changed line 42"). Explain intent and consequences.

## Messages that should not exist

| Message | Problem |
|---|---|
| `wip` | Squash before opening the PR |
| `asdf`, `.` | No message |
| `fix tests` | Which tests, which failure? |
| `update code` | Which code, which behavior? |
| `PR comments` | Describe the change, not the review |
| `feat: JIRA-1234` | Ticket is not a description; use `Refs:` |
