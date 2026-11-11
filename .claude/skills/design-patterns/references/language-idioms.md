# Language Idioms

Patterns are intents, not class diagrams. Before translating a GoF structure literally, check whether the language already provides the mechanism. This file lists the built-in equivalents and the idiomatic variants per language.

## Pattern → idiom at a glance

| Pattern | Python | TypeScript/JS | Java | C# | Go | Rust |
|---------|--------|---------------|------|----|----|------|
| **Iterator** | `__iter__`/`__next__` | `Symbol.iterator` | `Iterator`/`Iterable` | `IEnumerable`/`yield` | `range`/channels | `Iterator`/`IntoIterator` |
| **Strategy** | function / `Protocol` | function / interface | functional interface | delegate / interface | function / interface | closure / trait |
| **Decorator** | `@decorator`, wrapper | higher-order function, middleware | interface wrapper | delegate/middleware | interface wrapper | trait wrapper |
| **Singleton** | module object | module object | enum / DI | DI container | package var / `sync.Once` | `OnceLock` / module |
| **Observer** | callbacks / signals | `EventTarget`, EventEmitter | listeners / `Flow` | `event`/`IObservable` | channels | channels / `tokio::sync` |
| **Command** | closure / dataclass | closure / object | lambda / class | `Action`/class | func value | closure / enum |
| **State** | `Enum` + `match` | discriminated union | sealed + switch | enum + switch | interface + struct | enum + `match` |
| **Visitor** | `functools.singledispatch` / `match` | discriminated union + exhaustive switch | sealed + pattern matching | pattern matching / switch | interface | enum + `match` |
| **Adapter** | wrapper / `Protocol` | wrapper | wrapper | wrapper | interface implementation | trait impl |
| **Builder** | keyword args / fluent class | object literal / fluent | builder class | object initializer / builder | functional options | builder / struct update |
| **Proxy** | `__getattr__`, descriptor | built-in `Proxy` | dynamic proxy | `DispatchProxy` | interface wrapper | `Deref` / wrapper |
| **Composite** | recursive dataclass | recursive type | recursive class | recursive class | recursive struct | recursive enum |

## Python

- **Iterator**: implement `__iter__` returning a generator; do not write a GoF iterator class.
- **Strategy**: a function or `functools.partial` is usually enough. Use `typing.Protocol` for a named contract.
- **Decorator**: `@decorator` wraps callables; a class wrapper is for object decoration.
- **Singleton**: a module-level object is already a singleton. Prefer dependency injection for testability.
- **Observer**: callbacks, `blinker`, Django signals, or a small event bus.
- **State/Visitor**: `Enum` + `match` (3.10+) or `functools.singledispatch` can replace a class hierarchy.
- **Builder**: keyword arguments and dataclasses replace most builders; use a fluent class for query DSLs.
- **Proxy**: `__getattr__`, descriptors, or `functools.cached_property`.
- **Adapter**: a small wrapper class; do not subclass the third-party type.

## TypeScript / JavaScript

- **Strategy**: a function value or an object with methods. Do not create a class per algorithm unless it has state.
- **Decorator**: higher-order functions, middleware, or the built-in `Proxy` for interception.
- **Observer**: `EventTarget`, `EventEmitter`, or a typed callback set. Always return an unsubscribe function.
- **Iterator**: `Symbol.iterator` and generators; `for...of` is the client API.
- **State/Visitor**: a discriminated union plus an exhaustive `switch` (use `never` to enforce exhaustiveness) is often clearer than classes.
- **Singleton**: a module-scoped object; avoid `static instance` unless lazy init is required.
- **Builder**: object literals with a typed config; use a fluent builder only for query/DSL APIs.
- **Adapter**: a wrapper object implementing your interface; keep the SDK type out of public signatures.
- **Proxy**: the built-in `Proxy` is for dynamic interception; a class proxy is for typed access control.

## Java

