# Tooling for Refactoring

Formatters, linters, complexity and duplication tools per language. They make the mechanical part of a refactor cheap; **none of them chooses the refactoring for you**, and none replaces the test safety net.

## What each tool class buys you

| Class | What it catches | When to run |
|---|---|---|
| **Formatter** | Style drift introduced by move/extract/rename | After every step, saved by the editor |
| **Linter** | Unused code, shadowed names, dead branches left behind | After every step, blocking in CI |
| **Type checker** | Broken signatures after Extract/Move/Change Signature | After every step (non-negotiable for typed code) |
| **Complexity** | Methods/classes that still exceed the thresholds | Before and after a pass |
| **Duplication** | Copies created by parallel edits (Extract Class gone wrong) | Before and after a pass |
| **Dead code** | Orphans left by Inline/Move/Remove | Before finishing |
| **Dependency rules** | Layer violations the refactor was supposed to remove | In CI (architecture tests) |

## Per language

| Language | Format | Lint | Types | Complexity | Duplication |
|---|---|---|---|---|---|
| Python | `ruff format` / `black` | `ruff check` / `flake8` | `mypy` / `pyright` | `radon cc` | `pylint --enable=duplicate-code`, `jscpd` |
| TypeScript/JS | `prettier` / `biome format` | `eslint` / `biome lint` | `tsc --noEmit` | `eslint complexity`, `ts-arch` | `jscpd` |
| Dart/Flutter | `dart format` | `dart analyze` (+ `custom_lint`) | sound null safety (compiler) | `dcm` (commercial), `custom_lint` rules | `jscpd` |
| Go | `gofmt` / `goimports` | `golangci-lint` | compiler | `gocyclo`, `gocognit` | `dupl` |
| Rust | `rustfmt` | `clippy` | compiler | `clippy::cognitive_complexity` | `jscpd` |
| Java | `google-java-format` / `spotless` | `checkstyle`, `spotbugs` | compiler | `pmd` (CyclomaticComplexity) | `pmd` (CPD), `jscpd` |
| C# | `dotnet format` | `roslynator`, built-in analyzers | compiler | `dotnet-code-metrics`, CA1502 | `jscpd`, `dupfinder` |
| Ruby | `rubocop -a` | `rubocop` | `sorbet`/`rbs` (optional) | `rubocop` (Metrics/AbcSize) | `jscpd` |
| PHP | `pint` / `php-cs-fixer` | `phpstan`, `psalm` | `phpstan` | `phpmetrics` | `phpcpd`, `jscpd` |

Exact flags drift between versions: check `--help` before scripting a command, and pin versions in CI.

## Complexity thresholds

Use the triggers from `file-responsibility-rules.md`; keep the same thresholds across passes so deltas are comparable.

- **Method**: > 30–50 lines, cyclomatic complexity > 10, or nesting > 3.
- **Class/module**: > 300–500 lines, or more than one reason to change.
- **File**: > 400–600 lines in languages without partial classes.
- **Duplication**: a third copy is the trigger to extract; two copies are often acceptable.

Thresholds are **signals**, not gates. A 60-line method with a flat switch is not the same problem as a 12-line method with four nested conditionals.

## Safe automation rules

1. **One tool at a time.** Run the formatter, then the linter, then the type checker — never as one opaque command while refactoring.
2. **Never auto-fix and commit in the same step.** Auto-fix output is a diff you must read; a "safe" fix (`--fix`, `--unsafe-fixes`, `rubocop -a`) can change behavior.
3. **Type check after every signature change.** Extract/Move/Change Signature breaks callers silently in dynamic languages.
4. **Do not enable new rules mid-refactor.** New findings are a different task; record them in the audit backlog.
5. **Keep the formatter out of behavior diffs.** Format-only commits are reviewed separately so the refactor diff stays readable.

## Automation entry points

```bash
# Python
ruff format . && ruff check . && mypy .
radon cc -s -a src/

# TypeScript / JavaScript
npx prettier --write . && npx eslint . && npx tsc --noEmit

# Dart / Flutter
dart format . && dart analyze --fatal-infos
npx jscpd src/

# Go
gofmt -w . && golangci-lint run
gocyclo -over 10 .

# Rust
cargo fmt && cargo clippy -- -D warnings

# Java
mvn -q spotless:apply && mvn -q verify
mvn -q pmd:check

# Ruby
rubocop -a && rubocop

# PHP
vendor/bin/pint && vendor/bin/phpstan analyse
```

See also: `../SKILL.md` (refactoring loop, failure modes), `file-responsibility-rules.md` (hard size limits), `code-smells.md` (what to look for), `../../testing-patterns/references/test-strategy.md` (the safety net).
