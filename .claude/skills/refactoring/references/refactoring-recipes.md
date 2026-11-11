# Refactoring Recipes

Step-by-step playbooks for the situations that appear most often in application development. Each recipe is small, testable, and behavior-preserving.

Use with `../testing-patterns/references/test-strategy.md` and run the fast layer after every step:

```bash
./scripts/check.sh                 # formatting check → vet → all tests
go test ./internal/ui/ -count=1    # one package, fast loop
```

---

## 1. Long method → Extract Method

**Signal**: a method is longer than ~30–50 lines, needs comments to explain blocks, or has multiple levels of nesting.

**Goal**: each method says *what* it does; the reader can skip the *how*.

**Steps**:

1. Identify a contiguous block that has one purpose.
2. Name the new method after the purpose (`calculate_discount`, not `do_stuff`).
3. Copy the block; pass the variables it reads; return what it writes.
4. Replace the block with a call.
5. Run unit tests.
6. Repeat for the next block; do not extract everything at once.

```python
# Before
def process_order(order):
    # validate
    if not order.items:
        raise ValueError("empty order")
    if order.total <= 0:
        raise ValueError("invalid total")
    # calculate
    discount = 0
    if order.customer.is_premium:
        discount = order.total * 0.1
    total = order.total - discount
    # persist
    order.total = total
    order.save()
    order.notify_customer()

# After
def process_order(order):
    validate_order(order)
    order.total = apply_discount(order)
    order.save()
    order.notify_customer()

def validate_order(order):
    if not order.items:
        raise ValueError("empty order")
    if order.total <= 0:
        raise ValueError("invalid total")

def apply_discount(order):
    discount = order.total * 0.1 if order.customer.is_premium else 0
    return order.total - discount
```

**Verify**: the public behavior and the tests are unchanged. The diff should contain no new logic.

**Pitfalls**: extracting blocks that share too many locals (use Method Object instead); extracting one-liners that add no clarity.

---

## 2. God class → Extract Class

**Signal**: a class has many fields/methods, changes for unrelated reasons, or its name needs "and" to describe it.

**Goal**: one reason to change per class.

**Steps**:

1. List the fields and methods and group them by responsibility.
2. Pick the most cohesive group that can move without breaking the public API.
3. Create the new class; move fields and methods.
4. Keep a reference from the original class and delegate temporarily.
5. Update callers to use the new class when appropriate.
6. Remove the temporary delegation when no caller needs it.
7. Run tests after each move.

```python
# Before: Order does pricing, persistence and notifications
class Order:
    def total(self): ...
    def save(self): ...
    def send_confirmation(self): ...

# After
class Order:
    def __init__(self, pricing: "Pricing", repo: "OrderRepository", mailer: "Mailer"):
        self.pricing, self.repo, self.mailer = pricing, repo, mailer

    def total(self): return self.pricing.total(self)
    def save(self): self.repo.save(self)
    def send_confirmation(self): self.mailer.send(self)
```

**Verify**: callers still get the same behavior; the new classes have their own tests.

**Pitfalls**: splitting by line count instead of responsibility; creating an anemic data class and leaving all behavior behind.

---

## 3. Duplicate code → Extract Method / Pull Up Method

**Signal**: the same or near-identical block appears in two or more places, and a change would need edits in all of them.

**Goal**: one place to change; duplication only when it is incidental.

**Steps**:

1. Confirm the duplication is semantic (same reason to change), not accidental.
2. Extract the shared block into a function/method.
3. Replace each copy with a call.
4. If the copies are in sibling classes, pull the method up to the common parent.
5. Run tests.

```python
# Before: duplicated validation in two endpoints
def create_user(payload):
    if not payload.get("email"):
        raise ValueError("email required")
    if "@" not in payload["email"]:
        raise ValueError("invalid email")
    ...

def update_user(user_id, payload):
    if not payload.get("email"):
        raise ValueError("email required")
    if "@" not in payload["email"]:
        raise ValueError("invalid email")
    ...

# After
def validate_email(payload):
    email = payload.get("email")
    if not email:
        raise ValueError("email required")
    if "@" not in email:
        raise ValueError("invalid email")
    return email

def create_user(payload):
    email = validate_email(payload)
    ...

def update_user(user_id, payload):
    email = validate_email(payload)
    ...
```

