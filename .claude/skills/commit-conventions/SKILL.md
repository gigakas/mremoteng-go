---
name: commit-conventions
description: Use when writing or reviewing commit messages, splitting a change into commits, choosing a branch, or defining the repository's commit format. Covers Conventional Commits, branching, breaking changes, agent trailers, atomic history and squash/rebase policy.
---

# Commit Conventions

A commit message is the only documentation that travels with the change into `git log`, blame, review and release notes. It has one job: tell a future reader **what changed and why**, in a format a tool can parse.

The format is **Conventional Commits 1.0.0** plus two project additions: a breaking-change marker and an agent trailer for shared repositories.

## When to use

- Writing any commit message.
- Reviewing a PR and the history is a wall of "fix", "wip", "changes".
- Splitting a large change into reviewable commits.
- Defining or enforcing the format for a repo (hook, CI, PR titles).
- Preparing a release: changelog and version bump derive from commit types.

## When NOT to use

- The changelog entry or release notes themselves — `multi-agent-changelog`.
- The architecture decision behind the change — `documentation` (ADR).
- Rewriting published history to "fix" messages: that is a team decision with coordination, not a local convention.
- Commit signing setup — follow the org's policy; this skill only covers message content.

## The standard (Conventional Commits 1.0.0)

The normative structure, exactly as the specification defines it:

```text
<type>[optional scope][!]: <description>

[optional body]

[optional footer(s)]
```

| # | Rule (MUST unless marked) | Example |
|---|---|---|
| 1 | Header = `type`, optional `scope`, optional `!`, then `: ` (colon + space) and the description | `feat(api): add pagination` |
| 2 | `type` is a lowercase noun; the spec defines `feat` and `fix`, other nouns are allowed | `feat`, `fix`, `perf` |
| 3 | `scope` is a noun in parentheses, describing the area | `(checkout)`, `(auth)` |
| 4 | `!` goes immediately before `:`, and marks a breaking change | `feat(api)!: rename order_id` |
| 5 | The description follows the colon+space immediately; no other punctuation before it | `fix: reject duplicate invoice numbers` |
| 6 | The body, if present, starts **one blank line** after the description | see below |
| 7 | Footers, if present, start **one blank line** after the body | `Refs: #412` |
| 8 | A footer token is a word, or words joined by `-`; then `: ` and the value | `Co-authored-by: Ana <a@b.co>` |
| 9 | `BREAKING CHANGE` (or `BREAKING-CHANGE`) is uppercase and may be used as a footer token | `BREAKING CHANGE: …` |
| 10 | The body and footers are free-form text; only the header is parsed by tooling | — |

Style rules (SHOULD — Git project convention, not part of the spec):

- Imperative mood: "add", "reject", "rename" — not "added" or "adds".
- Subject ≤ **50** characters ideally; **72** is the hard limit used by the checker.
- No trailing period in the subject.
- Body wrapped at **72** characters; paragraphs separated by a blank line.

### Semantic versioning mapping

| Commit | Version bump |
|---|---|
| `fix:` | PATCH |
| `feat:` | MINOR |
| any type with `!` or `BREAKING CHANGE:` | MAJOR |
| `perf:`, `refactor:`, `docs:`, `test:`, `build:`, `ci:`, `style:`, `chore:` | none |

### Project additions (not in the spec)

These are local decisions, applied by the checker and stated here so nobody has to guess:

| Addition | Rule |
|---|---|
| Breaking change | `!` **and** a `BREAKING CHANGE:` footer explaining the migration (the spec allows either; we require both) |
| Agent attribution | Trailer `Agent: AC` or `Agent: OC` when both agents write to the repository |
| Scope alphabet | `[a-z0-9._/-]+` (lowercase) |
| Exempt headers | `Merge …`, `Revert "…"`, `fixup! …`, `squash! …` pass untouched |

### Types

| Type | Use for | Appears in changelog |
|---|---|---|
| `feat` | New user-visible capability | Yes (minor) |
| `fix` | Bug fix | Yes (patch) |
| `perf` | Performance improvement without behavior change | Yes |
| `refactor` | Structure change, same behavior | No |
| `docs` | Documentation only | No |
| `test` | Tests only | No |
| `build` | Build system, dependencies, packaging | No |
| `ci` | CI configuration and scripts | No |
| `chore` | Maintenance that fits nowhere else | No |
| `style` | Formatting/whitespace, no code meaning | No |
| `revert` | Reverts a previous commit | Yes |

If a change needs two types, it is two commits.

### Footers that matter

```text
BREAKING CHANGE: `POST /orders` now requires `idempotency_key`
Refs: #412
Closes: #398
Agent: OC
Co-authored-by: Ana <ana@example.com>
```

- `BREAKING CHANGE:` is required whenever the `!` marker is used; the body explains the migration.
- `Agent: AC|OC` identifies which agent produced the change when opencode and Claude Code share the repository. It is the commit counterpart of the `changelog.d/<AC|OC>/` fragment.
- `Closes:`/`Refs:` only with the real issue number; never invent one.

