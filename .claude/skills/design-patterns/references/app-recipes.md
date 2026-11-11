# App-Development Recipes

Common application features mapped to a small pattern set. Each recipe lists the signals, the structure, the steps, and when *not* to use it. Start with the smallest set; add the next pattern only when the variation is real.

For look-alike comparisons, see `pattern-relations.md`. For language primitives, see `language-idioms.md`.

---

## 1. Interchangeable providers (payments, shipping, auth, storage)

**Signals**: several providers behind one feature; the choice comes from config/tenant/runtime; each provider has a different SDK.

**Pattern set**: **Strategy** + **Factory** + **Adapter**.

**Structure**:

```
Checkout ──> PaymentStrategy (interface)
                 ├── StripeAdapter ──> Stripe SDK
                 ├── PayPalAdapter ──> PayPal SDK
                 └── ManualAdapter
PaymentFactory: config/provider name ──> PaymentStrategy
```

**Steps**:

1. Define the port your domain needs (`authorize`, `capture`, `refund`), not the SDK's API.
2. Write one adapter per provider; translate errors and data shapes.
3. Add a factory/registry that returns the strategy from config.
4. Inject the strategy into the use case; never import an SDK in the domain.
5. Test the use case with a fake strategy; test each adapter against a sandbox/fake.

**Avoid when**: there is exactly one provider and no near-term second one. A direct adapter is enough; skip the factory.

---

## 2. Plugin / extension system

**Signals**: third parties or modules add behavior; the core must not know concrete plugins; plugins are discovered/configured at startup.

**Pattern set**: **Factory Method** or **Abstract Factory** + **Registry** + **Strategy** (or **Command** for actions).

**Structure**:

```
PluginRegistry ──> PluginFactory ──> Plugin (interface)
                                      ├── CsvExporter
                                      ├── PdfExporter
                                      └── WebhookExporter
```

**Steps**:

1. Define a narrow plugin contract (one capability per plugin).
2. Provide a registry keyed by a stable name/version.
3. Let plugins register themselves at startup; validate the contract.
4. Isolate plugin failures: one broken plugin must not crash the host.
5. Version the contract; document compatibility.

**Avoid when**: the "plugins" are internal modules you control. Use plain modules and explicit wiring; a registry adds indirection.

---

## 3. Request pipeline / middleware

**Signals**: cross-cutting steps around a request (auth, logging, validation, rate limit, retry); order matters; steps are composable.

**Pattern set**: **Chain of Responsibility** + **Decorator**.

**Structure**:

```
Request -> Auth -> RateLimit -> Validate -> Handler
                \-> Logger (decorator around the chain)
```

**Steps**:

1. Define a handler interface with `handle(request, next)` or a list of middleware.
2. Keep each step single-purpose and order-explicit.
3. Decide whether a step can short-circuit (auth) or must always delegate (logging).
4. Test the chain order and the short-circuit behavior.
5. Make the order configuration visible, not hidden in imports.

**Avoid when**: there are two fixed steps. A function call sequence is clearer than a chain.

---

## 4. Domain events / notifications

**Signals**: several modules react to a state change; the publisher should not know subscribers; subscribers can be added/removed.

**Pattern set**: **Observer** + **Mediator** (when many-to-many coordination appears).

**Structure**:

```
OrderService ──publish──> EventBus ──> EmailHandler
                                   ├── InventoryHandler
                                   └── AuditHandler
```

**Steps**:

1. Define event types as immutable data.
2. Publish after the transaction commits (or use an outbox) to avoid notifying on rollback.
3. Make subscription lifecycle explicit; unsubscribe on shutdown.
4. Decide sync vs async; document ordering and retry semantics.
5. Add an idempotency key for handlers that can be retried.

**Avoid when**: the "subscriber" is the same module and a direct call is simpler. Observer is for decoupling, not for avoiding a function call.

---

## 5. Background jobs / undo-redo

**Signals**: work must be queued, retried, logged, or reversed; the sender should not wait.

**Pattern set**: **Command** + **Memento**.

**Structure**:

```
Invoker -> Command (execute/undo) -> Receiver
                 └── Memento (state snapshot)
```

**Steps**:

1. Encapsulate each job as a command with its payload.
2. Serialize the command for the queue; keep it small and idempotent.
3. Store enough state (memento) for undo if the feature needs it.
4. Define retry, dead-letter, and idempotency behavior.
5. Test the command in isolation and the queue integration separately.

**Avoid when**: the action is a simple function call with no queue/undo requirement. A closure is enough.

---

## 6. Document / workflow lifecycle

**Signals**: a record moves through states (draft → submitted → approved → cancelled); behavior and allowed transitions depend on the state; the same `switch` appears in many methods.

**Pattern set**: **State**.

**Structure**:

```
Document -> State (interface)
              ├── Draft
              ├── Submitted
              └── Approved
```

**Steps**:

