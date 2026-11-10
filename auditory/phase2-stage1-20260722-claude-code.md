# Audit — Phase 2, Stage 2.1

- **Date (UTC)**: 2026-07-22
- **Agent**: claude-code
- **Audited stage**: Protocol interface + factory
- **Commits covered**: uncommitted at audit time; see changelog fragments
  `20260722T212319Z-claude-code-phase-2-stage-2-1-add-the-protocol-inter.md`
  and `20260722T212248Z-claude-code-also-require-description-in-the-opencode.md`
  (the latter is a documentation fix unrelated to this stage's code, done in
  the same session)

## 1. Code quality

- `internal/protocol/protocol.go`: single exported type, the `Protocol`
  interface. Every method has a doc comment stating not just what it does
  but the contract implementers must honor (e.g. `Disconnect` must be
  idempotent, `OnError`/`OnClose` replace rather than accumulate callbacks)
  — this is the part of `ProtocolBase.cs` that was implicit in C# event
  semantics and needs to be explicit in a Go interface, since there is no
  base class to enforce it.
- `internal/protocol/factory.go:39-51`: `Register` panics on a nil
  constructor or a duplicate protocol type. This mirrors
  `database/sql.Register` / `image.RegisterFormat` — both are programmer
  errors caught at `init()` time (backend package load), never at runtime
  from user input, so a panic is appropriate and matches Go stdlib
  convention rather than forcing every backend's `init()` to handle an
  error it cannot meaningfully recover from.
- `registryMu sync.RWMutex` (`factory.go:12`) guards the registry map.
  Strictly, `Register` only runs from `init()` (single-goroutine, before
  `main`), so the mutex is defensive rather than currently load-bearing —
  justified because `Create` will be called from UI/session code after
  `init()`, i.e., genuinely concurrent reads, and a future test or backend
  that registers dynamically (outside `init()`) should not silently race.
- Test coverage (`protocol_test.go`, 7 tests): registry happy path,
  registry error paths (nil info, unregistered type, nil constructor,
  duplicate registration — each as its own table-free `Test*` per the
  project's `TestFunction_Scenario_ExpectedResult` convention), and a
  lifecycle test exercising every interface method plus both callbacks
  through a `fakeProtocol` test double, per the blueprint's explicit
  instruction ("Tests with a fake protocol implementation"). No real
  backend exists yet (stages 2.2-2.7), so there is nothing further to test
  at this stage.
- No duplication, no function over ~50 lines, no discarded errors.

## 2. Performance

Not applicable: this stage adds no I/O, no hot path — it's an interface
declaration and a map-backed registry populated once at startup and read
on each `Create` call (one per opened session, not a hot loop).

## 3. Architecture

- Stays entirely inside `internal/protocol/`, the package this stage owns
  per `blueprint/phase-2-protocols.md`. No other package touched.
- No new external dependencies (stdlib only: `context`, `fmt`, `sync`).
- Deliberately does **not** import any backend subpackage
  (`internal/protocol/ssh`, `.../vnc`, ...) — those don't exist yet, and
  even once they do, this package must never import them: each backend
  imports `internal/protocol` for the `Protocol` interface and calls
  `Register` from its own `init()`, so the dependency points one way only
  (backend → factory), leaving `internal/protocol` free of an import cycle
  and free of build tags for protocols the binary doesn't want to ship.
  Whichever binary wants a given protocol available blank-imports its
  package (documented in `factory.go`'s `Register` doc comment) — the
  actual wiring in `cmd/mremoteng` is deferred to when a first backend
  exists (stage 2.2), since blank-importing zero packages today would be
  dead code.
- `Constructor` receives both `*connection.ConnectionInfo` and the
  pre-resolved `connection.ConnectionValues` (`factory.go:17`) so every
  backend gets inheritance-resolved values without each one having to call
  `Effective()` itself — a deliberate deviation from `ProtocolFactory.cs`,
  which passes only `ConnectionInfo` and lets each concrete protocol read
  from it directly (C# has no separate raw/effective split; Go's
  `ConnectionInfo.Effective()` from stage 1.2 does, and factory is the
  natural place to resolve it once).
- `Connect(ctx context.Context) error` is the one deliberate departure from
  a literal `ProtocolBase.Connect()` port: idiomatic Go for an I/O-starting
  call, needed by later stages for connect-timeout and cancellation
  (e.g. closing a tab mid-handshake). Recorded here since it's an
  interface decision every later stage must follow.
- No impact on already-closed Phase 1 packages.

## 4. Evidence

- `./scripts/check.sh`: **green** (`gofmt` clean, `go vet` clean, all
  packages `ok`, including the new `internal/protocol`).
- `./scripts/smoke.sh`: binaries build and `mremoteng` starts correctly;
  the script's own `changelog compile` reproducibility check currently
  fails, but only because this session has other legitimate uncommitted
  `changelog.d/` fragments queued for commit (from a concurrent
  documentation task earlier in the same session) — `CHANGELOG.md` in the
  working tree is already ahead of the last commit. Verified
  `go run ./cmd/changelog compile` is itself idempotent (ran it twice
  consecutively with no further diff), which is what the check is actually
  meant to guard. This resolves itself once the pending fragments are
  committed; not a defect introduced by this stage.
- New tests in this stage: `internal/protocol/protocol_test.go` — 7 tests,
  all passing (see the changelog fragment for the full list).

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Commit the working tree (this stage's code + the unrelated changelog
  documentation fix from earlier in the session) so
  `./scripts/smoke.sh`'s reproducibility check passes clean again —
  owner: human (git operations are not taken without explicit request).
- No backend blank-imports anything into `cmd/mremoteng` yet — expected;
  the first protocol stage to close (2.2 SSH/Telnet/rlogin/raw/serial, per
  the blueprint) should add the import alongside its own registration.
