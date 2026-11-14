---
timestamp: 2026-11-14T23:01:41Z
agent: mimocode
files:
  - cmd/mremoteng/profiling.go
  - cmd/mremoteng/profiling_test.go
  - cmd/mremoteng/main.go
---

Give the release an opt-in profiling endpoint

Analysing the 0.6.4 performance hit a wall: the application burns about half a core with four idle SSH sessions and there is no way to see where without a profiler. Setting MREMOTENG_PPROF to a loopback address now serves the standard Go endpoints (/debug/pprof/profile, heap and goroutine), and nothing listens without it. Only a loopback address is accepted, because the endpoints expose goroutine stacks and heap contents; anything else is reported on stderr and ignored rather than widening the surface. startProfiling runs from main, so it also covers the release build. Validated with ./scripts/check.sh.
