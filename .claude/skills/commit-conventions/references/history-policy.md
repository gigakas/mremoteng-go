# History Policy

The message format is only half of it. The other half is the shape of the history: what a commit contains and how branches land.

## Branch workflow

One branch per feature or fix. `main` is the integration branch and is never edited directly.

```text
main (protected)
└── feat/invoice-export          one task, short-lived
     ├── refactor(pricing): extract the export port
     ├── test(reports): cover the empty export
     └── feat(reports): export invoices as CSV
```

Steps:

1. Update the base: `git switch main && git pull --ff-only`.
2. Create the branch from it: `git switch -c feat/<slug>`. A hotfix starts from the released tag instead.
3. Commit per logical step (see below); push early with `git push -u origin HEAD`.
4. Open a **draft PR** for visibility while the branch is alive.
5. Rebase on `main`; do not merge `main` into the branch repeatedly.
6. PR: title = the eventual commit subject, body = why and evidence, `Closes:`/`Refs:` to the issue.
7. Merge with **squash** by default (rebase when the series is the deliverable), then delete the branch.

| Prefix | Use for | Example |
|---|---|---|
| `feat/` | New capability | `feat/invoice-export` |
| `fix/` | Bug fix | `fix/duplicate-invoice-number` |
| `perf/` | Performance work | `perf/report-query` |
| `refactor/` | Behavior-preserving change | `refactor/pricing-rules` |
| `docs/` | Documentation only | `docs/runbook-deploy` |
| `test/` | Tests only | `test/expired-card` |
| `build/`, `ci/`, `chore/` | Tooling, pipeline, maintenance | `ci/cache-node-modules` |
| `hotfix/` | Urgent fix from a release tag | `hotfix/1.4.1` |
| `release/` | Version preparation | `release/1.4.0` |

Bots keep their own prefixes (`dependabot/`, `renovate/`).

Rules:

- **Short-lived**: days, not weeks. A long branch drifts, conflicts and hides integration bugs.
- **One branch per task**, not per developer; a second task gets a second branch.
- **Never commit to `main`** "just this once"; enable branch protection so the rule does not depend on discipline.
- **Hotfix**: branch from the released tag, merge to `main`, then cherry-pick or merge the fix into the release branch.
- **Two agents**: one branch per task, and one owner per branch unless they coordinate. Rebase before pushing; never force-push a branch another agent is building on.
- **Delete after merge**; stale branches hide unfinished work.

## One commit, one logical change

A commit should be the smallest change that:

1. builds,
2. passes the tests,
3. can be described with a single subject,
4. can be reverted without collateral damage.

If reverting it would break something unrelated, it is two commits.

### What to keep separate

| Keep separate | Why |
|---|---|
| Refactor and behavior change | A revert of the fix should not undo the cleanup |
| Formatting and logic | The logic diff becomes unreadable otherwise |
| Dependency bump and code migration | They fail for different reasons |
| Generated files and their source | Generated churn hides the real change |
| Test-only changes | Reviewers need to see that behavior did not change |

### What belongs together

- A behavior change and the test that pins it.
- A schema change and its migration.
- A public API change and its documentation update.
- A bug fix and the regression test that reproduces it.

## Splitting before committing

```bash
git add -p                        # stage hunks, not files
git diff --cached                 # review exactly what will be committed
git commit                        # message describes this unit
git stash push -p -- <path>       # park unrelated work
```

Rules:

- Stage by **hunk**, not by file, when a file mixes concerns.
- If a split requires editing code to separate, do the split as its own `refactor:` commit first.
- Never split by "first half / second half" of a function; split by behavior.
- Use `git commit --fixup=<sha>` for review corrections and squash later with `git rebase -i --autosquash`.

## Order matters

When a change touches several layers, commit in dependency order:

1. `refactor:` enable the change without altering behavior.
2. `test:` the failing/regression case (optional when it ships with the fix).
3. `feat:` / `fix:` the behavior.
4. `docs:` the consequence.

Each step builds. A reviewer can stop at any commit and get a coherent tree.

## Squash, rebase or merge

| Strategy | Use when | Effect on the log |
|---|---|---|
| **Squash** | Feature branch with noisy WIP commits | One commit per PR, PR title as subject |
| **Rebase** | Each commit is coherent and worth keeping | Linear history, preserves the atomic commits |
| **Merge commit** | Long-lived branch, parallel work, release trains | Keeps branch structure, adds a merge node |

Defaults that work for most teams:

- Squash for feature PRs; the PR title and description become the commit.
- Rebase when the commits themselves are the deliverable (a refactor series, a migration).
- Protect the main branch; require linear history only if the team uses rebase consistently.

Do not mix: a history that is sometimes squashed and sometimes not makes `git log` unreliable for changelog generation.

## Rewriting history

| Situation | Allowed |
|---|---|
| Local, unpushed commits | Amend, reword, reorder, squash freely |
| Pushed to your feature branch | Rewrite if you are the only author and announce it |
| Pushed to a shared branch | No rewriting; add a commit |
| Released tag | Never rewrite; revert if needed |

```bash
git commit --amend --no-edit          # add a forgotten file, keep the message
git rebase -i --autosquash main       # squash fixups and reword before the PR
```

Never amend a merge commit that others have based work on.

## Hooks and CI

- `commit-msg` hook: validate the format locally (see the checker in `scripts/`).
- CI: validate every commit in the PR range, not only the tip — history is what gets merged.
- PR title: validate it too; squash merges make it the commit subject.
- Do not make hooks mandatory for correctness; they are a convenience. CI is the gate.

```bash
# CI: check all commits introduced by the branch
git log --format=%B origin/main..HEAD
```

## Attribution

- **Human author**: `git config user.name` / `user.email`; co-authors via `Co-authored-by:`.
- **Agent author**: `Agent: AC|OC` trailer. Required when both agents write to the same repository, so `git log` and the changelog agree.
- **Pairing**: list the human first and the agent in the trailer; do not put an agent in `Co-authored-by`.

## Release mapping

| Commit type | Version bump | Changelog |
|---|---|---|
| `feat` | minor | Added |
| `fix` | patch | Fixed |
| `perf` | patch | Changed |
| `refactor`, `docs`, `test`, `build`, `ci`, `style`, `chore` | none | Usually omitted |
| any with `BREAKING CHANGE:` | major | Changed (breaking) |
| `revert` | patch | Fixed/Removed |

If the project uses the `multi-agent-changelog` skill, the fragment records the task; the release notes are generated from the commits between tags.

## Anti-patterns

| Anti-pattern | Problem | Fix |
|---|---|---|
| One commit per file | Unreviewable, unrevertable | One commit per logical change |
| One giant commit per PR | Review is a diff dump | Split by behavior, or squash intentionally |
| "fix the build" commits on main | History becomes archaeology | Fix in the branch, squash before merge |
| Rewriting shared history | Breaks everyone's clone | Revert instead |
| Enforcing format only in the hook | `--no-verify` bypasses it | CI gate on the commit range |
| Agent attribution in the commit body | Tools do not parse prose | Trailer: `Agent: OC` |
