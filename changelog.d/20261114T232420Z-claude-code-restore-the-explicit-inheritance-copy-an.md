---
timestamp: 2026-11-14T23:24:20Z
agent: claude-code
files:
  - internal/connection/inheritance.go
  - internal/connection/inheritance_bench_test.go
---

Restore the explicit inheritance copy and add an Effective benchmark

The reflection-based apply introduced in the previous refactor made Effective about 22 times slower (9 µs against 0.4 µs, 12 KB and 11 allocations per call at depth 5). The field-by-field copy is back, documented as deliberate, and BenchmarkEffective_Depth5 guards it; the reflection tests still check that every flag copies only its own field.
