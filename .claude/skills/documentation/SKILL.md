---
name: documentation
description: Use when starting a project, adding or changing a public API, recording an architecture decision, writing a runbook, onboarding contributors, or when existing docs may have drifted. Covers README, ADRs, API reference and endpoints, runbooks, guides, diagrams and docstrings.
---

# Documentation

Documentation is part of the product, not an afterthought. Write it **as code**: in the repository, versioned with the change, reviewed in the same pull request, and tested where possible.

## When to use

- Starting a project or a new module.
- Adding/changing a public API, configuration, or CLI.
- Making an architecture decision with trade-offs.
- Writing operational procedures (deploy, rollback, incidents).
- Onboarding contributors or users.
- A behavior changed and the existing docs may be stale.

## When NOT to write a new document

- The answer belongs in a code comment next to the logic.
- The information already exists and only needs a link.
- The document would duplicate the code (generated API docs do this better).
- Nobody will own it; an unowned document becomes stale and misleading.

Prefer **updating** an existing document over creating a new one.

## Choose the smallest document type

| Audience | Question | Document |
|----------|----------|----------|
| New contributor | "How do I run this?" | `README.md` |
| Maintainer | "Why was this designed this way?" | ADR (`docs/adr/`) |
| API consumer | "What are the endpoints, payloads and errors?" | API reference (OpenAPI/generated docs) |
| Operator | "How do I deploy/rollback/respond to an incident?" | Runbook (`docs/runbooks/`) |
| End user | "How do I accomplish a task?" | User guide (`docs/guides/`) |
| Future maintainer | "Why does this line exist?" | Code comment |
| Everyone | "What changed and why?" | `CHANGELOG.md` (`multi-agent-changelog`) |
| Architect | "How do the pieces fit?" | Diagram (C4/Mermaid) |

## Workflow

0. **Identify the audience and the question.** One document answers one primary question.
1. **Pick the smallest type** from the table.
2. **Outline first**: headings, then content. If the outline is confusing, the document is not ready.
3. **Put it near the code** (README at the root, ADRs in `docs/adr/`, runbooks in `docs/runbooks/`, API docs generated from the contract).
4. **Link it** from the README so it is discoverable.
5. **Review it in the same PR** as the code change.
6. **Test what can be tested**: commands, links, examples, generated docs.
7. **Delete or update** docs that the change invalidated. Stale docs are worse than no docs.

## Document types at a glance

### README

The front door. It must answer: what is this, why should I care, how do I install it, how do I run it, how do I test it, where do I go next.

See [references/readme.md](references/readme.md).

### ADR — Architecture Decision Record

One decision per file, with context, alternatives, decision, and consequences. Immutable once accepted; supersede with a new ADR instead of rewriting history.

See [references/adr.md](references/adr.md).

### API reference

Contract-first where possible: OpenAPI/JSON Schema/Protobuf, then generated reference plus hand-written examples. Document auth, errors, pagination, versioning, and idempotency.

See [references/api-docs.md](references/api-docs.md).

### Runbook

Operational steps for deploy, rollback, health checks, incidents, and escalation. Written for a tired person at 3 a.m.

See [references/runbooks.md](references/runbooks.md).

### User guide

Task-oriented, not feature-oriented. Start from what the user wants to accomplish.

See [references/user-guides.md](references/user-guides.md).

### Diagrams

Use C4 for architecture and Mermaid for flows/sequences/states. Keep diagrams as code and update them with the system.

See [references/diagrams.md](references/diagrams.md).

### Comments and docstrings

Comments explain **why**; docstrings document the public contract. Do not restate the code.

See [references/docstrings.md](references/docstrings.md).

## Documentation-as-code rules

- **Same repo, same PR**: a behavior change includes its docs.
- **Versioned**: docs live with the code they describe.
- **Reviewable**: docs get reviewed like code.
- **Testable**: commands, snippets and links are verified in CI where possible.
- **Generated where possible**: API reference from the contract, not hand-maintained duplicates.
- **Owned**: every document has an owner or a review trigger.
- **Discoverable**: linked from the README or an index; no orphan docs.

## Definition of Done (docs)

A change is documented when:

