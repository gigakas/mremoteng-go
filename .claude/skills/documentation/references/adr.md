# ADR — Architecture Decision Record

An ADR records **one significant decision**, the context that forced it, the alternatives considered, and the consequences. It is not a design document, a tutorial, or a meeting transcript.

## When to write an ADR

Write one when the decision is:

- **Architecturally significant**: hard to reverse, affects many modules, or constrains future work.
- **Contested**: there were real alternatives with trade-offs.
- **Non-obvious**: a future maintainer would ask "why on earth?".
- **Cross-cutting**: affects security, data, performance, deployment, or dependencies.

Do **not** write an ADR for:

- routine implementation choices (naming, formatting);
- decisions already obvious from the code;
- reversible experiments (use a spike/issue instead).

## File location and naming

```
docs/adr/
├── 0001-use-postgres-for-primary-storage.md
├── 0002-adopt-ports-and-adapters.md
└── 0003-use-outbox-for-domain-events.md
```

- `NNNN`: zero-padded, sequential, never reused.
- `title`: kebab-case, imperative or noun phrase.
- One decision per file.

## Lifecycle

| Status | Meaning |
|--------|---------|
| **Proposed** | Under discussion; not binding yet |
| **Accepted** | The current decision |
| **Rejected** | Considered and not adopted; keep for history |
| **Superseded by ADR-XXXX** | Replaced by a newer decision |
| **Deprecated** | Still in code but no longer recommended |

Accepted ADRs are **immutable**. To change a decision, write a new ADR and mark the old one superseded. Do not rewrite history.

## Template

```markdown
# ADR-0001: Use PostgreSQL for primary storage

- Status: Accepted
- Date: 2026-01-01
- Deciders: Alice, Bob
- Supersedes: —
- Superseded by: —

## Context

What forces are at play? Constraints, requirements, scale, team skills,
existing systems, deadlines. Be factual.

## Decision

What we will do. One paragraph, active voice.

## Alternatives considered

| Alternative | Pros | Cons | Why not chosen |
|-------------|------|------|----------------|
| PostgreSQL | ... | ... | — |
| MySQL | ... | ... | ... |
| SQLite | ... | ... | ... |

## Consequences

### Positive

- ...

### Negative

- ...

### Neutral / follow-ups

- ...
```

## Good example (compressed)

```markdown
# ADR-0002: Adopt ports and adapters

- Status: Accepted
- Date: 2026-01-02

## Context

Business logic imports the ORM and the payment SDK directly. Unit tests need a
real database, and swapping a provider requires editing use cases.

## Decision

Define ports (interfaces) in the domain and implement adapters in
`infrastructure/`. Use cases depend only on ports; the composition root wires
adapters.

## Alternatives considered

| Alternative | Pros | Cons | Why not chosen |
|-------------|------|------|----------------|
| Keep direct imports | Less code | Untestable, rigid | Blocks provider swap |
| Generic repository | Less boilerplate | Leaks query concerns | Hides domain needs |

## Consequences

Positive: use cases testable with fakes; providers swappable.
Negative: more interfaces and a composition root.
Follow-up: add an architecture test forbidding domain → infrastructure imports.
```

## Bad example

```markdown
# ADR: Database

We chose Postgres because it's better.
```

Why it fails: no context, no alternatives, no consequences, no decision owner, not actionable.

## Review checklist

- [ ] One decision, not a bundle.
- [ ] Context explains the forces, not the solution.
- [ ] At least two real alternatives (or "no viable alternative" justified).
- [ ] Consequences include negatives.
- [ ] Status is correct.
- [ ] Linked from the README/architecture doc.
- [ ] Superseded ADRs link forward.
- [ ] Date and deciders are recorded.

## Anti-patterns

| Anti-pattern | Problem | Fix |
|---|---|---|
| ADR as a full design doc | Nobody reads it | One decision, four sections |
| Rewriting an accepted ADR | Loses history | Supersede with a new ADR |
| No alternatives | Looks like a rationalization | Document the real trade-offs |
| No negative consequences | Unrealistic | List costs and risks |
| Decision without owner | No accountability | Record deciders |
| ADR for every small choice | Noise | Reserve for significant decisions |
