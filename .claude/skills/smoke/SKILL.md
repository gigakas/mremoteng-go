---
name: smoke
description: Run the project smoke test (builds all binaries and verifies they start). Use before committing and after changes to cmd/ or dependencies.
---

# Smoke test

```bash
./scripts/smoke.sh
```

Builds every binary (`./...`), runs `mremoteng` verifying its output, and
checks that `changelog compile` is reproducible (regenerating it must not
produce a diff in `CHANGELOG.md`).

If the reproducibility check fails, it almost always means someone edited
`CHANGELOG.md` by hand — the fix is to regenerate it:

```bash
go run ./cmd/changelog compile
```
