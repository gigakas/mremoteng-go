---
name: solid-principles
description: Use when designing classes/modules, reviewing architecture, or when code is rigid, fragile, or hard to extend. Applies SRP, OCP, LSP, ISP and DIP with the smallest fix, the smells that signal each violation, and how to decouple modules. Language-agnostic.
---

# SOLID Principles

Five design principles that make code easier to understand, change, and test. They are the conceptual glue between the `design-patterns` and `refactoring` skills — most design patterns exist to satisfy SOLID; most refactorings move code *toward* SOLID.

## When to use

- Designing a new class, module, or interface
- Reviewing code for rigidity, fragility, or tight coupling
- Deciding how to split responsibilities or reduce dependencies
- Diagnosing *why* a design keeps breaking when requirements change

## When NOT to use

- Trivial scripts, glue, and value objects with one implementation and no expected variation.
- Before the second variation exists — speculative abstraction is the exact failure this skill warns about.
- A local smell that a small refactoring already removes — use `refactoring` first.
- The framework dictates the seams (an ORM model, a generated client, a framework controller) — respect the boundary before applying DIP.

## The five principles

| Acronym | Principle | One-line summary |
|---------|-----------|------------------|
| **S** | Single Responsibility | A class has one reason to change |
| **O** | Open/Closed | Open for extension, closed for modification |
| **L** | Liskov Substitution | Subtypes must be substitutable for their base types |
| **I** | Interface Segregation | Many small interfaces beat one fat interface |
| **D** | Dependency Inversion | Depend on abstractions, not concretions |

## S — Single Responsibility Principle (SRP)

- **Rule**: A class/module should have only one reason to change. Change should affect one isolated piece, not ripple through unrelated logic.
- **Signs of violation**: A class has many unrelated methods; you describe it with "and" ("it validates *and* saves *and* emails"); changes to feature A keep touching the class you were editing for feature B.
- **Fix**: Extract Class, Move Method/Move Field (see the `refactoring` skill).
- **Related smells**: Large Class, Divergent Change.

## O — Open/Closed Principle (OCP)

- **Rule**: Software entities should be open for extension but closed for modification — add new behavior by adding new code, not editing existing code.
- **Signs of violation**: Every new case means editing a long `if/switch` in existing methods; repeated modification of the same class for new types.
- **Fix**: Introduce polymorphism/abstraction — Strategy, Factory Method, Template Method, Decorator (see `design-patterns`).
- **Related smells**: Switch Statements, Speculative Generality (don't pre-build extension points you don't need).

## L — Liskov Substitution Principle (LSP)

- **Rule**: A subclass must be usable anywhere its base class is expected, without breaking behavior or surprising callers.
- **Signs of violation**: A subclass throws `NotImplemented`, no-ops an inherited method, or changes pre/post-conditions (e.g., narrows accepted inputs, widens required outputs).
- **Fix**: Replace Inheritance with Delegation, or move the offending method down the hierarchy.
- **Related smells**: Refused Bequest, Parallel Inheritance Hierarchies.

```
# Violation: a Square that isn't a valid Rectangle
class Rectangle: set_width(w); set_height(h)
class Square(Rectangle): set_width(w) also changes height   # breaks LSP
```

## I — Interface Segregation Principle (ISP)

- **Rule**: Clients should not be forced to depend on methods they don't use. Split fat interfaces into smaller, role-specific ones.
- **Signs of violation**: Implementors stub out irrelevant methods; a client takes a giant interface but uses one method.
- **Fix**: Split the interface; clients depend on the narrow interface they actually need.
- **Related smell**: Refused Bequest (for interfaces), Temporary Field.

## D — Dependency Inversion Principle (DIP)

- **Rule**: High-level modules should not depend on low-level modules; both should depend on abstractions. Abstractions should not depend on details.
- **Signs of violation**: High-level business logic imports/constructs concrete low-level classes (DB, HTTP, file I/O) directly.
- **Fix**: Depend on an interface/port; inject the concrete implementation (constructor/setter injection). Enables test doubles (see `testing-patterns`).
- **Related patterns**: Factory Method, Abstract Factory, Adapter, Facade.

```
# Violation: high-level logic directly depends on a concrete database
class OrderService:
    def __init__(self): self.db = PostgresDatabase()   # hard-wired

# Fix: depend on an abstraction, inject the concrete
class OrderService:
    def __init__(self, db: Repository): self.db = db   # inject any Repository
```

## Procedure

1. **Identify the pain**: which principle is being violated? (rigidity → OCP/DIP, tangled responsibilities → SRP, surprising subclass behavior → LSP, fat interfaces → ISP).
2. **Choose the smallest fix**: prefer a refactoring (move/extract) over introducing a new abstraction or pattern.
3. **Apply one change, keep tests green** (see `test-driven-development` and `refactoring`).
4. **Stop at "good enough"**: SOLID is a guide, not a goal — don't over-abstract (Speculative Generality).

## Guardrails

- **SOLID is not an end in itself** — apply where it reduces coupling and aids change; don't abstract speculatively.
- **Prefer composition over inheritance** — it satisfies LSP/OCP with less coupling.
- **One reason to change is a guideline, not a metric** — use it to spot clusters, not to split every class into atoms.
- **Don't violate DIP for trivial value objects** — depend on abstractions where behavior varies, not everywhere.

## Common mistakes

| Mistake | Why It Fails | Fix |
|---------|--------------|-----|
| Over-applying SRP | A thousand tiny classes, harder to navigate | Group cohesive responsibilities; one *reason to change* |
| Abstracting for OCP prematurely | Unused extension points | Add abstraction when the second variation appears |
| Subclass that violates LSP | Silent bugs, surprising callers | Prefer delegation; only inherit for true is-a |
| Fat interfaces (violating ISP) | Clients coupled to unused methods | Split interfaces by role/consumer |
| Depending on concretions (violating DIP) | Rigid, untestable code | Depend on abstractions; inject dependencies |
| Treating SOLID as rules to obey always | Dogmatic, over-engineered code | Use as heuristics; prioritize clarity |

## Done when

- [ ] The principle is named, and the violation is pointed at in a file and line.
- [ ] The fix is the smallest change that restores substitutability or separation.
- [ ] Tests stayed green; new seams are covered by at least one test or architecture rule.
- [ ] No abstraction was added without a second variation to justify it.

## References

- `../design-patterns/references/language-idioms.md` — how each principle is expressed idiomatically per language
- `../refactoring/references/code-smells.md` — the smells that signal a violation
- `../testing-patterns/references/architecture-tests.md` — executable rules that keep DIP/SRP from regressing
- `../refactoring/SKILL.md` — the mechanics (Extract Class, Move Method, Replace Inheritance with Delegation)

## Repo adjustments (mremoteng-go)

- **Toolchain**: Go 1.25+ (`go.mod` declares `go 1.25.0`).
- **Package boundaries**: `internal/` packages are the unit of responsibility; `internal/ui` may depend on `internal/profiles` and `internal/connection`, never the reverse.
- **Interfaces**: this codebase already has small seams (session backends, serializers). Do not add an interface for a single implementation — DIP is satisfied by the seam, not by the number of types.
- **Gate**: `./scripts/check.sh` is mandatory before finishing; `gofmt -w <files>` first if it reports formatting.
- **Size rule**: functions over 50 lines or with mixed responsibilities get split before they are extended (AGENTS.md).
- **Changelog**: see the `changelog` skill; every logical change needs a fragment.
- **Boundary not to abstract**: RDP and AnyDesk run as external processes on purpose. Do not "improve" that into a linked client — the license boundary is documented in the migration plan.
