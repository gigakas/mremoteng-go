# Audit — Phase 1, Stage 5

- **Date (UTC)**: 2026-07-16
- **Agent**: opencode
- **Audited stage**: XML serialization (v28 writer)
- **Commits covered**: stage 1.5 working tree

## 1. Code quality

- `Serialize` owns latest-version root metadata, protection marker and optional full-file encryption (`internal/serialize/xml/serialize.go:35`). It always writes ConfVersion 2.8 and validates KDF settings.
- Node encoding is recursive and preserves tree order/IDs (`serialize_node.go:18-55`). Attribute generation uses `Effective()` values and encrypts all credential fields (`serialize_node.go:58`).
- `SaveFilter` provides explicit username/domain/password/inheritance redaction (`serialize.go:27`); filtered secrets are not encrypted or emitted.
- Inheritance attributes are generated from one declarative table and only true flags are written, matching C# (`serialize_inheritance.go:9`).
- Seven tests cover normal and full-file round trips, encrypted secrets, effective inheritance, inherited password omission, redaction, canonical node namespaces, runtime `Connected`, UseEnhancedMode correction and typed errors (`serialize_test.go:13-174`).

## 2. Performance

- Serialization is O(nodes × fields). Each saved secret performs the required PBKDF2/GCM operation; redacted secrets skip encryption.
- Full-file mode buffers node XML once before authenticated encryption. This is appropriate for configuration-sized documents and avoids unauthenticated streaming complexity.
- `Effective()` copies a fixed-size value record per node; no reflection or dynamic field lookup occurs.

## 3. Architecture

- Changes remain within stage-owned `internal/serialize/xml/`; no new dependencies or cross-stage edits were required.
- Nodes reset the root default namespace only at the top level, reproducing C#'s unqualified `Node` elements without redundant descendant declarations.
- `Connected` is written false because the Go model has no protocol-session list yet; `PleaseConnect` is a load/request flag and is intentionally not misrepresented as a live session.
- `UseEnhancedMode` is written from its actual field, intentionally correcting the C# v28 writer bug that writes `UseVmId` twice.
- Root `Export` and omission of the redundant `mrng` namespace alias are load-compatible but not byte-identical to the C# writer. Stage 1.7 will provide external-reader corpus validation.

## 4. Evidence

- `./scripts/check.sh`: PASS — gofmt, go vet and all tests green.
- `./scripts/smoke.sh`: PASS — binaries build/start and changelog is reproducible.
- New tests in this stage: 7 serializer tests; `go test -race ./internal/serialize/xml/` also PASS.

## 5. Verdict

- [x] Stage closed unconditionally
- [ ] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Stage 1.7 must confirm that the C# app reads Go-produced normal/full-file documents. Go round trips and independent C# decryption vectors are already green.
- Phase 1 is not complete, so the top-level README is unchanged.
