# Audit — Phase 1, Stage 3

- **Date (UTC)**: 2026-07-15
- **Agent**: opencode
- **Audited stage**: 1.3 Encryption (AES-GCM + PBKDF2 + legacy Rijndael)
- **Commits covered**: working-tree change set for stage 1.3 (uncommitted; recorded in the changelog fragment produced at stage close).

## 1. Code quality

Five source files, three test files (~250 LOC source, ~300 LOC tests), each with a single responsibility:

- `security.go` — `Provider` interface (`security.go:25`) and the `ErrEmptyPassword` sentinel (`security.go:38`); compile-time interface assertions for both providers.
- `kdf.go` — PBKDF2 wrapper + mRemoteNG parameters (`pbkdf2Iterations = 1000`, `aeadKeyBytes = 32`, `kdf.go:18-19`) and the password-encoding helper.
- `aead.go` — modern AES-256-GCM provider, layout `base64(salt[16] || nonce[16] || ciphertext || tag[16])` (`aead.go:17`, blob assembly at `aead.go:55-59`).
- `legacy.go` — AES-128-CBC provider, layout `base64(iv[16] || ciphertext)` (`legacy.go:19`), key = `md5.Sum(password)` (`legacy.go:90-91`).
- `pkcs7.go` — PKCS#7 pad/unpad with strict validation.

Findings:

- Error handling is consistent: every error path wraps with `%w` and a `security:` prefix; the empty-password case is a typed sentinel asserted with `errors.Is` in the tests.
- No speculative comments; the comments present are the package doc and the two non-obvious parity facts (the 16-byte GCM nonce rationale at `aead.go:20-23` and the PBKDF2-SHA1/1000 weakness note at `kdf.go:13-17`), both load-bearing for future maintainers.
- One minor duplication: `base64.StdEncoding` appears inline in both providers' encrypt/decrypt. Acceptable — extracting a helper would obscure the symmetric structure for no real gain at this size.
- `pkcs7Pad` allocates a pad slice and appends (`pkcs7.go:12-15`); trivial at the data sizes here (connection attribute strings).

## 2. Performance

This is not a hot path: encryption runs once per file save/load and per encrypted attribute, not per byte, exactly as in the C# app. `deriveAEADKey` runs 1000 HMAC-SHA1 iterations per call (`kdf.go:30`); measured cost is sub-millisecond, identical in profile to `Pkcs5S2KeyGenerator.DeriveKey` in the original. No I/O. No benchmark added — the per-attribute cost is negligible and matches the reference implementation; profiling would be premature.

## 3. Architecture

- **Package boundary**: only `internal/security/` was modified. `go.mod`/`go.sum` were **not** changed — `golang.org/x/crypto` was already added to the module by claude-code's phase-0.1 commit (`ad31676`), so this stage introduces no new dependency. The dependency is the one the blueprint mandates for PBKDF2.
- **Key decision — non-standard GCM nonce**: mRemoteNG uses a 16-byte GCM nonce (`NonceBitSize = 128`). `cipher.NewGCM` rejects non-12-byte nonces, so `cipher.NewGCMWithNonceSize` is used (`aead.go:100`). I verified in the toolchain (`crypto/internal/fips140/aes/gcm/gcm_asm.go:74-81`) that this path derives the initial counter from the nonce via GHASH per NIST SP 800-38D §8.1 — the same construction BouncyCastle's `GcmBlockCipher` applies, which is what makes Go-produced ciphertexts interoperable with C#-produced ones.
- **Format parity**: parameters were taken by reading the actual C# sources (`AeadCryptographyProvider.cs`, `Pkcs5S2KeyGenerator.cs`, `LegacyRijndaelCryptographyProvider.cs` from the mRemoteNG `develop` branch): PBKDF2-HMAC-**SHA1** (BouncyCastle `Pkcs5S2ParametersGenerator` default digest, **not** SHA-256), salt doubles as GCM associated data, Latin-1 password encoding via `pkcs5PasswordToBytes` (`kdf.go:42`).
- **Security trade-off, justified**: SHA-1 and 1000 PBKDF2 iterations are weak by modern standards but are part of the file format — changing them would break interoperability with existing files and the C# app (Phase 5 coexistence). Documented inline at `kdf.go:13-17`; a v2 format bump would be the place to strengthen them.
- **Collision avoidance**: while this stage was in progress claude-code committed phase-0.1. I detected that the spike is `//go:build spike`-gated (excluded from the default build) and that `go mod tidy` would have needlessly surfaced the spike's Fyne/xgb dependencies into my change set; I reverted that and left `go.mod`/`go.sum` untouched, so the two stages do not collide on module metadata.

## 4. Evidence

- `./scripts/check.sh`: green — `gofmt` clean, `go vet ./...` clean, `go test ./...` green.
- `./scripts/smoke.sh`: green — binaries build, `mremoteng` starts, changelog compile reproducible.
- New tests in this stage (22 total in `internal/security`):
  - `aead_test.go` — round-trip (ascii/unicode/long/aligned), wrong password, tampered ciphertext, tampered salt, per-encryption randomness, layout length, empty/empty-password/malformed input.
  - `legacy_test.go` — round-trip, MD5-key identity, per-encryption randomness, empty/malformed/non-aligned input, cross-provider incompatibility, interface conformance for both providers.
  - `kdf_test.go` — PBKDF2 against RFC 6070 vectors (c=1, 2, 4096), `deriveAEADKey` equals direct PBKDF2-SHA1/1000, password encoding for ASCII/Latin-1/astral code points, PKCS#7 round-trip and rejection of invalid padding.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- **Cross-validation against real C# output** (owner: stage 1.7, human + claude-code). The format parity here is derived from a direct port of the C# sources, which gives high confidence, but empirical proof that a Go-decrypted value matches a C#-encrypted value requires the compatibility corpus. Generating those files needs the C# app and is explicitly stage 1.7's deliverable; it is blocking for Phase 2, not for this stage.
- The phase is **not** complete after 1.3 (1.1, 1.2, 1.4, 1.5, 1.6, 1.7 remain), so the top-level `README.md` is intentionally left unchanged.