**Verify**: both call sites still reject the same inputs and accept the same valid inputs.

**Pitfalls**: extracting two blocks that only look alike but change for different reasons; over-DRYing tests or configuration.

---

## 4. Switch on type → Replace Conditional with Polymorphism

**Signal**: the same `switch`/`if-elif` on a type/status appears in multiple methods, and each branch has behavior.

**Goal**: add a new type by adding a class, not by editing every switch.

**Steps**:

1. Confirm the branching is repeated; a single switch can be fine.
2. Create an interface/base class for the behavior.
3. Create one implementation per branch.
4. Replace the conditional with a call through the interface.
5. Move branch-specific data into the implementation.
6. Run tests; delete the old branches.

```python
# Before
def shipping_cost(order):
    if order.type == "standard":
        return 5
    if order.type == "express":
        return 15
    if order.type == "international":
        return 40
    raise ValueError(order.type)

# After
class Shipping(ABC):
    @abstractmethod
    def cost(self) -> int: ...

class Standard(Shipping):
    def cost(self): return 5

class Express(Shipping):
    def cost(self): return 15

class International(Shipping):
    def cost(self): return 40

def shipping_cost(order):
    return order.shipping.cost()
```

**Verify**: every previous branch returns the same value; unknown types are rejected at construction, not at call time.

**Pitfalls**: using polymorphism for a two-branch conditional; creating a class hierarchy that duplicates data instead of behavior.

---

## 5. Nested conditionals → Guard Clauses

**Signal**: the happy path is indented three or more levels; `else` branches are far from their `if`.

**Goal**: flat, readable control flow with early exits.

**Steps**:

1. Identify the conditions that raise/return early.
2. Invert each one into a guard at the top.
3. Keep the happy path unindented.
4. Run tests.

```python
# Before
def pay(order):
    if order is not None:
        if order.is_paid:
            return "already paid"
        else:
            if order.total > 0:
                return charge(order)
            else:
                raise ValueError("invalid total")
    else:
        raise ValueError("missing order")

# After
def pay(order):
    if order is None:
        raise ValueError("missing order")
    if order.is_paid:
        return "already paid"
    if order.total <= 0:
        raise ValueError("invalid total")
    return charge(order)
```

**Verify**: the same exceptions and return values for every input.

**Pitfalls**: turning validation into exceptions when the caller expects a result object; hiding a real state machine that deserves the State pattern.

---

## 6. Long parameter list → Introduce Parameter Object

**Signal**: more than 3–4 parameters, the same group travels together, or callers regularly swap arguments.

**Goal**: one concept, one object; fewer ordering mistakes.

**Steps**:

1. Identify the parameters that belong together.
2. Create an immutable value object with named fields.
3. Change the function to accept it.
4. Update callers.
5. Add validation to the object if needed.
6. Run tests.

```python
# Before
def search(start_date, end_date, min_amount, max_amount, status):
    ...

# After
@dataclass(frozen=True)
class SearchCriteria:
    start_date: date
    end_date: date
    min_amount: float
    max_amount: float
    status: str

    def __post_init__(self):
        if self.start_date > self.end_date:
            raise ValueError("start_date must be before end_date")

def search(criteria: SearchCriteria):
    ...
```

**Verify**: invalid combinations are rejected in one place; callers are more readable.

**Pitfalls**: creating a "parameter bag" with no invariants; making the object mutable and shared across calls.

---

## 7. Primitive obsession → Value Object

**Signal**: strings/ints represent domain concepts (money, email, phone, date range), and validation is duplicated.

**Goal**: type safety and behavior next to the data.

**Steps**:

1. Pick one primitive concept with rules.
2. Create an immutable value object.
3. Move validation and formatting into it.
4. Replace primitives at the boundaries.
5. Run tests.

```python
# Before
def withdraw(account_id: str, amount: float): ...

# After
@dataclass(frozen=True)
class Money:
    amount: Decimal
    currency: str = "EUR"

    def __post_init__(self):
        if self.amount < 0:
            raise ValueError("amount cannot be negative")

def withdraw(account_id: str, amount: Money): ...
```

