# Phase 4 — Packaging and distribution

**Goal**: reproducible cross-platform builds and installable artifacts.
**Depends on**: Phase 3 (there must be an app to package).

**Owned files**: `Makefile`, `.github/workflows/`, `packaging/`.

## Stages

| # | Stage | Status |
|---|---|---|
| 4.1 | Cross-compilation matrix (linux/windows × amd64/arm64) | pending |
| 4.2 | CI: build + check + smoke on every PR | pending |
| 4.3 | Linux packaging (.deb/.rpm/Flatpak) with xfreerdp dependency | pending |
| 4.4 | Windows packaging (portable zip + FreeRDP binary alongside) | pending |
| 4.5 | Release channels (stable/preview/nightly) | pending |

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
