# Refactoring Techniques

Catalog of refactoring techniques grouped by purpose. Each entry: what it does and when to use it. Source: [refactoring.guru/techniques](https://refactoring.guru/refactoring/techniques).

## 1. Composing Methods

Simplify methods, remove code duplication, make them easier to understand.

- **Extract Method** — Move a fragment into a new method named by its intent. The most common refactoring. Use when a method is too long or needs explanatory comments.
- **Inline Method** — Replace a method call with its body when the method is as clear as its name. Reverse of Extract Method.
- **Extract Variable** — Put a complex expression's result in a self-explanatory variable. Use for long conditions/expressions.
- **Inline Temp** — Replace a temp variable that's just assigned from a simple expression with the expression itself.
- **Replace Temp with Query** — Replace a local variable holding a computed result with a method/query. Use to avoid passing temps around and to enable reuse.
- **Split Temporary Variable** — Use separate variables for different values assigned to the same temp (not loop accumulators). Use when a variable is reused for unrelated things.
- **Remove Assignments to Parameters** — Stop reassigning a function parameter; use a local variable instead.
- **Replace Method with Method Object** — Turn a long method with many locals into a class where locals become fields; then extract methods freely.
- **Substitute Algorithm** — Replace a method body with a clearer/cleaner algorithm.

## 2. Moving Features between Objects

Move functionality to the class where it belongs, hide implementation details, reduce coupling.

- **Move Method** — Move a method to the class it uses most. Use to fix Feature Envy.
- **Move Field** — Move a field to the class that uses it most.
- **Extract Class** — Split a class into two when part of its behavior/data can be separated. Use for Large Class / Divergent Change.
- **Inline Class** — Merge a class that does too little into the class that uses it. Use for Lazy Class.
- **Hide Delegate** — Add a wrapper method instead of exposing a delegate object (`a.getB().getC()` → `a.getC()`). Use to fix Message Chains.
- **Remove Middle Man** — When a class delegates too much, call the delegate directly. Reverse of Hide Delegate.
- **Introduce Foreign Method** — Add a standalone utility method that operates on a library object you can't modify. Use for Incomplete Library Class.
- **Introduce Local Extension** — Extend a library class via subclass or wrapper (composition) to add methods.

## 3. Organizing Data

Make working with data easier, replace primitives with richer objects, unify links.

- **Self Encapsulate Field** — Access a field through getters/setters rather than directly. Use when you need to control access.
- **Replace Data Value with Object** — Wrap a primitive field (e.g., a phone number) in a small class. Use for Primitive Obsession.
- **Change Value to Reference** — Replace many copies of an equal object with a single shared instance.
- **Change Reference to Value** — Replace shared objects with distinct copies when shared mutable state is a problem.
- **Replace Array with Object** — Replace an array/record used for heterogenous data with a class with named fields.
- **Duplicate Observed Data** — Copy data to a dedicated object (e.g., separate domain model from UI data).
- **Change Unidirectional Association to Bidirectional** — Add back-references when both sides need to navigate.
- **Change Bidirectional Association to Unidirectional** — Remove a back-reference that's no longer needed.
- **Replace Magic Number with Symbolic Constant** — Replace hard-coded literals with named constants.
- **Encapsulate Field** — Make public fields private and expose via accessors.
- **Encapsulate Collection** — Return a read-only view of a collection instead of the mutable collection itself.
- **Replace Type Code with Class** — Replace a type-code primitive with a small class.
- **Replace Type Code with Subclasses** — Replace a type code with a polymorphic subclass hierarchy.
- **Replace Type Code with State/Strategy** — Replace a type code with a State/Strategy object (when subclassing isn't possible).
- **Replace Subclass with Fields** — Remove subclasses whose only difference is a constant value; use a field instead.

## 4. Simplifying Conditional Expressions

Make conditionals easier to read and understand.

- **Decompose Conditional** — Extract the condition and each branch into named methods. Use when the "why" of a condition is unclear.
- **Consolidate Conditional Expression** — Merge multiple conditionals that lead to the same result into one.
- **Consolidate Duplicate Conditional Fragments** — Move identical code out of all branches to after the conditional.
- **Remove Control Flag** — Replace a boolean flag that controls a loop/break with `break`/`continue`/`return`.
- **Replace Nested Conditional with Guard Clauses** — Replace deep nesting with early `return`/`continue`/`raise` guards.
- **Replace Conditional with Polymorphism** — Replace type-based conditionals with polymorphic subclasses/strategy. Use for Switch Statements.
- **Introduce Null Object** — Replace `None`/null checks with a neutral object that does nothing. Use for repeated null checks.
- **Introduce Assertion** — Make assumptions explicit with an assertion at the top of a method.

## 5. Simplifying Method Calls

Make method interfaces simpler and easier to use.

- **Rename Method** — Give a method a name that says what it does.
- **Add Parameter** / **Remove Parameter** — Adjust the signature when the method needs more (or no longer needs) data.
- **Separate Query from Modifier** — Split a method that returns a value *and* changes state into two methods.
- **Parameterize Method** — Merge methods that do the same thing with different constants into one method with a parameter.
- **Replace Parameter with Explicit Methods** — Replace a method that switches on a parameter with separate named methods. Reverse of Parameterize Method.
- **Preserve Whole Object** — Pass the whole object instead of many of its fields.
- **Replace Parameter with Method Call** — Replace a parameter with a call to the method that can provide it (when the parameter is derivable).
- **Introduce Parameter Object** — Group parameters that naturally go together into a single object. Use for Long Parameter List / Data Clumps.
- **Remove Setting Method** — Make a field immutable by removing its setter (set via constructor only).
- **Hide Method** — Make an unused internal method private.
- **Replace Constructor with Factory Method** — Replace a constructor with a factory method for more control (caching, subclasses, clarity).
- **Replace Error Code with Exception** — Replace returned error codes with exceptions.
- **Replace Exception with Test** — Replace an exception thrown for control flow with an explicit condition check.

## 6. Dealing with Generalization

Move methods/fields up and down an inheritance hierarchy; add or remove classes.

- **Pull Up Field / Pull Up Method** — Move a field/method duplicated in subclasses to the superclass.
- **Pull Up Constructor Body** — Move common constructor logic to the superclass constructor.
- **Push Down Method / Push Down Field** — Move a method/field only used by some subclasses down to those subclasses.
- **Extract Subclass** — Create a subclass for a subset of features used only in some cases.
- **Extract Superclass** — Create a superclass for shared features of two classes.
- **Extract Interface** — Extract a common interface from a class so clients depend on the contract.
- **Collapse Hierarchy** — Merge a superclass/subclass that are too similar (or a subclass that's a Lazy Class).
- **Form Template Method** — Unify duplicated algorithm skeletons in subclasses into a template method with hooks.
- **Replace Inheritance with Delegation** — Replace a subclass with a separate object the class delegates to. Prefer composition. Use for Refused Bequest.
- **Replace Delegation with Inheritance** — Replace a delegating class with inheritance when the delegating class mirrors the whole interface.

## How to apply any technique

1. **Make the smallest mechanical change** that expresses the technique.
2. **Compile/lint** if the language needs it.
3. **Run the focused tests** (see `../testing-patterns/references/test-strategy.md`).
4. **Inspect the diff** — only structure should change.
5. **Commit** when green; undo immediately when red.

Do not combine two techniques in one step. If the change needs a new name, rename in a separate step.

## Core mechanics with before/after

### Extract Method

**Use when**: a method is too long, needs a comment to explain a block, or a block is duplicated.

**Mechanics**:
1. Create a new method named after *what* the block does (not *how*).
2. Copy the block into it.
3. Pass the variables the block reads; return the values it writes.
4. Replace the original block with a call.
5. Run tests.

```python
# Before
def print_invoice(invoice):
    print("**************************")
    print("***** Customer Invoice ****")
    print("**************************")
    for item in invoice.items:
        print(f"{item.name}: {item.amount}")

# After
def print_banner(title):
    line = "*" * (len(title) + 8)
    print(line)
    print(f"**** {title} ****")
    print(line)

def print_invoice(invoice):
    print_banner("Customer Invoice")
    for item in invoice.items:
        print(f"{item.name}: {item.amount}")
```

### Extract Variable

**Use when**: an expression is long or its meaning is unclear.

```python
# Before
if order.total > 100 and order.customer.is_premium and not order.has_discount:
    apply_free_shipping(order)

# After
qualifies_for_free_shipping = (
    order.total > 100
    and order.customer.is_premium
    and not order.has_discount
)
if qualifies_for_free_shipping:
    apply_free_shipping(order)
```

### Replace Nested Conditional with Guard Clauses

**Use when**: the happy path is buried inside nested `if`s.

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

### Replace Conditional with Polymorphism

**Use when**: the same `if/elif` on a type or status is repeated, and each branch has behavior.

```python
# Before
def shipping_cost(order):
    if order.type == "standard":
        return 5
    elif order.type == "express":
        return 15
    elif order.type == "international":
        return 40
    raise ValueError(order.type)

# After
class Shipping:
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

**Do not** apply this to a single `switch` that changes in one place; the smell is *repeated* type-based branching.

### Introduce Parameter Object

**Use when**: the same group of parameters travels together (Data Clump) or the list is long.

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

def search(criteria: SearchCriteria):
    ...
```

### Extract Class

**Use when**: a class has two or more unrelated responsibilities, or a subset of fields/methods always changes together.

**Mechanics**:
1. Create a new class for the cohesive subset.
2. Move fields and methods that belong to it.
3. Keep a reference from the original class (composition).
4. Delegate temporarily, then update callers.
5. Run tests after each move.

```python
# Before: Order knows about pricing and persistence
class Order:
    def total(self): ...
    def save(self): ...
    def send_confirmation(self): ...

# After: split responsibilities
class Order:
    def __init__(self, pricing: Pricing, repo: OrderRepository, mailer: Mailer):
        self.pricing, self.repo, self.mailer = pricing, repo, mailer

    def total(self): return self.pricing.total(self)
    def save(self): self.repo.save(self)
    def send_confirmation(self): self.mailer.send(self)
```

### Move Method

**Use when**: a method uses another class's data more than its own (Feature Envy).

**Mechanics**:
1. Copy the method to the target class.
2. Adjust references to the source (`self` → parameter or field).
3. Turn the original into a delegating call, or update callers.
4. Run tests; remove the old method when safe.

### Hide Delegate

**Use when**: clients walk a chain (`order.customer.address.city`) or know too much about a collaborator.

```python
# Before
city = order.customer.address.city

# After
class Order:
    def shipping_city(self):
        return self.customer.address.city

city = order.shipping_city()
```

### Replace Inheritance with Delegation

**Use when**: a subclass refuses most of the parent's behavior (Refused Bequest) or only inherits for code reuse.

```python
# Before
class Stack(list):        # inherits append, insert, pop(index), etc.
    pass

# After
class Stack:
    def __init__(self):
        self._items = []
    def push(self, item): self._items.append(item)
    def pop(self): return self._items.pop()
    def __len__(self): return len(self._items)
```

### Introduce Null Object

**Use when**: the same `if x is None` check is repeated and the "do nothing" behavior is valid.

```python
# Before
def notify(user):
    if user is not None:
        user.send("hello")

# After
class NullUser:
    def send(self, message): pass

def notify(user):
    (user or NullUser()).send("hello")
```

### Replace Magic Number with Symbolic Constant

**Use when**: a literal has a meaning that the reader must guess.

```python
# Before
if age >= 18:
    ...

# After
LEGAL_AGE = 18
if age >= LEGAL_AGE:
    ...
```

## Choosing between similar techniques

| Situation | Prefer | Instead of |
|-----------|--------|------------|
| Long method with clear blocks | Extract Method | Replace Method with Method Object |
| Long method with many locals | Replace Method with Method Object | Repeated Extract Method |
| Two responsibilities in one class | Extract Class | Extract Subclass |
| Subclass for a subset of features | Extract Subclass | Extract Class |
| Repeated type switch | Replace Conditional with Polymorphism | Parameterize Method |
| One method with a boolean flag | Replace Parameter with Explicit Methods | Two boolean parameters |
| Many parameters that travel together | Introduce Parameter Object | Preserve Whole Object |
| Null checks everywhere | Introduce Null Object | Optional chaining everywhere |
