---
description: Project smoke test (builds all binaries and verifies they start)
---

Run the smoke test:

```bash
./scripts/smoke.sh
```

Builds every binary, runs `mremoteng` verifying its output and checks that
`changelog compile` is reproducible. If the reproducibility check fails,
regenerate with `go run ./cmd/changelog compile` (it means someone edited
`CHANGELOG.md` by hand).
