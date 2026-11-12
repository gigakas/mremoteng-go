---
name: refactoring
description: Use when cleaning up code, removing duplication and code smells, simplifying methods/conditionals/classes, preparing code for a feature, or reducing technical debt. Smell→technique triage, extraction and move recipes, per-language tooling, and the test safety net.
---

# Refactoring

Refactoring is a systematic process of improving code **without creating new functionality**. It transforms messy code into clean, simple design. Source: [refactoring.guru](https://refactoring.guru/refactoring).

Refactoring is **not** rewriting. A rewrite changes behavior and structure at once; refactoring changes structure in small, reversible, behavior-preserving steps.

## When to use

- Cleaning up code you or someone else just wrote (before/after adding a feature).
- Removing a code smell (see [references/code-smells.md](references/code-smells.md)).
- Reducing technical debt in code your task already touches.
- Preparing code for an upcoming change ("make the change easy, then make the easy change").
- Reviewing code and proposing structural improvements.

## When NOT to refactor

- **No safety net and no time to build one** — add characterization tests first, or do not refactor.
- **The code is about to be deleted** — deleting is cheaper than cleaning.
- **You do not understand the behavior** — observe and test it first.
- **You are mid-feature with failing tests** — get green before restructuring.
- **The performance path is not measured** — refactoring for speed without benchmarks is guessing.
- **The change is really a rewrite** — split it; rewrites are not refactors.

## Core principles

1. **Preserve behavior.** Refactoring must not change what the code does. Tests are the safety net.
2. **Small steps.** Apply one refactoring at a time and run tests after each step.
3. **No new features.** Keep refactoring and feature work in separate commits/changes.
4. **Motivate each change.** Every technique has trade-offs; do not apply it mechanically.
5. **Stop when it is good enough.** Refactoring is a means, not a goal.

## The refactoring loop

```
0. Safety net   → tests green (unit + regression/characterization)
1. Smell        → name the problem, not the technique
2. Technique    → choose the smallest one that removes the smell
3. Mechanics    → apply in tiny, reversible steps
4. Test         → run after every step; undo if red
5. Commit       → checkpoint only when green
6. Record       → changelog/audit if the project uses them
```

### 0) Establish the safety net

```bash
# Before touching anything, confirm the tests pass
pytest .
# or the runner of your stack: npm test / go test ./... / cargo test / flutter test
```

Use the four-layer strategy in [`../testing-patterns/references/test-strategy.md`](../testing-patterns/references/test-strategy.md): unit tests for the loop, and regression/characterization tests for the behavior you are about to change. Run the fast layer before and after each step (`./scripts/check.sh`, which runs the formatting check, `go vet` and the full test suite); that file also defines the layers and their environment variables.

If no test covers the area, write a **characterization test** that pins the current behavior (including bugs) before refactoring. See `test-driven-development`.

### 1) Name the smell

Use [references/code-smells.md](references/code-smells.md). The smell determines the treatment; do not start from a technique you like.

Common entry points:

| Situation | Technique |
|-----------|-----------|
| Method too long | Extract Method |
| Class doing too much | Extract Class |
| Duplicate code | Extract Method / Pull Up Method |
| Long parameter list | Introduce Parameter Object |
| Nested conditions | Replace Nested Conditional with Guard Clauses |
| `if/elif` on type | Replace Conditional with Polymorphism |
| Magic numbers | Replace Magic Number with Symbolic Constant |
| Class that only delegates | Remove Middle Man / Inline Class |
| Method obsessed with another object's data | Move Method |
| Repeated `null` checks | Introduce Null Object |
| Dead code | Delete it |

### 2) Choose the smallest technique

See [references/techniques.md](references/techniques.md) for the full catalogue and [references/refactoring-recipes.md](references/refactoring-recipes.md) for step-by-step playbooks.

Prefer the technique that:
- removes the smell with the fewest new concepts;
- keeps the public API stable when possible;
- can be applied in one small step;
- does not require a new abstraction unless the variation already exists.

### 3) Apply the mechanics in small steps

For each step:

1. Make the mechanical change.
2. Run the tests.
3. If red, **undo the step** — the change altered behavior or the test was wrong.
4. Commit or checkpoint only when green.

Keep the diff reviewable. If a step touches more than one concern, split it.

### 4) Verify no behavior changed

```bash
pytest .          # full suite green
git diff          # review: structure changed, logic did not
```

Ask: "If I revert the refactor, does the product behave exactly the same?" If not, it was not a refactor.

## Smell → technique triage

| Smell | First technique | Alternatives |
|-------|-----------------|--------------|
| Long Method | Extract Method | Replace Temp with Query, Method Object |
| Large Class | Extract Class | Extract Subclass, Extract Interface |
| Primitive Obsession | Replace Data Value with Object | Replace Type Code with Class/Subclasses |
| Long Parameter List | Introduce Parameter Object | Preserve Whole Object |
| Data Clumps | Extract Class | Introduce Parameter Object |
| Switch Statements | Replace Conditional with Polymorphism | State/Strategy, Explicit Methods |
| Temporary Field | Extract Class | Introduce Null Object |
| Refused Bequest | Replace Inheritance with Delegation | Push Down Method/Field |
| Alternative Classes with Different Interfaces | Rename Method | Extract Superclass |
| Divergent Change | Extract Class | Move Method/Field |
| Shotgun Surgery | Move Method/Field | Inline Class |
| Parallel Inheritance Hierarchies | Move Method/Field | Collapse one hierarchy |
| Comments (explaining what) | Extract Method | Rename, Introduce Assertion |
| Duplicate Code | Extract Method | Pull Up Method, Form Template Method |
| Lazy Class | Inline Class | Collapse Hierarchy |
| Data Class | Move Method | Encapsulate Collection |
| Dead Code | Delete it | Remove Parameter |
| Speculative Generality | Collapse Hierarchy | Inline Class, Remove Parameter |
| Feature Envy | Move Method | Extract Method, Move Field |
| Inappropriate Intimacy | Move Method/Field | Hide Delegate |
| Message Chains | Hide Delegate | Extract Method, Move Method |
| Middle Man | Remove Middle Man | Inline Method |
| Incomplete Library Class | Introduce Foreign Method | Introduce Local Extension |

## Refactoring in the development flow

### Before a feature (preparatory refactoring)

Make the change easy, then make the easy change. Refactor the code that the feature will touch **before** adding behavior, in separate commits.

### While implementing (opportunistic refactoring)

Apply the Boy Scout Rule only within the code you are already touching:
- rename a misleading variable,
- extract a method you just made longer,
- remove a dead branch you just discovered.

Do **not** wander into unrelated modules: that turns a focused change into a risky sweep.

### After a bug fix

Add the regression test first. Then clean up the area only if the fix exposed a structural problem. Keep the fix and the cleanup in separate commits when possible.

### During review

Comment on the smallest actionable technique, not on style preferences. Point to the smell and the refactoring, e.g., "This method has three responsibilities; Extract Class would separate validation from persistence."

## When tests do not exist

Refactoring without tests is dangerous, but sometimes necessary. Use the strongest available safety net:

1. **Characterization tests** — encode current outputs for representative inputs.
2. **Golden master / approval tests** — capture full output for a complex system.
3. **Logging/observability** — compare behavior before and after in a real environment.
4. **Feature flags** — keep the old path available while the new one is verified.
5. **Branch + small commits** — make every step revertible.

If none of these are possible, limit the refactor to mechanical, obviously safe changes (rename, extract with no logic change) and stop.

## Metrics and guardrails

- **File/function size** — review thresholds and hard limits in [references/file-responsibility-rules.md](references/file-responsibility-rules.md).
- **Cyclomatic complexity** — high complexity usually means Extract Method or Guard Clauses.
- **Duplication** — extract only when the duplication is real, not incidental.
- **Coverage** — use it to find untested risk, not as a target.
- **Change frequency** — code that changes often and is hard to change is the best refactoring candidate.

## Tooling

See [references/tooling.md](references/tooling.md) for per-language formatters, linters, complexity tools, and commands. Automation helps with the mechanical part; it does not choose the refactoring for you.

## Failure modes / debugging

- **Tests break after a refactor**: you changed behavior. Revert the step and re-apply more narrowly, or the test depended on an implementation detail.
- **Refactoring becomes a rewrite**: stop. Split the work into behavior-preserving steps, or treat it as a rewrite with its own plan.
- **No tests exist**: write characterization tests first; refactoring without a safety net is gambling.
- **The diff is unreviewable**: your steps are too big. Undo and split.
- **The abstraction keeps growing**: you are refactoring toward a pattern, not toward the problem. Stop and re-read the smell.

## Guardrails

- **Never refactor and add features in the same change** — review and rollback become impossible.
- **One technique at a time** — compound refactors hide which step broke behavior.
- **Run tests after every step**, not at the end.
- **Do not refactor speculative or dead code you do not own** without tests.
- **Do not introduce a pattern to remove a smell** unless the pattern is the simplest fix.
- **Keep the public API stable** unless the refactor explicitly targets it.

## Common mistakes

| Mistake | Why It Fails | Fix |
|---------|--------------|-----|
| Refactoring without tests | No way to detect behavior changes | Add characterization tests first |
| Changing behavior "while I'm here" | Turns a safe refactor into a risky rewrite | Separate feature work from refactoring |
| Applying techniques mechanically | Adds indirection with no benefit | Only refactor when a smell or upcoming change justifies it |
| Big-bang refactor | Hard to isolate what broke | Small, testable steps, commit green |
| Ignoring the smell behind the smell | Treating symptoms, not cause | Fix the root cause (e.g., extract the class, don't just rename) |
| Refactoring everything at once | Unreviewable diff, merge conflicts | Stay within the task's blast radius |
| Renaming instead of restructuring | Cosmetic change, same design | Move/extract the responsibility |

## Done when

- [ ] Behavior is unchanged: the same tests pass before and after.
- [ ] The fast suite was green after every step, not only at the end.
- [ ] Each commit is small, reversible, and named after the technique.
- [ ] No new feature, test, or dependency entered the change.
- [ ] Coverage of the touched area did not drop.

## References

- [references/code-smells.md](references/code-smells.md) — 23 smells in 6 groups, with signs, detection signals and treatments
- [references/techniques.md](references/techniques.md) — 60+ techniques in 6 groups, plus mechanics and before/after examples
- [references/refactoring-recipes.md](references/refactoring-recipes.md) — step-by-step playbooks for common app-development situations
- [references/tooling.md](references/tooling.md) — formatters, linters, complexity and duplication tools per language
- [references/file-responsibility-rules.md](references/file-responsibility-rules.md) — per-language size/complexity thresholds
- [`../testing-patterns/references/test-strategy.md`](../testing-patterns/references/test-strategy.md) — safety net, four test layers, Definition of Done

## Repo adjustments (mremoteng-go)

- **Toolchain**: Go 1.25+ (`go.mod` declares `go 1.25.0`). `./scripts/check.sh` and `./scripts/smoke.sh` put `$HOME/.local/go/bin` on `PATH` themselves.
- **Gate**: `./scripts/check.sh` (formatting check → `go vet` → `go test ./...`, including the compatibility integration suite) is mandatory before finishing. It does not format: run `gofmt -w <files>` on whatever it reports and rerun.
- **Single package**: `go test ./internal/serialize/xml/ -run '^TestName$' -count=1`.
- **Safety net for model changes**: `go test ./integration/ -run '^TestCSharpCorpus_' -count=1 -v`. Fixtures must come from the C# app, never from the Go writer.
- **Size rule**: AGENTS.md sets the bar at 50 lines — extract duplication or split mixed-responsibility functions before extending them.
- **Commits**: keep refactoring separate from functional changes. A refactor-only commit is what makes the behavior-preserving claim reviewable.
- **Changelog**: every logical change needs a fragment (`go run ./cmd/changelog new -agent <agent> -summary ...`). See the `changelog` skill; `CHANGELOG.md` is never edited by hand.
- **Tooling**: `gofmt` for formatting, `go vet` for lints, `go test -bench` for measurements. No extra linter is configured, so do not introduce one inside a refactoring commit.
