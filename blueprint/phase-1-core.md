# Phase 1 — Data core (model parity, no UI or protocols)

**Goal**: faithful port of the original C# connection model, XML/CSV
serialization and encryption, validated against real files.
**Depends on**: nothing (may run in parallel with Phase 0).

**Owned packages**: `internal/connection/`, `internal/serialize/xml/`,
`internal/serialize/csv/`, `internal/security/`.

## Stages

| # | Stage | Status | Agent |
|---|---|---|---|
| 1.1 | Connection model and tree | done | opencode |
| 1.2 | Inheritance resolution | done | opencode |
| 1.3 | Encryption (AES-GCM + PBKDF2 + legacy Rijndael) | done | opencode |
| 1.4 | XML deserialization v26/v27/v28 | done | opencode |
| 1.5 | XML serialization (v28 writer) | done | opencode |
| 1.6 | CSV serialization | done | opencode |
| 1.7 | Compatibility corpus | done | human + opencode |

> Phase 1 delegated to OpenCode by the user (2026-07-16). Order: 1.1
> first (blocks everything); then 1.2 and, in parallel if using suffixed
> agents (opencode-2, ...), 1.4–1.5 and 1.6. Parallel agents should use
> separate git worktrees to keep `git status`-based hooks per-agent.

### Parallelism & collision notes

Per-stage package ownership (disjoint — safe for parallel agents):

- 1.1–1.2 → `internal/connection/` (sequential between themselves: 1.2
  builds on 1.1).
- 1.3 → `internal/security/` — no dependency on 1.1; delegable from day one.
- 1.4–1.5 → `internal/serialize/xml/` — need 1.1 (model) and 1.3 (encrypted
  attributes); read-only imports of those packages, never edits.
- 1.6 → `internal/serialize/csv/` — needs 1.1; read-only import.
- 1.7 → `testdata/corpus/` + integration tests — needs every other stage
  green plus the human generating files with the C# app.

Maximum useful parallelism: 1.1 (claude-code) + 1.3 (opencode) from the
start; 1.6 (opencode) once 1.1 lands; 1.4–1.5 once 1.1 and 1.3 land.

### 1.1 Connection model and tree
- `ConnectionInfo` (flat struct, superset of every protocol's fields, same
  as the original `AbstractConnectionRecord.cs`) and `ContainerInfo`
  (folder = node with children; homogeneous tree).
- Tests: tree construction, traversal, add/move/remove nodes.

### 1.2 Inheritance resolution
- Replacement for the C# reflection mechanism (`GetPropertyValue<T>`):
  resolve at read time against `Parent` using an explicit type switch or
  generated code (`go:generate`) — decide in this stage and audit the
  decision.
- `Inherit<Field>` flags as in the original `ConnectionInfoInheritance.cs`.
- Tests: inheritance on/off per field, multi-level chains, cloning the flag
  template to children.

### 1.3 Encryption
- AES-GCM (`crypto/cipher`) + PBKDF2 (`golang.org/x/crypto/pbkdf2`) for new
  files; read-only AES-CBC for the legacy Rijndael format.
- Tests: encrypt/decrypt round-trip, vectors generated with the C# app.

### 1.4–1.5 XML serialization
- Same versioned pattern as the original: one deserializer per `ConfVersion`
  (26/27/28); reject versions above the maximum supported; always write the
  latest version.
- **Never break old files**: missing attributes take defaults.

### 1.6 CSV
- Parity with `CsvConnectionsSerializerMremotengFormat.cs`: value and
  inheritance columns kept in sync, stable column order.

### 1.7 Compatibility corpus (phase acceptance test)
- Generate ≥20 connection files with the C# app: versions 26/27/28,
  encrypted and plain, with inheritance, nested folders, credentials.
- Store them in `testdata/corpus/` and verify in integration tests that the
  Go port reads each one with identical results (host, username, decrypted
  password, inheritance flags).
- **Blocking for Phase 2**: no green corpus, no progress.

## Exit criteria
All stages done with their audits, corpus 1.7 green in
`./scripts/check.sh`, top-level README updated.
