# Creational Patterns

Patterns that provide object-creation mechanisms, increasing flexibility and reuse. Source: [refactoring.guru/creational-patterns](https://refactoring.guru/design-patterns/creational-patterns).

For each pattern: intent, problem, solution, applicability, participants, trade-offs, relations, and Python/TypeScript examples. Use the language primitive first — see `language-idioms.md`.

---

## Factory Method

- **Intent**: Define an interface for creating an object, but let subclasses decide which class to instantiate.
- **Problem**: The exact type to create depends on runtime information, and the client should not be coupled to concrete classes.
- **Solution**: A `Creator` declares a `factory_method()` returning a product interface. Concrete creators override it. Client code calls the factory method, not a constructor.
- **Use when**: You want to centralize creation, let subclasses choose the product, or provide an extension point for new product types.
- **Participants**: `Product`, `ConcreteProduct`, `Creator`, `ConcreteCreator`.
- **Trade-offs**: + decouples client from concrete products; + supports extension. − adds a class per product family; − can be replaced by a factory function in simple cases.
- **Relations**: Often the first step before **Abstract Factory**; **Template Method** often calls a factory method; **Prototype** avoids subclassing creators.

```python
from abc import ABC, abstractmethod

class Transport(ABC):
    @abstractmethod
    def deliver(self) -> str: ...

class Truck(Transport):
    def deliver(self) -> str: return "by road"

class Logistics(ABC):
    @abstractmethod
    def create_transport(self) -> Transport: ...

    def plan(self) -> str:
        return self.create_transport().deliver()   # client uses the abstraction

class RoadLogistics(Logistics):
    def create_transport(self) -> Transport: return Truck()
```

```ts
interface Transport { deliver(): string }

class Truck implements Transport { deliver() { return "by road" } }

abstract class Logistics {
  protected abstract createTransport(): Transport
  plan(): string { return this.createTransport().deliver() }
}

class RoadLogistics extends Logistics {
  protected createTransport(): Transport { return new Truck() }
}
```

**Pitfalls**: a factory method per class when a single factory function with a config would do; forgetting to return the abstraction instead of the concrete type.

---

## Abstract Factory

- **Intent**: Create families of related objects without specifying their concrete classes.
- **Problem**: The system must work with several families of products (e.g., UI widgets per theme/OS) and the products in a family must be used together.
- **Solution**: An abstract factory declares one creation method per product type; each concrete factory produces a consistent family. Clients depend only on the abstract interfaces.
- **Use when**: Products come in families, you need to enforce consistency within a family, and you want to swap families at runtime.
- **Participants**: `AbstractFactory`, `ConcreteFactory`, `AbstractProductA/B`, `ConcreteProductA1/B1…`.
- **Trade-offs**: + guarantees family consistency; + isolates concrete classes. − adding a new product type means changing every factory; − more interfaces/classes.
- **Relations**: Often implemented with **Factory Method**; **Builder** focuses on one complex product, Abstract Factory on families.

```python
class Button(ABC):
    @abstractmethod
    def paint(self) -> str: ...

class DarkButton(Button):
    def paint(self) -> str: return "dark button"

class LightButton(Button):
    def paint(self) -> str: return "light button"

class UIFactory(ABC):
    @abstractmethod
    def button(self) -> Button: ...

class DarkFactory(UIFactory):
    def button(self) -> Button: return DarkButton()

def render(factory: UIFactory) -> str:
    return factory.button().paint()   # works with any family
```

```ts
interface Button { paint(): string }
class DarkButton implements Button { paint() { return "dark button" } }
class LightButton implements Button { paint() { return "light button" } }

interface UIFactory { button(): Button }
class DarkFactory implements UIFactory { button(): Button { return new DarkButton() } }

function render(factory: UIFactory) { return factory.button().paint() }
```

**Pitfalls**: using it when only one family exists; letting clients construct concrete products and bypass the family.

---

## Builder

- **Intent**: Construct a complex object step by step; the same process can create different representations.
- **Problem**: A constructor has many optional parameters (telescoping constructor) or the object needs a specific assembly order.
- **Solution**: A `Builder` interface declares `build_part()` steps; a `ConcreteBuilder` tracks the result and exposes `get_result()`. A `Director` (optional) defines the build order.
- **Use when**: Objects have many optional parts, construction is multi-step, or you want a readable fluent API.
- **Participants**: `Builder`, `ConcreteBuilder`, `Director` (optional), `Product`.
- **Trade-offs**: + readable construction; + reuses build steps; + avoids telescoping constructors. − more code than a plain constructor; − mutable builder can be misused.
- **Relations**: **Abstract Factory** creates families; **Builder** creates one complex object step by step. **Fluent interfaces** are a common lightweight variant.

```python
class QueryBuilder:
    def __init__(self) -> None:
        self._parts: list[str] = []

    def select(self, *cols: str) -> "QueryBuilder":
        self._parts.append(f"SELECT {', '.join(cols)}"); return self

    def where(self, cond: str) -> "QueryBuilder":
        self._parts.append(f"WHERE {cond}"); return self

    def build(self) -> str:
        return " ".join(self._parts)

sql = QueryBuilder().select("id", "name").where("age > 18").build()
```

```ts
class QueryBuilder {
  private parts: string[] = []
  select(...cols: string[]) { this.parts.push(`SELECT ${cols.join(", ")}`); return this }
  where(cond: string) { this.parts.push(`WHERE ${cond}`); return this }
  build() { return this.parts.join(" ") }
}
const sql = new QueryBuilder().select("id", "name").where("age > 18").build()
```

**Pitfalls**: builder with no optional parts; exposing the mutable product before `build()`; using a builder for a 2-field value object.

---

## Prototype

- **Intent**: Copy existing objects without depending on their concrete classes.
- **Problem**: Creating an object is expensive, or its class is not known at compile time, but you have an instance to copy.
- **Solution**: A `Prototype` interface declares `clone()`. Concrete prototypes implement copying (shallow or deep) and may carry a registry of reusable prototypes.
- **Use when**: Creation is costly (DB/network), you need many similar objects, or you want to avoid parallel factory hierarchies.
- **Participants**: `Prototype`, `ConcretePrototype`, `PrototypeRegistry` (optional), `Client`.
- **Trade-offs**: + avoids costly re-initialization; + removes creator subclasses. − deep copying is error-prone (cycles, shared references); − cloning can bypass invariants.
- **Relations**: Alternative to **Factory Method**/Abstract Factory when creation is expensive; often combined with **Memento** for snapshots.

```python
import copy

class Document:
    def __init__(self, title: str, sections: list[str]) -> None:
        self.title = title
        self.sections = sections

    def clone(self) -> "Document":
        return copy.deepcopy(self)   # independent copy

original = Document("Report", ["Intro"])
copy_doc = original.clone()
copy_doc.sections.append("Body")
```

```ts
class Document {
  constructor(public title: string, public sections: string[]) {}
  clone(): Document { return structuredClone(this) }
}
const original = new Document("Report", ["Intro"])
const copyDoc = original.clone()
copyDoc.sections.push("Body")
```

**Pitfalls**: shallow copy where deep is needed; cloning objects that hold resources (sockets, file handles); forgetting to re-register observers after cloning.

---

## Singleton

- **Intent**: Ensure a class has only one instance and provide a global access point.
- **Problem**: A shared resource (config, connection pool, cache) must have exactly one instance.
- **Solution**: A private constructor plus a static accessor that lazily creates the instance.
- **Use when**: Exactly one instance is a real domain constraint and global access is acceptable.
- **Participants**: `Singleton` (private constructor, static `get_instance()`).
- **Trade-offs**: + controlled single instance; + lazy initialization. − hidden global state; − hard to test and mock; − can hide design problems.
- **Relations**: Often replaced by **Dependency Injection** or a module-level object; **Facade** may expose a singleton, but the facade itself does not need to be one.

```python
class Config:
    _instance: "Config | None" = None

    def __new__(cls) -> "Config":
        if cls._instance is None:
            cls._instance = super().__new__(cls)
            cls._instance.values = {}
        return cls._instance

# Prefer this in Python: a module-level object is already a singleton.
config = Config()
```

```ts
class Config {
  private static instance: Config
  private constructor(public values: Record<string, string> = {}) {}
  static get instance(): Config {
    return (Config.instance ??= new Config())
  }
}

// Prefer this in TS: a module-scoped object is already a singleton.
export const config = { values: {} as Record<string, string> }
```

**Pitfalls**: using it as a global variable; no reset hook for tests; lazy initialization races in multithreaded code. **Prefer DI or a module-level object** unless the single-instance constraint is real.

---

## Choosing between creational patterns

| Need | Pattern |
|------|---------|
| Defer concrete class choice to subclasses | Factory Method |
| Create families of related objects | Abstract Factory |
| Build a complex object step by step | Builder |
| Clone an existing object | Prototype |
| Exactly one shared instance | Singleton (last resort) |
| One product chosen by a value | Factory function / registry (no pattern) |

> Rule of thumb: start with a **factory function**; introduce Factory Method/Abstract Factory only when creation must vary independently or families must stay consistent. Start with a **constructor**; introduce Builder only when the parameter list or assembly order becomes a real problem.