- [ ] The README still describes how to install, run and test.
- [ ] Public API/config/CLI changes are reflected in the reference.
- [ ] A significant architecture decision has an ADR (or the existing one is superseded).
- [ ] Operational changes have a runbook update.
- [ ] User-facing changes have a guide/help update.
- [ ] Commands and examples were executed or validated.
- [ ] Links resolve.
- [ ] `CHANGELOG.md` has an entry (see `multi-agent-changelog`).
- [ ] Obsolete docs were updated or deleted.

## Drift prevention

- Add a **docs check** to CI: link checker, Markdown lint, generated-docs drift check.
- When a public signature changes, update the docstring/reference in the same commit.
- When a command changes, update the README/runbook in the same commit.
- When an ADR is superseded, mark it `Superseded by ADR-XXXX` and link the new one.
- Review docs quarterly or when onboarding someone; onboarding exposes gaps.
- Prefer a short, correct document over a long, outdated one.

## Templates

### README skeleton

```markdown
# Project name

One-sentence description.

## Why
## Requirements
## Install
## Quickstart
## Configuration
## Usage
## Architecture
## Testing
## Contributing
## License
```

### ADR skeleton

```markdown
# ADR-0001: Title

- Status: Proposed | Accepted | Superseded by ADR-XXXX
- Date: YYYY-MM-DD
- Deciders: names

## Context
## Decision
## Alternatives considered
## Consequences
```

### Runbook skeleton

```markdown
# Runbook: Service X

## Purpose
## Scope
## Prerequisites
## Health checks
## Deploy
## Rollback
## Common incidents
## Escalation
```

## Guardrails

- **One document, one question.** Do not mix a tutorial, a reference and an ADR.
- **Do not duplicate code.** Link to the source or generate the reference.
- **Do not document intentions as facts.** If it is planned, label it.
- **Do not leave TODOs without an owner and a date.**
- **Do not rewrite accepted ADRs.** Supersede them.
- **Do not create orphan docs.** Link from the README/index.
- **Do not let docs drift.** Update them in the same change.

## Common mistakes

| Mistake | Why It Fails | Fix |
|---------|--------------|-----|
| README that only says "see docs" | No front door | Put install/run/test in the README |
| ADR that is a design doc | Too long, never read | One decision, four sections |
| API docs copied by hand | Drift | Generate from OpenAPI/schema |
| Runbook with no rollback | Incident gets worse | Include rollback and verification |
| Comments that restate code | Noise | Explain *why* |
| Docs in a separate wiki nobody updates | Drift | Keep docs in the repo |
| Huge user guide by feature | Users cannot find tasks | Task-oriented guides |
| No owner | Stale docs | Assign owner/review trigger |

## References

- [references/readme.md](references/readme.md) — README structure, quickstart, badges, checklist
- [references/adr.md](references/adr.md) — ADR format, lifecycle, examples
- [references/api-docs.md](references/api-docs.md) — contract-first API docs, per-language generators
- [references/runbooks.md](references/runbooks.md) — operational runbooks and incident response
- [references/user-guides.md](references/user-guides.md) — task-oriented guides and troubleshooting
- [references/diagrams.md](references/diagrams.md) — C4 and Mermaid diagrams
- [references/docstrings.md](references/docstrings.md) — docstrings, comments, per-language conventions
- `multi-agent-changelog` — `CHANGELOG.md` format and per-agent fragments
- `code-audit` — audit history and improvement backlog

## Repo adjustments (mremoteng-go)

- **Toolchain**: Go 1.25+ (`go.mod` declares `go 1.25.0`).
- **Language**: code comments, documentation, changelog entries and audits are written in English.
- **Existing docs map** — extend these, do not invent new trees:
  - `blueprint/linux-preview.md` — active delivery track; it overrides the phase order in `blueprint/README.md`
  - `docs/MIGRATION_PLAN.md` — design rationale; never infer implementation status from its roadmap
  - `auditory/` — stage audits (`TEMPLATE.md`, `README.md`); committed audits are immutable, a re-review is a new audit
  - `CHANGELOG.md` — generated; the source of truth is `changelog.d/`
- **Changelog**: fragments via `go run ./cmd/changelog new -agent <agent> -summary ...`; `make changelog` regenerates. See the `changelog` skill.
- **Gate**: `./scripts/check.sh` before finishing.
- **Versioned facts**: the window title and About dialog carry version, commit and build time. If you document UI behavior, verify it on a rebuilt binary (`./scripts/dev.sh`), not from source alone.
