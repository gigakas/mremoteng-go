# Architecture Tests

Architecture tests are **executable design rules**. Unit tests verify behavior; architecture tests verify structure: dependency direction, layers, module boundaries and cycles. They fail the build when code drifts from the intended design.

## What to enforce

Start with the rules that reflect a real pain in the project:

- **Dependency direction**: domain must not import infrastructure, UI, ORM or HTTP SDKs.
- **Ports and adapters**: adapters depend inward; the domain defines the ports.
- **No cycles** between modules/packages.
- **Layer boundaries**: API → application → domain; never the reverse.
- **Bounded contexts**: no direct imports between contexts; communicate through events/ports.
- **Shared kernel**: shared code must not depend on feature code.
- **Forbidden APIs**: no raw SQL outside repositories, no `requests` in domain, no `eval`.

Do not start with ten rules. Pick the **three** that prevent the most likely regression.

## Python — import-linter

```ini
# .importlinter
[importlinter]
root_package = myapp

[importlinter:contract:layers]
name = Layered architecture
type = layers
layers =
    myapp.api
    myapp.application
    myapp.domain

[importlinter:contract:domain-isolation]
name = Domain must not depend on infrastructure
type = forbidden
source_modules =
    myapp.domain
forbidden_modules =
    myapp.infrastructure
    sqlalchemy
    requests
```

Run in CI:

```bash
lint-imports
```

## TypeScript / JavaScript — dependency-cruiser

```js
// .dependency-cruiser.js
module.exports = {
  forbidden: [
    {
      name: "domain-no-infra",
      severity: "error",
      from: { path: "^src/domain" },
      to: { path: "^src/(infrastructure|ui|api)" },
    },
    {
      name: "no-circular",
      severity: "error",
      from: {},
      to: { circular: true },
    },
  ],
  options: {
    doNotFollow: { path: "node_modules" },
    tsConfig: { fileName: "tsconfig.json" },
  },
}
```

```bash
npx depcruise src --config .dependency-cruiser.js
```

## Java — ArchUnit

```java
@AnalyzeClasses(packages = "com.example")
class ArchitectureTest {

  @ArchTest
  static final ArchRule domain_does_not_depend_on_infrastructure =
      noClasses()
          .that().resideInAPackage("..domain..")
          .should().dependOnClassesThat()
          .resideInAPackage("..infrastructure..");

  @ArchTest
  static final ArchRule no_cycles =
      slices().matching("com.example.(*)..").should().beFreeOfCycles();
}
```

## C# — NetArchTest

```csharp
[Fact]
public void Domain_should_not_depend_on_infrastructure()
{
    var result = Types.InAssembly(typeof(Order).Assembly)
        .That().ResideInNamespace("MyApp.Domain")
        .ShouldNot().HaveDependencyOn("MyApp.Infrastructure")
        .GetResult();

    Assert.True(result.IsSuccessful);
}
```

## Go — go-arch-lint

```yaml
# .go-arch-lint.yml
version: 3
module: github.com/example/myapp

components:
  domain:
    in: internal/domain
  infrastructure:
    in: internal/infrastructure

commonComponents:
  - domain

deps:
  infrastructure:
    mayDependOn:
      - domain
  domain:
    mayNotDependOn:
      - infrastructure
```

```bash
go-arch-lint check
```

## Rust — crate and module boundaries

Rust does not have a universal architecture-test tool; enforce boundaries with:

- separate crates for `domain` and `infrastructure`;
- `pub(crate)` visibility instead of `pub`;
- a CI check that the domain crate does not depend on infrastructure crates:

```bash
cargo tree -p domain | grep infrastructure && exit 1 || true
```

- `cargo deny` for dependency policy.

## Where to run them

- **Unit layer**: architecture tests are fast; run them with the unit suite.
- **PR gate**: fail on violations.
- **Nightly**: full graph checks for large monorepos.
- **Local**: pre-commit or the agent's `/audit` command.

## Writing a good rule

1. Name it after the intent: `domain-no-infra`, not `rule-3`.
2. Make the failure message actionable: "Domain must not import infrastructure; use a port."
3. Keep the allowed list narrow; do not whitelist everything.
4. Review rules like code; delete rules that no longer reflect the design.
5. Add a rule only after a real violation or a real risk.

## Guardrails

- **Do not overconstrain**: a rule that blocks legitimate work will be disabled.
- **Do not encode preferences as architecture**: naming/style belongs to lint.
- **Do not whitelist silently**: document exceptions with an owner and a reason.
- **Keep architecture tests fast**; they should not need a database or network.
- **Pair with `code-audit`**: an architecture violation is a `High`/`Critical` suggestion.

## Checklist

- [ ] Three rules that reflect real pain.
- [ ] Domain/infrastructure direction enforced.
- [ ] No cycles.
- [ ] Layer boundaries enforced.
- [ ] Failure messages are actionable.
- [ ] Runs in the fast suite/PR.
- [ ] Exceptions documented.
- [ ] Linked from the architecture doc/README.
