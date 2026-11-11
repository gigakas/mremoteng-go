# Pattern Relations

Patterns are not independent recipes. Most real designs combine two or three, and many patterns look alike until you name the *intent*. Use this file to separate look-alikes and to avoid over-composing.

## Decision tree

Answer in order; stop at the first branch that matches.

```
What is the main problem?

1. Object creation?
   ├─ One product, choice at runtime ................ Factory Method or factory function
   ├─ Families of related products .................. Abstract Factory
   ├─ Complex object, many optional parts ........... Builder
   ├─ Expensive/unknown class, copy an instance ..... Prototype
   └─ Exactly one instance required ................. Singleton (prefer module/DI)

2. Object composition or access?
   ├─ Incompatible interface ........................ Adapter
   ├─ Two independent dimensions .................... Bridge
   ├─ Tree / part-whole ............................. Composite
   ├─ Add behavior dynamically ...................... Decorator
   ├─ Simplify a subsystem .......................... Facade
   ├─ Many objects share state (memory) ............. Flyweight
   └─ Control access (lazy/auth/cache/remote) ....... Proxy

3. Communication or responsibility?
   ├─ One-to-many notification ...................... Observer
   ├─ Many-to-many coordination ..................... Mediator
   ├─ Request handled by one of many ................ Chain of Responsibility
   ├─ Encapsulate a request (queue/undo) ............ Command
   ├─ Traverse a structure .......................... Iterator
   ├─ Snapshot/restore .............................. Memento
   ├─ Behavior depends on internal state ............ State
   ├─ Swap an algorithm ............................. Strategy
   ├─ Fixed skeleton, variable steps ................ Template Method
   └─ Add operations to a stable hierarchy .......... Visitor
```

If no branch matches, **do not use a pattern**. A plain function, conditional, or data structure is the right answer.

## Look-alike comparisons

### Adapter vs Bridge vs Decorator vs Proxy vs Facade

| Pattern | Interface | What it changes | Typical trigger |
|---------|-----------|-----------------|-----------------|
| **Adapter** | Different | Translates one interface to another | Integrating a third-party/legacy API |
| **Bridge** | Same abstraction, separate implementor | Splits two dimensions of variation | Shape × renderer, device × OS |
| **Decorator** | Same | Adds behavior before/after delegation | Logging, retry, compression |
| **Proxy** | Same | Controls access (lazy/auth/cache/remote) | Expensive object, permission check |
| **Facade** | New, simpler | Hides a subsystem behind a small API | Complex library orchestration |

### Strategy vs State vs Template Method vs Command

| Pattern | Who chooses behavior | State ownership | Mechanism |
|---------|----------------------|-----------------|-----------|
| **Strategy** | Client | Usually stateless | Composition |
| **State** | The object itself, via transitions | Stateful | Composition |
| **Template Method** | Base class fixes order; subclass fills steps | Inherited | Inheritance |
| **Command** | Invoker triggers a request object | Request carries its data | Composition |

### Factory Method vs Abstract Factory vs Builder vs Prototype

| Pattern | Creates | Key question |
|---------|---------|--------------|
| **Factory Method** | One product, chosen by subclass | "Which class should I instantiate?" |
| **Abstract Factory** | A family of products | "Which family must stay consistent?" |
| **Builder** | One complex product | "How do I assemble many optional parts?" |
| **Prototype** | A copy of an existing object | "Can I clone instead of constructing?" |

### Observer vs Mediator vs Chain of Responsibility

| Pattern | Topology | Direction | Stops? |
|---------|----------|-----------|--------|
| **Observer** | One publisher → many subscribers | Broadcast | No |
| **Mediator** | Many colleagues ↔ one mediator | Routed | Depends |
| **Chain** | Linear handlers | Passed along | Yes, when handled |

### Composite vs Decorator

| Pattern | Structure | Purpose |
|---------|-----------|---------|
| **Composite** | Tree of many components | Treat leaf and branch uniformly |
| **Decorator** | Wrapper around one component | Add behavior |

### Iterator vs Visitor

| Pattern | Responsibility |
|---------|----------------|
| **Iterator** | Traverse elements |
| **Visitor** | Operate on elements without changing their classes |

### Facade vs Mediator

| Pattern | Knows about | Purpose |
|---------|-------------|---------|
| **Facade** | Subsystem internals | Simplify access |
| **Mediator** | Colleagues | Coordinate peer communication |

## Common combinations

| Combination | Why it works | Example |
|-------------|--------------|---------|
| **Factory Method + Strategy** | Factory chooses the strategy; context uses it | Payment provider selected by config |
| **Abstract Factory + Bridge** | Factory creates a consistent implementation family; bridge varies the abstraction | UI toolkit per OS |
| **Builder + Composite** | Builder assembles a tree step by step | Report/document builder |
| **Composite + Iterator** | Uniform tree + traversal | File system, menu tree |
| **Composite + Visitor** | Stable tree + new operations | Export/analyze a document model |
| **Decorator + Chain** | Wrap a handler with logging/auth, then pass along | HTTP middleware |
| **Observer + Mediator** | Mediator coordinates; observer notifies | Chat room + presence events |
| **Command + Memento** | Command executes; memento stores undo state | Editor undo/redo |
| **State + Strategy** | State object delegates algorithm to a strategy | Workflow step with pluggable rules |
| **Adapter + Facade** | Facade simplifies; adapters normalize each subsystem | Unified payment gateway |
| **Proxy + Flyweight** | Proxy controls access; flyweight shares instances | Lazy shared images |

## Combinations to avoid

| Combination | Why it hurts | Better |
|-------------|--------------|--------|
| **Singleton + everything** | Hidden global state, untestable | Inject dependencies |
| **Abstract Factory + Builder + Prototype for one product** | Three creation layers for one object | Pick one creation mechanism |
| **Decorator stack with implicit order** | Behavior depends on invisible ordering | Make order explicit; test the stack |
| **Observer + Mediator + Event Bus** | Three notification paths, hard to trace | Choose one communication model |
| **Strategy per branch of an `if`** | More classes than behavior | Keep the conditional until variation is real |
| **Visitor on a changing hierarchy** | Every new type touches every visitor | Polymorphic method on the elements |

## Pattern density check

Before adding a pattern, ask:

1. **Can I name the problem without the pattern?** If not, stop.
2. **Is there a language primitive?** See `language-idioms.md`.
3. **Does the pattern remove a real pain?** If it only adds structure, stop.
4. **Can a new team member follow it in 5 minutes?** If not, simplify.
5. **Does it make tests easier?** If not, the abstraction is suspect.

A design with five patterns for a CRUD screen is not "enterprise"; it is a maintenance cost.
