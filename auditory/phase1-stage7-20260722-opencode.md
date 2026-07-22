# Audit — Phase 1, Stage 1.7

- **Date (UTC)**: 2026-07-22
- **Agent**: opencode
- **Audited stage**: Compatibility corpus
- **Commits covered**: `061810a` through the stage-closing working tree

## 1. Code quality

- The manifest now declares 20 independently hashed C# fixtures and verifies provenance, unique content, version coverage, encryption modes, credentials, nesting, and inheritance (`integration/corpus_test.go:114-173`, `testdata/corpus/manifest.json:1-255`).
- Integration coverage decrypts every fixture, rejects a wrong password, checks metadata and node values, and round-trips the complete model through the Go v2.8 writer (`integration/corpus_test.go:69-110`).
- Fixture hashes canonicalize CRLF to LF before SHA-256 so provenance is stable across Windows and Linux checkouts without weakening any non-line-ending byte comparison (`integration/corpus_test.go:208-214`).
- Reverse validation against mRemoteNG C# commit `48723ba0` found and fixed two writer incompatibilities: the declaration now uses C#'s exact lowercase `utf-8`, and full-file plaintext omits whitespace nodes (`internal/serialize/xml/serialize.go:61-75`, `internal/serialize/xml/serialize_node.go:18-32`). Regression tests cover both constraints (`internal/serialize/xml/serialize_test.go:13-27`, `internal/serialize/xml/serialize_test.go:69-88`).
- No duplicated production logic or mixed-responsibility function was introduced. The corpus is intentionally data-heavy; the single shared expectation profile avoids repeating expected nodes for 15 matrix variants.

## 2. Performance

- Corpus execution is test-only and processes 20 small files in parallel (`integration/corpus_test.go:76-80`); the complete integration package finishes in approximately 1.2 seconds on the audit machine.
- Production serialization retains O(n) traversal. Compact full-file output removes indentation writes before the existing one-time encryption and reduces plaintext/ciphertext size (`internal/serialize/xml/serialize_node.go:18-30`).
- CRLF canonicalization allocates one normalized fixture copy during tests only. No benchmark is warranted because this path is outside application runtime.

## 3. Architecture

- Changes remain within Phase 1 ownership: `internal/serialize/xml/`, `integration/`, and `testdata/corpus/`, plus required phase-closing documentation. No new Go dependency was added.
- Fixtures were produced directly by `XmlConnectionNodeSerializer26`, `XmlConnectionNodeSerializer27`, and `XmlConnectionNodeSerializer28` from original C# commit `48723ba0`; source identity is recorded per fixture in the manifest.
- Bidirectional evidence was executed externally with the original C# deserializer: both normal and full-file Go v2.8 outputs were accepted with all four nodes and effective credentials recovered. This satisfies the interoperability risk deferred by stages 1.4 and 1.5.
- The local audit environment required Go 1.26.5 and Visual Studio Text Template Transformation to build the reference C# project; these are verification tools, not project dependencies.

## 4. Evidence

- `./scripts/check.sh`: passed (`gofmt`, `go vet ./...`, and `go test ./...`).
- `./scripts/smoke.sh`: passed (all binaries built, `mremoteng` started, changelog reproducible).
- New tests in this stage: `TestCSharpCorpus_DeserializeAndRoundTrip_MatchesManifest`, complete-corpus validation, canonical fixture digest validation, C#-compatible XML declaration assertion, and `TestNodeSerializer_FullFilePayload_OmitsWhitespaceNodes`.
- External cross-check: mRemoteNG C# commit `48723ba0` accepted Go-produced normal and full-file v2.8 documents and recovered four expected nodes from each.

## 5. Verdict

- [x] Stage closed unconditionally
- [ ] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

None. Phase 1 is complete and the top-level `README.md` was updated. Phase 2 may begin.
