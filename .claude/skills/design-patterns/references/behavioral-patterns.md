# Behavioral Patterns

Patterns that handle efficient communication and the assignment of responsibilities between objects. Source: [refactoring.guru/behavioral-patterns](https://refactoring.guru/design-patterns/behavioral-patterns).

For each pattern: intent, problem, solution, applicability, participants, trade-offs, relations, and Python/TypeScript examples. Use the language primitive first — see `language-idioms.md`.

---

## Chain of Responsibility

- **Intent**: Pass a request along a chain of handlers until one of them handles it.
- **Problem**: A request must be processed by one of several handlers, but the right one is not known up front (validation, approvals, middleware).
- **Solution**: Each `Handler` holds a reference to the next; it either handles the request or passes it along.
- **Use when**: Multiple objects can handle a request, the handler is determined at runtime, or you want to compose processing steps.
- **Participants**: `Handler`, `ConcreteHandler`, `Client`.
- **Trade-offs**: + decouples sender from receiver; + flexible ordering. − a request may go unhandled; − harder to debug the chain; − ordering matters.
- **Relations**: **Decorator** always delegates and adds behavior; **Chain** may stop. **Command** encapsulates the request; **Mediator** centralizes communication.

```python
class Handler(ABC):
    def __init__(self, next: "Handler | None" = None) -> None: self.next = next
    def handle(self, amount: int) -> str | None:
        if self.next: return self.next.handle(amount)
        return None

class Manager(Handler):
    def handle(self, amount: int) -> str | None:
        if amount <= 1_000: return "manager approved"
        return super().handle(amount)

class Director(Handler):
    def handle(self, amount: int) -> str | None:
        if amount <= 10_000: return "director approved"
        return super().handle(amount)
```

```ts
abstract class Handler {
  constructor(protected next?: Handler) {}
  handle(amount: number): string | null {
    return this.next ? this.next.handle(amount) : null
  }
}
class Manager extends Handler {
  handle(a: number) { return a <= 1000 ? "manager approved" : super.handle(a) }
}
class Director extends Handler {
  handle(a: number) { return a <= 10000 ? "director approved" : super.handle(a) }
}
```

**Pitfalls**: no terminal handler; handlers that silently swallow errors; chain order hidden in configuration.

---

## Command

- **Intent**: Turn a request into a standalone object containing all the request's information.
- **Problem**: You need to queue, log, retry, or undo operations, or decouple the sender from the receiver.
- **Solution**: A `Command` interface with `execute()` (and optionally `undo()`); concrete commands bind a receiver to an action; an `Invoker` triggers them.
- **Use when**: Undo/redo, transactional behavior, job queues, macro recording, or deferred execution.
- **Participants**: `Command`, `ConcreteCommand`, `Receiver`, `Invoker`, `Client`.
- **Trade-offs**: + decouples sender/receiver; + supports undo, queues, logging. − many small classes; − undo requires storing enough state.
- **Relations**: **Memento** stores state for undo; **Strategy** swaps algorithms; **Chain** passes a request, Command encapsulates it.

```python
class Command(ABC):
    @abstractmethod
    def execute(self) -> None: ...
    @abstractmethod
    def undo(self) -> None: ...

class TextEditor:
    def __init__(self) -> None: self.text = ""

class InsertText(Command):
    def __init__(self, editor: TextEditor, value: str) -> None:
        self.editor, self.value = editor, value
    def execute(self) -> None: self.editor.text += self.value
    def undo(self) -> None: self.editor.text = self.editor.text[: -len(self.value)]

editor = TextEditor()
cmd = InsertText(editor, "hello")
cmd.execute(); cmd.undo()
```

```ts
interface Command { execute(): void; undo(): void }
class TextEditor { text = "" }

class InsertText implements Command {
  constructor(private editor: TextEditor, private value: string) {}
  execute() { this.editor.text += this.value }
  undo() { this.editor.text = this.editor.text.slice(0, -this.value.length) }
}
```

**Pitfalls**: commands that mutate shared state without a clear receiver; undo that does not restore all side effects; using Command when a simple callback suffices.

---

## Iterator

- **Intent**: Traverse a collection without exposing its underlying representation.
- **Problem**: Different collection types (list, tree, graph) need traversal; clients should not depend on internals.
- **Solution**: An `Iterator` exposes `next()`/`has_next()`; the collection returns an iterator.
- **Use when**: You need uniform traversal, multiple traversal strategies, or lazy iteration over a large structure.
- **Participants**: `Iterator`, `ConcreteIterator`, `Aggregate`, `ConcreteAggregate`.
- **Trade-offs**: + encapsulation; + multiple simultaneous traversals; + lazy evaluation. − extra objects; − stateful iterators can be invalidated by mutation.
- **Relations**: **Composite** often needs an iterator; **Visitor** separates traversal from operations; **Factory Method** can create iterators.
- **Language note**: Most languages provide this natively. Use `__iter__`/`__next__` (Python), `Symbol.iterator` (TS/JS), `range`/channels (Go), `Iterator`/`IntoIterator` (Rust). Do not re-implement.

```python
class Tree:
    def __init__(self, value: int, children: list["Tree"] | None = None) -> None:
        self.value, self.children = value, children or []

    def __iter__(self):
        yield self.value
        for child in self.children:
            yield from child   # depth-first

root = Tree(1, [Tree(2), Tree(3)])
assert list(root) == [1, 2, 3]
```

```ts
class Tree implements Iterable<number> {
  constructor(public value: number, public children: Tree[] = []) {}
  *[Symbol.iterator](): Iterator<number> {
    yield this.value
    for (const child of this.children) yield* child
  }
}
```

**Pitfalls**: exposing internal indices; invalidating iterators on mutation without detection; implementing a custom iterator when the language already has one.

---

## Mediator

- **Intent**: Reduce chaotic dependencies between objects by routing all communication through a central mediator.
- **Problem**: Many components talk directly to each other, producing a tangled web of dependencies.
- **Solution**: A `Mediator` coordinates interactions; components only know the mediator, not each other.
- **Use when**: Many-to-many communication is getting out of hand (form widgets, chat rooms, orchestration).
- **Participants**: `Mediator`, `ConcreteMediator`, `Colleague` classes.
- **Trade-offs**: + reduces coupling; + centralizes coordination. − the mediator can become a god object; − adds an indirection.
- **Relations**: **Observer** is one-to-many notification; **Mediator** is many-to-many coordination. **Facade** simplifies a subsystem but does not coordinate peers.

```python
class ChatRoom:
    def __init__(self) -> None: self.users: list["User"] = []
    def join(self, user: "User") -> None: self.users.append(user)
    def broadcast(self, sender: "User", msg: str) -> None:
        for user in self.users:
            if user is not sender: user.receive(msg)

class User:
    def __init__(self, name: str, room: ChatRoom) -> None:
        self.name, self.room = name, room
        room.join(self)
    def send(self, msg: str) -> None: self.room.broadcast(self, msg)
    def receive(self, msg: str) -> None: print(f"{self.name}: {msg}")
```

```ts
class ChatRoom {
  users: User[] = []
  join(u: User) { this.users.push(u) }
  broadcast(sender: User, msg: string) {
    for (const u of this.users) if (u !== sender) u.receive(msg)
  }
}
class User {
  constructor(public name: string, private room: ChatRoom) { room.join(this) }
  send(msg: string) { this.room.broadcast(this, msg) }
  receive(msg: string) { console.log(`${this.name}: ${msg}`) }
}
```

**Pitfalls**: mediator that knows every detail of every colleague; using it when components only need one-way events; growing the mediator into a god object.

---

## Memento

- **Intent**: Capture and restore an object's internal state without violating encapsulation.
- **Problem**: You need snapshots/undo, but exposing internal state would break encapsulation.
- **Solution**: An `Originator` creates a `Memento` (opaque snapshot); a `Caretaker` stores/restores them.
- **Use when**: Undo/rollback, checkpoints, or transactional state restoration.
- **Participants**: `Originator`, `Memento`, `Caretaker`.
- **Trade-offs**: + preserves encapsulation; + simple undo. − memory cost of snapshots; − deep copies can be expensive; − mementos can become stale.
- **Relations**: **Command** uses Memento for undo; **Prototype** can create snapshots; **Iterator** can restore traversal state.

```python
from dataclasses import dataclass

@dataclass(frozen=True)
class Snapshot:
    text: str

class Editor:
    def __init__(self) -> None: self.text = ""
    def save(self) -> Snapshot: return Snapshot(self.text)
    def restore(self, s: Snapshot) -> None: self.text = s.text

editor = Editor()
editor.text = "draft"
saved = editor.save()
editor.text = "edited"
editor.restore(saved)
assert editor.text == "draft"
```

```ts
class Snapshot { constructor(readonly text: string) {} }
class Editor {
  text = ""
  save() { return new Snapshot(this.text) }
  restore(s: Snapshot) { this.text = s.text }
}
```

**Pitfalls**: mementos that expose mutable internals; unbounded history; snapshotting resources that cannot be restored (connections, locks).

---

## Observer

- **Intent**: Define a one-to-many dependency so that when one object changes state, all its dependents are notified automatically.
- **Problem**: Many objects need to react to another object's changes without tight coupling.
- **Solution**: A `Publisher` maintains a list of `Subscriber`s and notifies them; subscribers register/unregister.
- **Use when**: Event-driven updates (pub/sub, UI reacting to model changes, domain events).
- **Participants**: `Publisher`/`Subject`, `Subscriber`/`Observer`, `ConcretePublisher`, `ConcreteSubscriber`.
- **Trade-offs**: + loose coupling; + dynamic subscription. − notification order is unspecified; − leaks if unsubscribe is forgotten; − cascading updates are hard to trace.
- **Relations**: **Mediator** centralizes many-to-many; **Observer** is one-to-many. **Event bus** is a decoupled Observer variant.
- **Language note**: Prefer built-in event systems (`EventEmitter`, signals, DOM events) when they fit.

```python
class Publisher:
    def __init__(self) -> None: self._subs: list = []
    def subscribe(self, fn) -> None: self._subs.append(fn)
    def unsubscribe(self, fn) -> None: self._subs.remove(fn)
    def notify(self, event: str) -> None:
        for fn in list(self._subs): fn(event)

pub = Publisher()
seen: list[str] = []
pub.subscribe(seen.append)
pub.notify("order.created")
assert seen == ["order.created"]
```

```ts
type Listener = (event: string) => void
class Publisher {
  private subs = new Set<Listener>()
  subscribe(fn: Listener) { this.subs.add(fn); return () => this.subs.delete(fn) }
  notify(event: string) { for (const fn of [...this.subs]) fn(event) }
}
```

**Pitfalls**: forgetting to unsubscribe (memory leaks); subscribers that throw and break the chain; assuming notification order; using Observer for request/response.

---

## State

- **Intent**: Let an object alter its behavior when its internal state changes (it appears to change class).
- **Problem**: An object's behavior is full of `if`/`switch` on its state, and transitions are scattered.
- **Solution**: A `Context` holds a `State` object; each concrete state implements behavior for that state and may transition the context.
- **Use when**: An object has many states and state-specific behavior (Draft→Submitted→Approved), and transitions are complex.
- **Participants**: `Context`, `State`, `ConcreteState`.
- **Trade-offs**: + localizes state behavior; + explicit transitions. − more classes; − overkill for two states; − state objects may need context access.
- **Relations**: **Strategy** is chosen by the client and usually stateless; **State** transitions itself and is stateful. **Template Method** fixes an algorithm; State changes behavior by state.

```python
class Order:
    def __init__(self) -> None: self.state: "OrderState" = Draft()
    def next(self) -> str: return self.state.next(self)

class OrderState(ABC):
    @abstractmethod
    def next(self, order: Order) -> str: ...

class Draft(OrderState):
    def next(self, order: Order) -> str:
        order.state = Submitted()
        return "submitted"

class Submitted(OrderState):
    def next(self, order: Order) -> str:
        order.state = Approved()
        return "approved"

class Approved(OrderState):
    def next(self, order: Order) -> str: return "already approved"
```

```ts
interface OrderState { next(order: Order): string }
class Order {
  state: OrderState = new Draft()
  next() { return this.state.next(this) }
}
class Draft implements OrderState {
  next(o: Order) { o.state = new Submitted(); return "submitted" }
}
```

**Pitfalls**: state objects that mutate context internals directly; transitions that are not validated; using State for a simple boolean flag.

---

## Strategy

- **Intent**: Define a family of algorithms, encapsulate each one, and make them interchangeable.
- **Problem**: You need to swap algorithms at runtime (sorting, payment, compression) without hardcoding.
- **Solution**: A `Context` delegates to a `Strategy` interface; concrete strategies implement the algorithm.
- **Use when**: Multiple variants of an algorithm exist, or a class has many conditional branches that differ only in behavior.
- **Participants**: `Context`, `Strategy`, `ConcreteStrategy`.
- **Trade-offs**: + open/closed for algorithms; + testable in isolation. − clients must choose a strategy; − extra objects when a function would do.
- **Relations**: **State** changes behavior by internal state; **Strategy** is chosen externally. **Template Method** varies steps via inheritance; Strategy via composition.
- **Language note**: In Python/TS/Go/Rust, a function value is often the best Strategy.

```python
class PaymentStrategy(Protocol):
    def pay(self, amount: int) -> str: ...

class CardPayment:
    def pay(self, amount: int) -> str: return f"card {amount}"

class CashPayment:
    def pay(self, amount: int) -> str: return f"cash {amount}"

class Checkout:
    def __init__(self, strategy: PaymentStrategy) -> None: self.strategy = strategy
    def pay(self, amount: int) -> str: return self.strategy.pay(amount)

checkout = Checkout(CardPayment())
```

```ts
interface PaymentStrategy { pay(amount: number): string }
class Checkout {
  constructor(private strategy: PaymentStrategy) {}
  pay(amount: number) { return this.strategy.pay(amount) }
}
const card: PaymentStrategy = { pay: (a) => `card ${a}` }
```

**Pitfalls**: strategy with only one implementation; strategy exposing context state; choosing a class hierarchy when a function suffices.

---

## Template Method

- **Intent**: Define the skeleton of an algorithm, deferring some steps to subclasses.
- **Problem**: Several classes share an algorithm's structure but differ in specific steps.
- **Solution**: A base class defines a template method that calls abstract/overridable step methods; subclasses override the steps.
- **Use when**: The algorithm order is fixed but steps vary; you want to avoid duplicating the skeleton.
- **Participants**: `AbstractClass`, `ConcreteClass`.
- **Trade-offs**: + removes duplication; + enforces order. − inheritance coupling; − hard to change the skeleton; − can violate LSP if subclasses cannot implement a step.
- **Relations**: **Strategy** uses composition and swaps the whole algorithm; **Template Method** uses inheritance and varies steps. **Factory Method** is often a step inside it.

```python
class Report(ABC):
    def generate(self) -> str:            # template method
        return f"{self.header()}\n{self.body()}\n{self.footer()}"

    @abstractmethod
    def header(self) -> str: ...
    @abstractmethod
    def body(self) -> str: ...
    def footer(self) -> str: return "-- end --"

class SalesReport(Report):
    def header(self) -> str: return "Sales"
    def body(self) -> str: return "42 orders"
```

```ts
abstract class Report {
  generate() { return `${this.header()}\n${this.body()}\n${this.footer()}` }
  protected abstract header(): string
  protected abstract body(): string
  protected footer() { return "-- end --" }
}
class SalesReport extends Report {
  protected header() { return "Sales" }
  protected body() { return "42 orders" }
}
```

**Pitfalls**: template method calling overridable methods in a constructor; subclasses that must no-op a step (LSP smell); using inheritance when a strategy/composition would be clearer.

---

## Visitor

- **Intent**: Separate algorithms from the objects they operate on; add operations without changing element classes.
- **Problem**: You need to add many unrelated operations to a stable set of classes without polluting them.
- **Solution**: A `Visitor` declares a `visit()` per element type; elements accept a visitor (`element.accept(visitor)` → `visitor.visit(element)`).
- **Use when**: The class hierarchy is stable and you frequently add operations (exporters, analyzers, renderers).
- **Participants**: `Visitor`, `ConcreteVisitor`, `Element`, `ConcreteElement`, `ObjectStructure`.
- **Trade-offs**: + open/closed for operations; + groups related behavior. − adding a new element type means changing every visitor; − double dispatch is unfamiliar; − can break encapsulation.
- **Relations**: **Iterator** traverses; **Visitor** operates. **Composite** is a common object structure for Visitor. **Strategy** swaps one algorithm; Visitor adds many operations.

```python
class Shape(ABC):
    @abstractmethod
    def accept(self, visitor: "ShapeVisitor") -> str: ...

class Circle(Shape):
    def __init__(self, r: float) -> None: self.r = r
    def accept(self, visitor: "ShapeVisitor") -> str: return visitor.visit_circle(self)

class Square(Shape):
    def __init__(self, side: float) -> None: self.side = side
    def accept(self, visitor: "ShapeVisitor") -> str: return visitor.visit_square(self)

class AreaVisitor:
    def visit_circle(self, c: Circle) -> str: return f"circle area {3.14 * c.r ** 2}"
    def visit_square(self, s: Square) -> str: return f"square area {s.side ** 2}"
```

```ts
interface Shape { accept(v: ShapeVisitor): string }
class Circle implements Shape {
  constructor(readonly r: number) {}
  accept(v: ShapeVisitor) { return v.visitCircle(this) }
}
class Square implements Shape {
  constructor(readonly side: number) {}
  accept(v: ShapeVisitor) { return v.visitSquare(this) }
}
interface ShapeVisitor { visitCircle(c: Circle): string; visitSquare(s: Square): string }
```

**Pitfalls**: unstable element hierarchy (every new type touches all visitors); visitors that need private state; using Visitor when a simple function over a discriminated union works (TS) or `match` (Python 3.10+).

---

## Choosing between behavioral patterns

| Need | Pattern |
|------|---------|
| Handler determined at runtime along a chain | Chain of Responsibility |
| Queue/undo/log operations as objects | Command |
| Traverse a collection | Iterator |
| Centralize many-to-many communication | Mediator |
| Snapshot/restore state | Memento |
| Notify dependents of change | Observer |
| Behavior depends on internal state | State |
| Swap algorithms at runtime | Strategy |
| Fixed skeleton, variable steps | Template Method |
| Add operations to a stable hierarchy | Visitor |

> Strategy vs State vs Template Method: all change behavior, but Strategy swaps a whole algorithm (client-chosen), State changes by internal state, Template Method fixes the skeleton with overridable steps.