- **Strategy**: a functional interface + lambdas; a class only when the strategy has state.
- **Observer**: `java.util.concurrent.Flow`, reactive streams, or an explicit listener list. `java.util.Observer` is deprecated.
- **Iterator**: `Iterable`/`Iterator`; use streams for transformations.
- **State/Visitor**: sealed interfaces + pattern matching (Java 17+) reduce boilerplate.
- **Builder**: a static nested builder for many optional fields; records for immutable value objects.
- **Singleton**: enum singleton or a DI container; avoid lazy double-checked locking unless measured.
- **Adapter**: a wrapper implementing the target interface; do not extend the adaptee unless it is designed for it.
- **Proxy**: JDK dynamic proxies or bytecode proxies; keep the interface narrow.

## C#

- **Strategy**: a delegate or an interface; `Func<T>` covers most cases.
- **Observer**: `event`, `IObservable<T>`, or `Channel<T>` for async pipelines.
- **Iterator**: `IEnumerable<T>` + `yield return`.
- **State/Visitor**: switch expressions + pattern matching; sealed hierarchies for exhaustiveness.
- **Builder**: object initializers or a fluent builder for complex objects.
- **Singleton**: DI container (`AddSingleton`); avoid static mutable state.
- **Adapter**: wrapper class implementing the target interface.
- **Proxy**: `DispatchProxy` or a wrapper; `Lazy<T>` for lazy initialization.

## Go

- **Strategy**: a function type or a small interface; do not build a class hierarchy.
- **Decorator**: an interface wrapper (e.g., `http.Handler` middleware).
- **Observer**: channels or callbacks; close channels carefully and define ownership.
- **Iterator**: `range` over slices/maps; Go 1.23+ `iter.Seq` for custom sequences.
- **Builder**: functional options (`WithTimeout`, `WithLogger`) are the idiomatic builder.
- **Singleton**: package-level variable + `sync.Once` if lazy; prefer explicit wiring.
- **Adapter**: an interface implementation around a third-party client.
- **Proxy**: an interface wrapper; no dynamic proxies.
- **Composite**: recursive struct with a slice of an interface.

## Rust

- **Strategy**: a closure (`Box<dyn Fn>`) or a trait object; generics for static dispatch.
- **Decorator**: a trait wrapper; avoid `Deref` tricks unless the semantics are clear.
- **Observer**: channels (`std::sync::mpsc`, `tokio::sync`) or a subscriber list with `Rc<RefCell<...>>`.
- **Iterator**: `Iterator`/`IntoIterator`; do not write a manual iterator.
- **State/Visitor**: an `enum` + `match` is usually better than a trait-object hierarchy.
- **Builder**: a builder struct with `self`-returning methods; `..Default::default()` for optional fields.
- **Singleton**: `OnceLock`/`LazyLock` or a module-level value; avoid global mutable state.
- **Adapter**: a newtype implementing your trait around a foreign type.
- **Proxy**: a wrapper implementing the same trait; `Deref` only for smart-pointer semantics.

## Patterns that change shape

| Pattern | Why it changes | Idiomatic form |
|---------|----------------|----------------|
| **Singleton** | Global state is hard to test | DI / module object / `OnceLock` |
| **Iterator** | Built into the language | Protocol/`Iterator` trait |
| **Strategy** | Functions are first-class | Function value / lambda |
| **Command** | Closures capture the request | Closure / function queue |
| **Decorator** | Higher-order functions | Middleware / wrapper function |
| **Visitor** | Pattern matching over closed types | `match` / exhaustive switch |
| **Builder** | Named/keyword arguments | Config object / functional options |
| **Factory Method** | Constructors + functions are enough | Factory function / registry |
| **Abstract Factory** | DI container + config | Provider module / factory function |

## Anti-patterns when translating

- **Class-per-algorithm in Python/TS/Go/Rust**: a function is usually enough.
- **Singleton as a global variable**: hidden state, hard tests; inject instead.
- **GoF Iterator class**: use the language's iteration protocol.
- **Visitor hierarchy for a closed union**: use `match`/`switch` with exhaustiveness.
- **Builder for a 2-field object**: use a constructor or object literal.
- **Deep inheritance in Go/Rust**: these languages favor composition; do not emulate Java.
- **Java-style interfaces everywhere in Python**: use `Protocol` for typing, not for ceremony.

## Rule of thumb

1. Use the language primitive.
2. If the primitive cannot express the intent, use the smallest pattern.
3. If the pattern needs more than three participants, check whether the problem is actually a different pattern.