**Verify**: the same invalid inputs are rejected; valid inputs behave identically.

**Pitfalls**: wrapping every string in a class; making the value object mutable; leaking the primitive representation.

---

## 8. Feature envy → Move Method

**Signal**: a method accesses another object's data more than its own; it looks like it belongs to the other class.

**Goal**: behavior lives next to the data it uses.

**Steps**:

1. Identify the target class (the one whose data is used most).
2. Copy the method to the target.
3. Replace references to the source with `self`/parameters.
4. Make the original delegate, or update callers.
5. Run tests; remove the old method.

```python
# Before
class Invoice:
    def total_with_tax(self, tax_rate):
        return sum(item.price for item in self.items) * (1 + tax_rate)

# After: the calculation belongs to the invoice, but the tax rule belongs to TaxPolicy
class TaxPolicy:
    def apply(self, amount):
        return amount * (1 + self.rate)

class Invoice:
    def total(self):
        return sum(item.price for item in self.items)
```

**Verify**: the calculation result is identical; the target class has a clear reason to own the method.

**Pitfalls**: moving a method that is genuinely a use case/service concern; creating a bidirectional dependency.

---

## 9. Message chain → Hide Delegate

**Signal**: clients call `a.b().c().d()`; a change in the middle object breaks many callers.

**Goal**: one method on the first object; clients do not know the chain.

**Steps**:

1. Add a method on the first object that returns the final value.
2. Replace the chain at call sites.
3. If the chain is used for different purposes, add separate methods.
4. Run tests.

```python
# Before
city = order.customer.address.city

# After
class Order:
    def shipping_city(self):
        return self.customer.address.city

city = order.shipping_city()
```

**Verify**: callers get the same value; the chain is no longer part of the public API.

**Pitfalls**: creating a Middle Man; hiding a delegate that clients legitimately need for other operations.

---

## 10. Refused bequest → Replace Inheritance with Delegation

**Signal**: a subclass ignores or no-ops most inherited methods, or inherits only for code reuse.

**Goal**: composition instead of forced inheritance.

**Steps**:

1. Create a field in the subclass holding the former parent.
2. Add methods that delegate only the behavior actually used.
3. Update callers to the new type.
4. Remove the inheritance.
5. Run tests.

```python
# Before
class Stack(list):        # inherits insert, remove, sort, ...
    pass

# After
class Stack:
    def __init__(self):
        self._items = []
    def push(self, item): self._items.append(item)
    def pop(self): return self._items.pop()
    def __len__(self): return len(self._items)
```

**Verify**: only the intended operations remain; callers cannot use inherited methods that violate the abstraction.

**Pitfalls**: replacing inheritance with a delegating class that exposes the whole interface (Middle Man); forgetting to update type checks.

---

## 11. Speculative generality → Collapse / Inline

**Signal**: abstract classes, hooks, parameters, or configuration that nothing uses; "for the future" code.

**Goal**: remove unneeded indirection; add it when the second variation appears.

**Steps**:

1. Search for usages of the abstraction.
2. If there is one implementation and no concrete plan, inline it.
3. Remove unused parameters/hooks.
4. Run tests.
5. Add the abstraction back only when a second real variation exists.

**Verify**: behavior unchanged; public API may shrink (a good thing).

**Pitfalls**: deleting an extension point that a plugin/consumer actually uses; confusing "unused in this repo" with "unused by consumers".

---

## 12. Legacy code with no tests → Characterization first

**Signal**: you must change code that has no tests and unclear behavior.

**Goal**: pin current behavior before touching structure.

**Steps**:

1. Pick representative inputs, including edge cases and known bugs.
2. Run the code and record actual outputs.
3. Write characterization tests that assert those outputs (bugs included).
4. Mark them clearly as characterization tests.
5. Refactor in small steps, keeping the tests green.
6. If a test fails, you changed behavior; decide whether that was intended.

**Verify**: the characterization suite passes before and after the refactor, except for intentional behavior changes (which get their own test update).

**Pitfalls**: "fixing" behavior while characterizing; writing tests that depend on unstable output (timestamps, ordering) without normalizing it.
