# Structural Patterns

Patterns that explain how to assemble objects and classes into larger structures while keeping them flexible and efficient. Source: [refactoring.guru/structural-patterns](https://refactoring.guru/design-patterns/structural-patterns).

For each pattern: intent, problem, solution, applicability, participants, trade-offs, relations, and Python/TypeScript examples. Use the language primitive first — see `language-idioms.md`.

---

## Adapter

- **Intent**: Let objects with incompatible interfaces work together.
- **Problem**: You need an existing class (third-party, legacy) but its interface does not match what your code expects.
- **Solution**: A wrapper (`Adapter`) implements the target interface and delegates to the wrapped object (composition), translating calls and data.
- **Use when**: Integrating a library or legacy code you cannot change; you want to isolate a dependency behind your own port.
- **Participants**: `Target`, `Adaptee`, `Adapter`, `Client`.
- **Trade-offs**: + reuses existing code; + isolates third-party details. − extra layer; − can mask a bad interface instead of fixing it.
- **Relations**: **Bridge** is designed up front to separate dimensions; **Decorator** adds behavior without changing the interface; **Proxy** keeps the same interface; **Facade** simplifies a whole subsystem.

```python
class RoundHole:
    def __init__(self, radius: float) -> None: self.radius = radius
    def fits(self, peg_radius: float) -> bool: return peg_radius <= self.radius

class SquarePeg:
    def __init__(self, width: float) -> None: self.width = width

class SquarePegAdapter:
    def __init__(self, peg: SquarePeg) -> None: self.peg = peg
    @property
    def radius(self) -> float:
        return self.peg.width * (2 ** 0.5) / 2   # circumscribed radius

hole = RoundHole(5)
assert hole.fits(SquarePegAdapter(SquarePeg(7)).radius)
```

```ts
interface Peg { radius(): number }

class SquarePeg {
  constructor(private width: number) {}
  get widthValue() { return this.width }
}

class SquarePegAdapter implements Peg {
  constructor(private peg: SquarePeg) {}
  radius() { return this.peg.widthValue * Math.SQRT2 / 2 }
}
```

**Pitfalls**: putting business logic in the adapter; adapting too many methods (the adapter becomes a second implementation); not handling units/error-code differences explicitly.

---

## Bridge

- **Intent**: Separate an abstraction from its implementation so both can vary independently.
- **Problem**: Two orthogonal dimensions (shape × color, device × OS, message × channel) combined via inheritance explode into a class per combination.
- **Solution**: The abstraction holds a reference to an implementation interface; concrete implementations are swapped at runtime. Two independent hierarchies.
- **Use when**: You have two or more dimensions of variation and want to avoid a class explosion; you want to change implementation at runtime.
- **Participants**: `Abstraction`, `RefinedAbstraction`, `Implementor`, `ConcreteImplementor`.
- **Trade-offs**: + avoids combinatorial subclasses; + independent evolution. − more indirection; − overkill when only one dimension varies.
- **Relations**: **Strategy** swaps an algorithm inside one object; **Bridge** separates two hierarchies that both vary. **Adapter** is applied after the fact; Bridge is designed up front.

```python
class Renderer(ABC):
    @abstractmethod
    def draw(self, shape: str) -> str: ...

class VectorRenderer(Renderer):
    def draw(self, shape: str) -> str: return f"vector {shape}"

class RasterRenderer(Renderer):
    def draw(self, shape: str) -> str: return f"raster {shape}"

class Shape(ABC):
    def __init__(self, renderer: Renderer) -> None: self.renderer = renderer
    @abstractmethod
    def draw(self) -> str: ...

class Circle(Shape):
    def draw(self) -> str: return self.renderer.draw("circle")
```

```ts
interface Renderer { draw(shape: string): string }
class VectorRenderer implements Renderer { draw(s: string) { return `vector ${s}` } }
class RasterRenderer implements Renderer { draw(s: string) { return `raster ${s}` } }

abstract class Shape {
  constructor(protected renderer: Renderer) {}
  abstract draw(): string
}
class Circle extends Shape { draw() { return this.renderer.draw("circle") } }
```

**Pitfalls**: creating a bridge when only one dimension varies; leaking implementation details through the abstraction.

---

## Composite

- **Intent**: Compose objects into tree structures to represent part-whole hierarchies; clients treat individual objects and compositions uniformly.
- **Problem**: Code must handle a single object and a group of objects the same way (file/folder, menu/item, category/product).
- **Solution**: A common `Component` interface; `Leaf` implements behavior, `Composite` holds children and delegates/recurses.
- **Use when**: The domain is a tree and clients should not care whether a node is a leaf or a branch.
- **Participants**: `Component`, `Leaf`, `Composite`, `Client`.
- **Trade-offs**: + uniform treatment; + easy recursive operations. − can make the design overly general; − hard to restrict which children are valid.
- **Relations**: **Decorator** wraps one object and adds behavior; **Composite** aggregates many. **Iterator** is often used to traverse the tree; **Visitor** to add operations.

```python
class Node(ABC):
    @abstractmethod
    def size(self) -> int: ...

class File(Node):
    def __init__(self, size: int) -> None: self._size = size
    def size(self) -> int: return self._size

class Folder(Node):
    def __init__(self) -> None: self.children: list[Node] = []
    def add(self, node: Node) -> None: self.children.append(node)
    def size(self) -> int: return sum(c.size() for c in self.children)

root = Folder(); root.add(File(10)); sub = Folder(); sub.add(File(5)); root.add(sub)
assert root.size() == 15
```

```ts
interface Node { size(): number }
class FileNode implements Node { constructor(private n: number) {} size() { return this.n } }
class Folder implements Node {
  private children: Node[] = []
  add(n: Node) { this.children.push(n) }
  size() { return this.children.reduce((s, c) => s + c.size(), 0) }
}
```

**Pitfalls**: using Composite when there is no tree; letting leaves expose child-management methods; unbounded recursion without cycle detection.

---

## Decorator

- **Intent**: Attach additional responsibilities to an object dynamically, without modifying its class or creating subclasses.
- **Problem**: You need optional behaviors (logging, caching, retry, compression) in many combinations — subclassing explodes.
- **Solution**: A `Decorator` implements the same interface as the component, holds a reference to it, and adds behavior before/after delegating.
- **Use when**: Responsibilities should be addable/removable at runtime; you want to compose behaviors without a rigid hierarchy.
- **Participants**: `Component`, `ConcreteComponent`, `Decorator`, `ConcreteDecorator`.
- **Trade-offs**: + flexible composition; + single-responsibility decorators. − many small objects; − stack order matters and can be hard to debug.
- **Relations**: **Proxy** controls access with the same interface; **Adapter** changes the interface; **Chain of Responsibility** passes requests along, while Decorator always delegates and may add behavior.
- **Language note**: Python decorators (`@decorator`) and TS higher-order functions are related but wrap callables, not objects.

```python
class Notifier(ABC):
    @abstractmethod
    def send(self, msg: str) -> str: ...

class EmailNotifier(Notifier):
    def send(self, msg: str) -> str: return f"email: {msg}"

class LoggingNotifier(Notifier):
    def __init__(self, inner: Notifier) -> None: self.inner = inner
    def send(self, msg: str) -> str:
        print(f"log: {msg}")
        return self.inner.send(msg)

notifier = LoggingNotifier(EmailNotifier())
```

```ts
interface Notifier { send(msg: string): string }
class EmailNotifier implements Notifier { send(m: string) { return `email: ${m}` } }

class LoggingNotifier implements Notifier {
  constructor(private inner: Notifier) {}
  send(m: string) { console.log(`log: ${m}`); return this.inner.send(m) }
}
```

**Pitfalls**: decorator stack order that is implicit and untested; decorators that change the interface; wrapping the same object twice by accident.

---

## Facade

- **Intent**: Provide a simplified interface to a complex subsystem.
- **Problem**: Client code is coupled to many subsystem classes and their correct call order.
- **Solution**: A `Facade` exposes a small set of convenient methods and delegates to the subsystem behind it.
- **Use when**: You want a simple entry point over a library or a set of services; you want to decouple clients from subsystem internals.
- **Participants**: `Facade`, subsystem classes, `Client`.
- **Trade-offs**: + reduces coupling and complexity for clients; + centralizes orchestration. − can become a god object; − does not prevent direct subsystem access.
- **Relations**: **Adapter** changes one interface; **Facade** simplifies many. **Mediator** coordinates peer objects; Facade only exposes a simpler API.

```python
class VideoDecoder:
    def load(self, path: str) -> str: return f"decoded {path}"

class AudioMixer:
    def mix(self) -> str: return "mixed audio"

class VideoConverter:   # facade
    def __init__(self) -> None:
        self.decoder = VideoDecoder()
        self.mixer = AudioMixer()
    def convert(self, path: str) -> str:
        return f"{self.decoder.load(path)} + {self.mixer.mix()}"

assert VideoConverter().convert("a.mp4").startswith("decoded")
```

```ts
class VideoDecoder { load(p: string) { return `decoded ${p}` } }
class AudioMixer { mix() { return "mixed audio" } }

class VideoConverter {
  private decoder = new VideoDecoder()
  private mixer = new AudioMixer()
  convert(path: string) { return `${this.decoder.load(path)} + ${this.mixer.mix()}` }
}
```

**Pitfalls**: facade that grows into a god object; facade that hides errors the client must handle; clients still reaching into the subsystem.

---

## Flyweight

- **Intent**: Share common state among many similar objects to save memory.
- **Problem**: A program needs a huge number of near-identical objects (characters, particles, map tiles) and memory is constrained.
- **Solution**: Split state into *intrinsic* (shared, immutable) and *extrinsic* (context-dependent). A factory caches and shares intrinsic state.
- **Use when**: Many objects share substantial identical state and memory is the bottleneck.
- **Participants**: `Flyweight`, `FlyweightFactory`, `Context` (stores extrinsic state).
- **Trade-offs**: + large memory savings; + fewer allocations. − more complex code; − CPU cost to recompute/transfer extrinsic state; − only worth it with real measurements.
- **Relations**: **Singleton** is a factory with one instance; **Flyweight** is a factory with many shared instances. **Composite** often uses flyweights for leaves.

```python
from dataclasses import dataclass

@dataclass(frozen=True)
class TreeType:          # intrinsic, shared
    name: str
    color: str

class TreeFactory:
    _types: dict[str, TreeType] = {}
    @classmethod
    def get(cls, name: str, color: str) -> TreeType:
        key = f"{name}:{color}"
        if key not in cls._types:
            cls._types[key] = TreeType(name, color)
        return cls._types[key]

class Tree:              # extrinsic context
    def __init__(self, x: int, y: int, kind: TreeType) -> None:
        self.x, self.y, self.kind = x, y, kind
```

```ts
class TreeType {
  constructor(readonly name: string, readonly color: string) {}
}
class TreeFactory {
  private static types = new Map<string, TreeType>()
  static get(name: string, color: string) {
    const key = `${name}:${color}`
    if (!this.types.has(key)) this.types.set(key, new TreeType(name, color))
    return this.types.get(key)!
  }
}
```

**Pitfalls**: applying it without measuring memory; mutable flyweight state (breaks sharing); caching without an eviction policy.

---

## Proxy

- **Intent**: Provide a substitute for another object to control access to it.
- **Problem**: You need lazy initialization, access control, logging, caching, or remote access without changing the real object or its clients.
- **Solution**: A `Proxy` implements the same interface as the real subject and controls when/how to call it.
- **Use when**: You need transparent interception of access.
- **Variants**: Virtual (lazy), Protection (access control), Remote (network), Logging/Caching.
- **Participants**: `Subject`, `RealSubject`, `Proxy`, `Client`.
- **Trade-offs**: + transparent control; + can add caching/lazy loading. − extra indirection; − can hide latency or failures; − proxies can diverge from the real object.
- **Relations**: **Decorator** adds behavior; **Proxy** controls access; **Adapter** changes the interface; **Facade** simplifies a subsystem.
- **Language note**: JS `Proxy` and Python `__getattr__`/descriptors are built-in interception mechanisms; use them before writing a class proxy.

```python
class Image:
    def display(self) -> str: return "image"

class LazyImage:
    def __init__(self, path: str) -> None:
        self.path = path
        self._real: Image | None = None
    def display(self) -> str:
        if self._real is None:
            print(f"loading {self.path}")
            self._real = Image()
        return self._real.display()
```

```ts
interface Image { display(): string }
class RealImage implements Image { display() { return "image" } }

class LazyImage implements Image {
  private real?: RealImage
  constructor(private path: string) {}
  display() {
    if (!this.real) { console.log(`loading ${this.path}`); this.real = new RealImage() }
    return this.real.display()
  }
}
```

**Pitfalls**: proxy that silently swallows errors; caching proxy without invalidation; protection proxy that trusts client-provided roles.

---

## Choosing between structural patterns

| Need | Pattern |
|------|---------|
| Reconcile incompatible interfaces | Adapter |
| Two independent dimensions of variation | Bridge |
| Uniform tree/part-whole treatment | Composite |
| Add responsibilities dynamically | Decorator |
| Simple entry point to a subsystem | Facade |
| Save memory with shared state | Flyweight |
| Control/transparent access to an object | Proxy |

> Adapter vs Bridge vs Decorator vs Proxy: all wrap an object, but differ by *intent* — Adapter reconciles interfaces, Bridge separates dimensions, Decorator adds behavior, Proxy controls access.
