---
name: check
description: Run the full project verification (gofmt, go vet, unit tests). Use before considering any code change done.
---

# Full verification

```bash
./scripts/check.sh
```

Runs, in order: `gofmt -l` (fails if any file is unformatted), `go vet ./...`
and `go test ./...`. If `gofmt` reports files, run `gofmt -w .` and re-run
the script.

For a single package during development:

```bash
go test ./internal/changelog/
go test ./internal/... -run 'TestName'
```

## Test conventions

- Unit tests live next to the code (`_test.go`, same package).
- Naming: `TestFunction_Scenario_ExpectedResult` (inherited from the original
  C# project).
- Table-driven cases (`map[string]...` or a case slice) for variants of the
  same scenario.
- `t.TempDir()` for temporary files; never write outside it.
