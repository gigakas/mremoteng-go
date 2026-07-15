---
name: audit
description: Produce the closing audit for a blueprint stage (code quality, performance, architecture). Mandatory when finishing every stage; a stage does not count as done without it.
---

# Stage closing audit

1. Copy `auditory/TEMPLATE.md` to:

   ```
   auditory/phase<N>-stage<M>-YYYYMMDD-claude-code.md
   ```

   (today's UTC date; N = blueprint phase, M = stage within the phase).

2. Fill in **every** section with concrete findings, citing `file:line`
   where applicable:
   - **Code quality**: duplication, complexity, error handling, test
     coverage of the new code, introduced debt.
   - **Performance**: unnecessary allocations, I/O on hot paths, algorithmic
     costs; measure with a benchmark (`go test -bench`) when in real doubt.
   - **Architecture**: package boundary compliance, new dependencies,
     deviations from the blueprint and their justification.

3. Evidence: run `./scripts/check.sh` and `./scripts/smoke.sh` and record
   the results in the corresponding section.

4. If this stage completes the phase, update the top-level `README.md`
   (project status, available functionality).

5. Record the audit in the changelog:

   ```bash
   go run ./cmd/changelog new -agent claude-code -summary "Phase <N> stage <M>: closing audit"
   ```
