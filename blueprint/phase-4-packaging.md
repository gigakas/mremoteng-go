# Phase 4 — Packaging and distribution

**Goal**: reproducible cross-platform builds and installable artifacts.
**Depends on**: Phase 3 (there must be an app to package).

**Owned files**: `Makefile`, `.github/workflows/`, `packaging/`.

## Stages

| # | Stage | Status | Agent |
|---|---|---|---|
| 4.1 | Cross-compilation matrix (linux/windows × amd64/arm64) | pending | opencode |
| 4.2 | CI: build + check + smoke on every PR | pending | opencode |
| 4.3 | Linux packaging (.deb/.rpm/Flatpak) with xfreerdp dependency | pending | claude-code |
| 4.4 | Windows packaging (portable zip + FreeRDP binary alongside) | pending | claude-code |
| 4.5 | Release channels (stable/preview/nightly) | pending | human + claude-code |

### Parallelism & collision notes

- File ownership per stage: 4.1 → `Makefile` (build targets), 4.2 →
  `.github/workflows/`, 4.3 → `packaging/linux/`, 4.4 →
  `packaging/windows/`, 4.5 → `.github/workflows/release*` + `packaging/`
  glue.
- 4.1 and 4.2 are declarative and verifiable in CI — good OpenCode
  candidates; 4.2 depends on 4.1's make targets. 4.3/4.4 need install
  testing on real systems, so they stay with claude-code. 4.5 needs the
  human to define channel policy.
- 4.3 and 4.4 can run in parallel (disjoint `packaging/` subtrees) but not
  with 4.5, which touches both.

### Notes

- `xfreerdp` is **never vendored into the binary**: on Linux it is a package
  dependency; on Windows the official FreeRDP binary ships next to the
  portable zip (same pattern as `PuTTYNG.exe` in the original project).
- CI must gate merges on `./scripts/check.sh` and `./scripts/smoke.sh` —
  unlike the original project, tests are a merge gate here.
- Keep the original project's channel structure (stable/preview/nightly)
  during the transition.

## Exit criteria
Stages done with audits; a tagged commit produces installable artifacts for
all four targets from CI; top-level README updated with install docs.