1. Enumerate states and allowed transitions explicitly.
2. Put state-specific behavior in state classes; validate transitions there.
3. Keep the context's public API stable (`submit`, `approve`, `cancel`).
4. Persist the state key, not the state object.
5. Test every transition, including invalid ones.

**Avoid when**: there are two states and one boolean. A flag and guards are clearer.

---

## 7. Reports / exports over a stable model

**Signals**: many operations over a stable class hierarchy (PDF, CSV, JSON, audit, tax calculation); adding an operation touches every class.

**Pattern set**: **Visitor** + **Composite** (for tree-shaped models).

**Structure**:

```
ReportElement (interface) ──accept(visitor)──> ReportVisitor
  ├── Section (composite)                        ├── PdfVisitor
  └── Field (leaf)                               └── CsvVisitor
```

**Steps**:

1. Confirm the element hierarchy is stable; Visitor is a bad fit for frequently added types.
2. Define a visitor per operation, not per element.
3. Use Composite if the report is a tree.
4. Keep traversal in an iterator or the composite; keep operations in visitors.
5. Test each visitor against a representative model.

**Avoid when**: the model changes often, or the language has exhaustive pattern matching over a closed union (TS discriminated unions, Rust enums, Python `match`). Use that first.

---

## 8. Caching, logging, access control

**Signals**: the same object needs lazy loading, permission checks, caching, or logging without changing its interface.

**Pattern set**: **Proxy** (access control/lazy/cache) + **Decorator** (behavior).

**Structure**:

```
Client -> Subject (interface)
            ├── RealSubject
            ├── LazyProxy
            ├── CachedProxy
            └── LoggingDecorator
```

**Steps**:

1. Decide whether you are controlling access (Proxy) or adding behavior (Decorator).
2. Keep the same interface; do not leak the proxy in public types.
3. Define cache invalidation and permission failure behavior explicitly.
4. Test the proxy with a fake real subject, including failure paths.
5. Avoid stacking proxies with implicit order; make order explicit.

**Avoid when**: a single wrapper function or language-level decorator solves it. Do not create a class hierarchy for one behavior.

---

## 9. Nested trees and permissions

**Signals**: categories/folders/menus with unlimited depth; operations must work on a leaf and a branch uniformly; permissions inherit down the tree.

**Pattern set**: **Composite** + **Iterator** (+ **Visitor** for operations).

**Structure**:

```
Node (interface)
  ├── Leaf
  └── Composite (children: Node[])
```

**Steps**:

1. Define the common operation (`size`, `render`, `can_read`) on the component.
2. Keep child management on the composite, not the leaf.
3. Use an iterator for traversal; avoid exposing internal arrays.
4. Decide inheritance rules for permissions and test cycles/depth limits.
5. Add a Visitor only when operations multiply.

**Avoid when**: the tree has a fixed depth of one or two. Explicit fields are simpler.

---

## 10. Complex configuration / query objects

**Signals**: many optional parameters; construction has an order; callers build the same complex object repeatedly; telescoping constructors.

**Pattern set**: **Builder** (fluent or director-based).

**Structure**:

```
QueryBuilder.select(...).where(...).orderBy(...).limit(...).build()
```

**Steps**:

1. Separate required from optional fields.
2. Make each step return the builder; validate in `build()`.
3. Keep the built object immutable.
4. Add a director only if multiple representations share the same steps.
5. Test validation and the final object, not the builder internals.

**Avoid when**: the object has two or three fields. A constructor or a plain object literal is clearer.

---

## 11. Testable infrastructure (ports and adapters)

**Signals**: business logic imports the database/HTTP SDK directly; tests need a real service; swapping infrastructure is painful.

**Pattern set**: **Dependency Inversion** + **Adapter** + **Factory**.

**Structure**:

```
UseCase -> Repository (port) <- SqlRepository (adapter)
                             <- InMemoryRepository (test fake)
```

**Steps**:

1. Define ports that match the use case's needs, not the framework's API.
2. Implement adapters at the boundary; keep domain types free of SDK types.
3. Inject ports into the use case; construct them in a composition root.
4. Test the use case with an in-memory fake; test the adapter with an integration test.
5. Enforce the dependency direction with an architecture test (see `../testing-patterns/references/architecture-tests.md`).

**Avoid when**: the "infrastructure" is a pure function or a value object. Not everything needs a port.

---

## Choosing a recipe

| If the problem is… | Start with |
|--------------------|------------|
| Swapping providers/algorithms | Strategy + Factory |
| Adding plugins/extensions | Plugin contract + Registry |
| Cross-cutting request steps | Chain + Decorator |
| Reacting to changes | Observer (then Mediator if many-to-many) |
| Queue/retry/undo | Command + Memento |
| State-dependent behavior | State |
| Many operations on a stable model | Visitor (or exhaustive match) |
| Access control/lazy/cache | Proxy |
| Trees | Composite + Iterator |
| Complex construction | Builder |
| Testability/infrastructure | Ports + Adapters + DI |

> If a recipe needs more than three patterns for a CRUD feature, the design is probably over-engineered. Remove the pattern that no current requirement needs.
