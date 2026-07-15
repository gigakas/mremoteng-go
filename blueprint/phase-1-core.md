# Phase 1 — Data core (model parity, no UI or protocols)

**Goal**: faithful port of the original C# connection model, XML/CSV
serialization and encryption, validated against real files.
**Depends on**: nothing (may run in parallel with Phase 0).

**Owned packages**: `internal/connection/`, `internal/serialize/xml/`,
`internal/serialize/csv/`, `internal/security/`.

## Stages

| # | Stage | Status |
|---|---|---|
| 1.1 | Connection model and tree | pending |
| 1.2 | Inheritance resolution | pending |
| 1.3 | Encryption (AES-GCM + PBKDF2 + legacy Rijndael) | pending |
| 1.4 | XML deserialization v26/v27/v28 | pending |
| 1.5 | XML serialization (v28 writer) | pending |
| 1.6 | CSV serialization | pending |
| 1.7 | Compatibility corpus | pending |

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
