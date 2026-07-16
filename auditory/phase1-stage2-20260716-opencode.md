# Audit — Phase 1, Stage 2

- **Date (UTC)**: 2026-07-16
- **Agent**: opencode
- **Audited stage**: Inheritance resolution
- **Commits covered**: stage 1.2 working tree, committed immediately after this audit

## 1. Code quality

- `InheritanceFlags` explicitly mirrors every current C# `ConnectionInfoInheritance` property (`internal/connection/inheritance.go:5`). No runtime reflection or unchecked type conversion is used.
- `Effective` returns a value copy and never mutates `Raw` (`internal/connection/inheritance.go:121`), making local persisted state and behavioral inherited state unambiguous.
- The resolver is an explicit type-safe assignment table (`internal/connection/inheritance.go:146`). Although long, it has one responsibility and compile-time field/type checking; the exhaustive mapping test (`internal/connection/inheritance_test.go:149`) iterates every flag and verifies the matching field is inherited.
- Root semantics match C#: inheritance is inactive for roots and direct root children, while detached normal nodes report active but safely return local values without a parent (`internal/connection/inheritance.go:112-132`).
- Template cloning uses value semantics, so descendants cannot share mutable flag state (`internal/connection/inheritance.go:136`).
- Ten new tests cover disabled/enabled fields across string/int/bool/enum types, uninterrupted and interrupted multi-level chains, direct-root behavior, detached behavior, all-on/all-off, clone independence, exhaustive field mapping and recursive template propagation (`internal/connection/inheritance_test.go:9-207`).

## 2. Performance

- `Effective` is O(depth × enabled-field-count) and allocates no heap-backed collections. Tree cycles are already rejected by stage 1.1, so recursion terminates.
- Returning `ConnectionValues` by value copies a fixed-size record. This is preferable to mutable cached state and avoids invalidation complexity; connection reads are UI/protocol setup operations, not a high-frequency hot path.
- Recursive template propagation is O(descendants) and uses the existing preorder slice. No I/O or external calls occur.

## 3. Architecture

- Changes remain inside phase-owned `internal/connection/`; no dependencies or module changes were introduced.
- **Decision**: explicit assignment instead of reflection or generated code. This keeps the implementation portable, reviewable and statically typed. The exhaustive reflection-based test is test-only and detects mapping omissions without adding runtime reflection.
- `Raw` is intentionally public for deserializers/editors; behavioral consumers must call `Effective`. This convention is documented at the field and method boundary and will be followed by XML/CSV/protocol stages.
- A root node was added because the original inheritance activation rule depends on `RootNodeInfo`. Roots cannot be inserted as children, preserving tree topology.
- Configurable default inheritance preferences are deferred to Phase 3 settings persistence; shipped C# defaults are all false, which the zero-value Go flags reproduce exactly.

## 4. Evidence

- `./scripts/check.sh`: PASS — gofmt, go vet and all tests green.
- `./scripts/smoke.sh`: PASS — binaries build/start and changelog is reproducible.
- New tests in this stage: 10 inheritance/root tests; `go test -race ./internal/connection/` also PASS.

## 5. Verdict

- [x] Stage closed unconditionally
- [ ] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- None for stage 1.2. XML and CSV stages must write/read `Raw` and use `Effective` when reproducing C# getter behavior.
- Phase 1 is not complete, so the top-level README is unchanged.
