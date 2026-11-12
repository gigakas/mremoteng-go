---
name: design-patterns
description: Use when designing class/object structure, comparing patterns ("Strategy or State?"), reviewing a rigid or fragile design, or planning a feature that needs extensibility or swappable implementations. Intent-based selection tables, pattern relations, language idioms, and app recipes.
---

# Design Patterns

Design patterns are typical solutions to common problems in software design. Each one is a blueprint you adapt to your language and problem — not code you paste. Source: [refactoring.guru](https://refactoring.guru/design-patterns).

**Catalog scope**: the 22 patterns catalogued by refactoring.guru — 5 creational, 7 structural, 10 behavioral. The GoF book lists 23; the site (and this skill) omits **Interpreter**, which modern languages usually cover with `match`/`switch` expressions. Terminology, grouping and intent follow the source; language idioms, app recipes and the selection workflow are our additions.

## When to use

- Designing a new feature and deciding how to organize classes/objects.
- Choosing between look-alike patterns (Strategy vs State, Adapter vs Bridge vs Decorator vs Proxy).
- A design feels rigid, fragile, or hard to extend, and you recognize a known problem shape.
- Reviewing a design and asking "is there a proven structure for this?".
- Communicating intent: patterns give the team a shared vocabulary.

## When NOT to use

- You cannot name the concrete problem the pattern solves.
- A language primitive or standard library already provides the behavior.
- The variation does not exist yet ("we might need it later").
- The pattern would add indirection without removing a real pain point.

## How to choose a pattern

### 1) State the problem in one sentence

Write it without naming a pattern: "I need to swap payment providers at runtime", "I need to notify several modules when an order changes". If you cannot state it, you are not ready to choose.

### 2) Classify the intent

| Intent | Group |
|--------|-------|
| Control how objects are created | Creational |
| Compose objects into larger structures | Structural |
| Control communication and responsibility between objects | Behavioral |

### 3) Check what the language already gives you

Many patterns are partially built in. Use the primitive first (see [references/language-idioms.md](references/language-idioms.md)):

- **Iterator** → `for`/`__iter__` (Python), `Symbol.iterator` (TS/JS), `range` (Go).
- **Strategy** → a function value / higher-order function in most languages.
- **Decorator** → function decorators/wrappers, middleware, or proxies.
- **Singleton** → module-level object or DI container.
- **Observer** → built-in events/`EventEmitter`/signals.
- **Command** → closures or function queues.

### 4) Match the intent, then confirm in the table

| You want to… | Pattern |
|--------------|---------|
| Choose which class to instantiate at runtime, defer to subclasses | Factory Method |
| Create families of related objects without depending on concrete classes | Abstract Factory |
| Construct complex objects step by step | Builder |
| Clone existing objects (avoid costly re-initialization) | Prototype |
| Ensure a single instance of a class | Singleton |
| Make incompatible interfaces work together | Adapter |
| Separate abstraction from implementation so both vary independently | Bridge |
| Treat individual objects and compositions uniformly (tree structures) | Composite |
| Add responsibilities to objects dynamically without subclassing | Decorator |
| Provide a simplified interface to a complex subsystem | Facade |
| Share many small objects to save memory | Flyweight |
| Control access to an object (lazy, remote, protected, logging) | Proxy |
| Pass a request along a chain of handlers until one handles it | Chain of Responsibility |
| Turn a request into an object (queue, log, undo) | Command |
| Traverse a collection without exposing its structure | Iterator |
| Reduce direct coupling between many objects via a central coordinator | Mediator |
| Save/restore an object's state (undo, snapshots) | Memento |
| Notify many objects of a state change | Observer |
| Change behavior when an object's internal state changes | State |
| Swap algorithms at runtime | Strategy |
| Define the skeleton of an algorithm, let subclasses fill steps | Template Method |
| Add operations to objects without modifying their classes | Visitor |

### 5) Separate look-alikes before committing

Use [references/pattern-relations.md](references/pattern-relations.md). The most common confusions:

- **Adapter vs Bridge vs Decorator vs Proxy** — all wrap an object; intent differs (reconcile interface / split dimensions / add behavior / control access).
- **Strategy vs State vs Template Method** — client-chosen algorithm / internal-state-driven behavior / fixed skeleton with overridable steps.
- **Factory Method vs Abstract Factory vs Builder** — one product via subclass / family of products / step-by-step construction.
- **Observer vs Mediator** — one-to-many notification / many-to-many coordination.
- **Composite vs Decorator** — tree of many / wrapper of one.

### 6) Sketch the participants

Name the roles (Context + Strategy, Publisher + Subscriber, Component + Leaf + Composite) and map them onto existing classes before writing code. If a role has no real responsibility, the pattern does not fit.

### 7) Implement the smallest version

Start with the minimum number of classes that expresses the pattern. Do not add hooks, registries, or extension points that nothing uses yet.

### 8) Verify behavior, not structure

Tests should assert outcomes through the abstraction. If a test has to know which concrete class was used, the design is probably leaking.

## Pattern or simpler alternative

Reach for a pattern only when the simpler option fails the requirement:

| Problem | Simpler first | Use the pattern when |
|---------|---------------|----------------------|
| Pick behavior by a value | `dict`/`map` of functions, `match` | Behavior must be swappable at runtime or extended by third parties |
| One shared instance | Module-level object, DI singleton | You need lazy creation, reset for tests, or controlled global access |
| Add behavior around a call | Higher-order function, middleware | You need many composable instances and runtime ordering |
| Create one object | Constructor, factory function | Creation must vary independently of the client |
| Create a family | Factory function with a config | Products must stay consistent across a family |
| Notify listeners | Language events/signals | You own the subscription lifecycle and need ordering/filtering |
| Traverse a collection | Built-in iterator/`for` | You need multiple traversal strategies over one structure |
| Undo | Keep a previous value | You need multi-step snapshots and rollback |
| Validate input | Guard clauses / schema | Validation rules form a chain chosen at runtime |

## Recipes for app development

Common application scenarios and the pattern set that fits them live in [references/app-recipes.md](references/app-recipes.md):

- Payments/integrations with interchangeable providers
- Plugin/extension systems
- Request pipelines and middleware
- Event-driven notifications
- Background jobs and undo/redo
- Document/workflow lifecycles
- Reports/exports over a stable model
- Caching, logging, and access control
- Nested trees and permissions
- Complex configuration/builders

## Language idioms

What is idiomatic — and what is already built in — per language is in [references/language-idioms.md](references/language-idioms.md): Python, TypeScript/JavaScript, Java, C#, Go, Rust. Read it before translating a pattern literally.

## Implementation checklist

- [ ] The problem is stated without naming a pattern.
- [ ] The language primitive was considered first.
- [ ] The pattern matches the *intent*, not just the structure.
- [ ] Clients depend on abstractions, not concrete classes.
- [ ] Composition is preferred over inheritance where both work.
- [ ] No unused extension points (no speculative generality).
- [ ] Public names describe the domain, not the pattern (unless the pattern is the vocabulary).
- [ ] Behavior is covered by tests through the abstraction.
- [ ] The reason for the pattern is documented in one short comment or ADR.

## Guardrails

- **Patterns are blueprints, not goals** — adapt them; do not force every class into a role.
- **No pattern without a problem** — speculative structure is technical debt.
- **Prefer composition over inheritance** — Strategy, Decorator, Composite, and Bridge all lean this way.
- **Use the language** — if `__iter__`, decorators, closures, or events solve it, use them.
- **Keep it small** — a pattern that needs six classes for one behavior is probably over-engineered.
- **Make it testable** — if the pattern makes tests harder, the abstraction is wrong.

## Common mistakes

| Mistake | Why It Fails | Fix |
|---------|--------------|-----|
| Pattern without a problem | Unneeded indirection, harder to read | Only apply when a real design problem exists |
| Matching by structure, not intent | Wrong pattern for the job | Diagnose intent first, then choose |
| Forcing a pattern the language provides | Reinventing the wheel | Use `__iter__`, decorators, closures, events |
| Singleton as a global variable | Hidden state, hard to test | Prefer dependency injection / module state |
| Deep pattern nesting | Over-engineering | Compose patterns sparingly |
| Abstract factory for one product | Extra indirection for nothing | Start with Factory Method or a factory function |
| Strategy for two fixed branches | More code than an `if` | Keep the conditional until variation is real |
| Decorator stack with implicit order | Behavior depends on invisible ordering | Make order explicit and test it |
| Observer without unsubscribe | Leaks and double notifications | Define lifecycle/ownership of subscriptions |

## Done when

- [ ] The problem statement names the variation that must be isolated.
- [ ] A simpler alternative (function, composition, language idiom) was rejected with a reason.
- [ ] The chosen pattern maps to an intent, not a copied class diagram.
- [ ] Tests still describe behavior after the change.
- [ ] No extension point exists that the current requirements do not use.

## References

- [references/creational-patterns.md](references/creational-patterns.md) — Factory Method, Abstract Factory, Builder, Prototype, Singleton
- [references/structural-patterns.md](references/structural-patterns.md) — Adapter, Bridge, Composite, Decorator, Facade, Flyweight, Proxy
- [references/behavioral-patterns.md](references/behavioral-patterns.md) — Chain of Responsibility, Command, Iterator, Mediator, Memento, Observer, State, Strategy, Template Method, Visitor
- [references/pattern-relations.md](references/pattern-relations.md) — decision tree, look-alike comparisons, common combinations
- [references/app-recipes.md](references/app-recipes.md) — pattern sets for common application features
- [references/language-idioms.md](references/language-idioms.md) — built-in equivalents and idiomatic variants per language

## Repo adjustments (mremoteng-go)

- **Toolchain**: Go 1.25+ (`go.mod` declares `go 1.25.0`).
- **Language idioms first**: Go function values, closures and one-or-two-method interfaces replace most GoF patterns here. `Strategy` is usually just a function parameter.
- **Existing seams**: session backends, the shared XML/CSV serializer interface and the Fyne driver abstraction. Read them before introducing a pattern they already solve.
- **New dependencies**: every external dependency needs justification in the stage audit; a pattern that removes one is worth the indirection.
- **Do not unify the RDP/AnyDesk path**: it stays an external process. `Adapter` over `exec.Cmd`, never an in-process client.
- **Gate**: `./scripts/check.sh` before finishing. UI changes also need a rebuild with `./scripts/dev.sh` (`-tags x11`) before you claim they work.
- **Changelog**: see the `changelog` skill.
