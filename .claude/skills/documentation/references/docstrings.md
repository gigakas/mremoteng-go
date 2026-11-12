# Docstrings and Comments

Comments explain **why**; docstrings document the **public contract**. Neither should restate what the code already says.

## Comments vs docstrings

| | Comment | Docstring |
|---|---|---|
| **Purpose** | Explain a non-obvious decision | Document the public API |
| **Audience** | Future maintainer | Caller / generated docs |
| **Location** | Next to the tricky line | Attached to the function/class/module |
| **Content** | Why, trade-off, workaround, constraint | Params, return, errors, side effects, example |

## When to comment

- A non-obvious **why**: workaround, business rule, historical constraint.
- A **trade-off**: why this approach instead of the obvious one.
- A **surprising side effect** the signature does not show.
- A **TODO** with an owner and a reason.

Do not comment:

- what the next line obviously does;
- commented-out code (delete it; Git remembers);
- a changelog (that is `CHANGELOG.md`);
- a substitute for a better name.

```python
# Bad: restates the code
i += 1  # increment i

# Good: explains a constraint
# The upstream API rejects page sizes above 100; keep this below the SDK default.
page_size = min(requested, 100)
```

## Docstring rules

- Document the **contract**, not the implementation.
- Include params, return, raised errors, and side effects.
- Add one runnable example for non-trivial public functions.
- State units, formats, and nullability.
- Keep it in sync with the signature; a stale docstring is a bug.
- Do not duplicate generated API docs; link to them.

## Per-language conventions

### Python

Use PEP 257 + a consistent style (Google, NumPy, or reST). Google style:

```python
def charge(amount: Money, idempotency_key: str) -> Receipt:
    """Charge a customer once per idempotency key.

    Args:
        amount: Amount to charge, in the account currency.
        idempotency_key: Client-generated key; repeated calls return the
            original receipt.

    Returns:
        The receipt for the charge.

    Raises:
        InsufficientFunds: If the account cannot cover the amount.
    """
```

### TypeScript / JavaScript

Use TSDoc/JSDoc for public APIs:

```ts
/**
 * Charge a customer once per idempotency key.
 *
 * @param amount - Amount in minor units (cents).
 * @param idempotencyKey - Client-generated key; repeated calls return the original receipt.
 * @returns The receipt for the charge.
 * @throws {InsufficientFundsError} If the account cannot cover the amount.
 */
export function charge(amount: number, idempotencyKey: string): Receipt
```

### Java

Javadoc:

```java
/**
 * Charges a customer once per idempotency key.
 *
 * @param amount amount in minor units
 * @param idempotencyKey client-generated key
 * @return the receipt
 * @throws InsufficientFundsException if the account cannot cover the amount
 */
public Receipt charge(long amount, String idempotencyKey)
```

### C#

XML comments:

```csharp
/// <summary>Charges a customer once per idempotency key.</summary>
/// <param name="amount">Amount in minor units.</param>
/// <param name="idempotencyKey">Client-generated key.</param>
/// <returns>The receipt.</returns>
/// <exception cref="InsufficientFundsException">Account cannot cover the amount.</exception>
public Receipt Charge(long amount, string idempotencyKey)
```

### Go

Doc comments start with the identifier:

```go
// Charge charges a customer once per idempotency key.
// Repeated calls with the same key return the original receipt.
func Charge(amount int64, idempotencyKey string) (Receipt, error)
```

### Rust

Rustdoc:

```rust
/// Charges a customer once per idempotency key.
///
/// # Errors
/// Returns [`Error::InsufficientFunds`] if the account cannot cover the amount.
pub fn charge(amount: i64, idempotency_key: &str) -> Result<Receipt, Error>
```

### Ruby

YARD conventions with `#` comments directly above the method:

```ruby
# Applies the member discount to an order total.
#
# @param amount [Integer] order total in cents
# @param member [Boolean] whether the customer is a member
# @return [Integer] discounted total in cents
# @raise [ArgumentError] when amount is negative
# @example
#   discount(10_000, member: true) # => 9_000
def discount(amount, member:)
  raise ArgumentError, "amount must be >= 0" if amount.negative?
  member ? amount * 90 / 100 : amount
end
```

- Tags: `@param [Type]`, `@return [Type]`, `@raise`, `@example`, `@see`, `@deprecated`.
- Generate with `yard doc`; keep the tag types consistent with the signature.
- RuboCop's `Style/Documentation` can enforce the class/module summary, not the tags.

### PHP

PHPDoc blocks above classes, methods and properties:

```php
/**
 * Apply the member discount to an order total.
 *
 * @param int $amount Order total in cents.
 * @param bool $member Whether the customer is a member.
 * @return int Discounted total in cents.
 * @throws \InvalidArgumentException When the amount is negative.
 */
function discount(int $amount, bool $member): int
{
    if ($amount < 0) {
        throw new \InvalidArgumentException('amount must be >= 0');
    }
    return $member ? intdiv($amount * 90, 100) : $amount;
}
```

- Tags: `@param`, `@return`, `@throws`, `@var`, `@deprecated`; Psalm/PHPStan add `@psalm-*`/`@phpstan-*` for generics.
- Generate with phpDocumentor; static analyzers read the same tags, so a wrong type is a bug.
- Enforce block presence with `phpcs`/`php-cs-fixer` (e.g., `Squiz.Commenting.FunctionComment`).

### Dart

Dartdoc uses `///` above the declaration; the first paragraph is the summary:

```dart
/// Applies the member discount to an order amount in cents.
///
/// Returns the discounted total in cents. Throws an [ArgumentError] when
/// [amount] is negative.
///
/// ```dart
/// discount(10000, member: true); // 9000
/// ```
int discount(int amount, {required bool member}) {
  if (amount < 0) throw ArgumentError.value(amount, 'amount', 'must be >= 0');
  return member ? amount * 90 ~/ 100 : amount;
}
```

- Reference other declarations with `[Name]`; use `{@template}`/`{@macro}` for repeated blocks.
- Generate with `dart doc`; the `public_member_api_docs` lint enforces presence on public APIs.
- There are no `@param` tags: document parameters in prose or as `[param]` references.

## TODO format

```text
TODO(owner, 2026-03-01): remove the fallback after the provider migration.
```

A TODO without an owner and a date is permanent debt.

## Checklist

- [ ] Public functions/classes have docstrings.
- [ ] Params, return, errors, side effects documented.
- [ ] Units/formats/nullability stated.
- [ ] Example for non-trivial public APIs.
- [ ] Comments explain why, not what.
- [ ] No commented-out code.
- [ ] TODOs have owner + date.
- [ ] Docstrings updated with the signature.

## Anti-patterns

| Anti-pattern | Problem | Fix |
|---|---|---|
| Comment restating code | Noise | Remove or explain why |
| Docstring without errors | Callers cannot handle failure | Document raised errors |
| Stale docstring | Misleading | Update with the signature |
| Commented-out code | Confusing | Delete; use Git |
| `@param` mismatch | Broken generated docs | Fix or generate from types |
| TODO with no owner | Permanent debt | Add owner + date |
