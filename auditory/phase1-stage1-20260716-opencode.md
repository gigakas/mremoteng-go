# Audit — Phase 1, Stage 1

- **Date (UTC)**: 2026-07-16
- **Agent**: opencode
- **Audited stage**: Connection model and tree
- **Commits covered**: stage 1.1 working tree, committed immediately after this audit

## 1. Code quality

- `ConnectionValues` is the flat protocol superset (`internal/connection/model.go:38`), while `ConnectionInfo.Raw` explicitly identifies persisted local values (`internal/connection/model.go:27`). This avoids ambiguity when stage 1.2 adds inherited effective reads.
- IDs and parent links are private (`internal/connection/model.go:19-23`); constructors validate imported IDs and random IDs use RFC 4122 version 4 bits (`internal/connection/model.go:158-171`, `internal/connection/model.go:241-250`).
- Tree mutation is centralized in `InsertChild` (`internal/connection/tree.go:118`): it validates before detaching, maintains both relationship directions and rejects cycles. `validatedNodeBase` (`internal/connection/tree.go:280`) prevents a container's base record being inserted as a leaf alias.
- `Children` returns a copy and `Node` is sealed to the package, so callers cannot bypass mutation invariants. A private container base replaces public embedding after review found the aliasing risk.
- Error values support `errors.Is`; malformed indices include the rejected value with `%w` context.
- Eighteen unit tests cover defaults, ID generation/preservation, protocol ports, construction, ordering, cross-parent moves, removal, traversal, cycle rejection, atomic failed moves, defensive child slices, alias rejection and nil receivers (`internal/connection/model_test.go:9-94`, `internal/connection/tree_test.go:9-173`).

## 2. Performance

- Child lookup, removal and reordering are O(n), matching the original ordered `List<ConnectionInfo>` and appropriate for a UI tree.
- `Children` and `Descendants` allocate result slices intentionally to prevent external mutation. No I/O or background work occurs.
- ID generation reads 16 cryptographically random bytes once per node. No benchmark was warranted: operations are user-driven and tree sizes are expected to be small relative to serialization/protocol costs.

## 3. Architecture

- Changes are confined to the phase-owned `internal/connection/` package; no dependency or module-file changes were introduced.
- String-backed enums preserve the canonical C# XML/CSV tokens and can retain unknown future values. `ProtocolSerial` is the only extension; it is isolated and required by Phase 2.2.
- Containers carry a complete connection record through a private base (`internal/connection/tree.go:32`), preserving their role as inheritance templates without exposing a second mutable node identity.
- The C# model allows ancestor cycles; the Go tree deliberately rejects them (`ErrCycle`). This safety improvement does not alter valid serialized trees and prevents recursion failures in traversal/serialization.
- UI property-change events, sorting, cloning and inheritance resolution are intentionally excluded: they are outside 1.1. The `Raw` boundary is prepared for 1.2's explicit effective-value resolver.

## 4. Evidence

- `./scripts/check.sh`: PASS — gofmt, go vet and all unit tests green.
- `./scripts/smoke.sh`: PASS — binaries build/start and changelog is reproducible.
- New tests in this stage: 18 tests in `model_test.go` and `tree_test.go`; `go test -race ./internal/connection/` also PASS.

## 5. Verdict

- [x] Stage closed unconditionally
- [ ] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- None for stage 1.1. Stage 1.2 will add inheritance flags and effective-value resolution on the established `Raw` boundary.
- Phase 1 is not complete, so the top-level README is unchanged.