## Procedure

0. **Work on a branch.** `git switch -c feat/<slug>` from up-to-date `main`; never commit to `main` directly. Hotfixes branch from the released tag. See `references/history-policy.md`.
1. **Decide the unit.** One commit = one logical change that builds and passes tests on its own.
2. **Choose the type** by the effect on the product, not the file touched.
3. **Name the scope** from the domain (`invoice`, `checkout`), not the layer (`service`, `utils`).
4. **Write the subject** as an instruction: "reject duplicate invoice numbers", not "duplicate invoice numbers rejected" or "fixed bug".
5. **Add the body only for the why.** If the diff explains it, skip the body.
6. **Add a body only for the why**, and never a trailer (see the repository
   adjustments below).
7. **Split or squash** before committing (see `references/history-policy.md`).
8. **Verify** the history with `git log --oneline` before pushing.

## Enforcing it

| Layer | How |
|---|---|
| Local hook | Not installed in this repository; the rules live in `AGENTS.md` |
| CI | `git log --format=%B <range>` reviewed against the repository adjustments |
| PR title | Same wording as the commit subject |

This repository deliberately has no commit-message hook: the mandate is a
changelog fragment, which the changelog tool verifies at write time.

Hooks are guidance, not a wall: allow `Merge`, `fixup!` and `squash!` to pass untouched.

## Guardrails

- **Never commit secrets** — check the diff, not just the message; a `.env` in a commit is a rotation incident.
- **Do not amend or force-push shared history** to fix wording; add a clarifying commit or coordinate.
- **Do not bypass hooks** (`--no-verify`) without saying why in the PR.
- **One type per commit**; mixed commits cannot be reverted or released selectively.
- **The subject is not a changelog line.** The changelog is written for users; the subject is for developers.
- **Do not encode the ticket in the type** (`feat-JIRA-123`); put it in `Refs:`.

## Done when

- [ ] Header matches `<type>(<scope>)!: <subject>` with a valid type.
- [ ] Subject is imperative, lowercase, ≤ 72 chars, no trailing period.
- [ ] Breaking changes carry `!` and a `BREAKING CHANGE:` footer explaining migration.
- [ ] Body, if present, explains *why* and is separated by a blank line.
- [ ] Footers reference real issues; shared repos include `Agent: AC|OC`.
- [ ] The commit builds and passes tests on its own.
- [ ] `check-commit-msg.sh` accepts the message.
- [ ] The branch is named by type (`feat/`, `fix/`, ...), merged via PR and deleted.

## Common mistakes

| Mistake | Why it fails | Fix |
|---|---|---|
| `fix` / `wip` / `changes` / `update` | Carries no information; breaks changelog generation | Type + scope + effect |
| `feat: added login` | Past tense and vague | `feat(auth): add passwordless login` |
| Mixing refactor and behavior | Cannot revert or release one without the other | Two commits, in order |
| Breaking change hidden in the body | Consumers miss it | `!` + `BREAKING CHANGE:` footer |
| Ticket number as the whole subject | Nobody knows what changed | Description + `Refs: ABC-123` |
| Formatting commit mixed with logic | Review diff becomes unreadable | `style:` commit first, then the change |
| Inventing an issue number | Broken traceability | Only real references |

## References

- [`references/examples.md`](references/examples.md) — good/bad messages by type, breaking changes, reverts, agent trailers
- [`references/history-policy.md`](references/history-policy.md) — atomic commits, splitting, squash/rebase, amend and shared history
- `multi-agent-changelog` — the per-task fragment that consumes these commits
- `documentation` — ADRs for the decisions behind a change
- Conventional Commits 1.0.0: <https://www.conventionalcommits.org/en/v1.0.0/>

## Repo adjustments (mremoteng-go)

AGENTS.md overrides the generic advice above, and the override is absolute:

- **No trailers, no signatures.** The message carries the configured Git user
  only: no `Co-Authored-By`, no `Generated with`, no agent or model name. A
  model name in the trailer is a rule violation here, not a convention.
- **One logical change per commit**, and every logical change — code, docs, an
  audit — needs its own fragment:

  ```bash
  go run ./cmd/changelog new -agent <agent> -summary "<imperative summary>"
  ```

  The tool detects changed files from Git, regenerates `CHANGELOG.md`, and
  refuses a summary containing `---`. Never hand-edit `CHANGELOG.md` and never
  edit or delete an existing fragment.
- **Imperative, English, no phase-less ambiguity**: summarise the *why*; put the
  detail in `-description`.
- **Keep refactoring separate from functional commits.** A change that both moves
  code and changes behaviour is two commits.
- **Stage files by name.** `git add -A` and `git add .` risk committing a `.env`,
  a credential file or a build artefact; this repository has all three.
- Commit history is not rewritten on a shared branch: fix with a new commit.

