---
description: Full project verification (gofmt, go vet, unit tests)
---

Run the full verification and fix whatever fails:

```bash
./scripts/check.sh
```

If `gofmt` reports unformatted files, run `gofmt -w .` and re-run the
script. Test conventions: unit tests next to the code (`_test.go`, same
package), naming `TestFunction_Scenario_ExpectedResult`, table-driven cases,
`t.TempDir()` for temporary files.
