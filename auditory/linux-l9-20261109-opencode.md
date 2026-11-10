# Audit — Linux track, stage L9

- **Date (UTC)**: 2026-11-09
- **Agent**: opencode
- **Audited stage**: L9 — encrypted credential vault
- **Scope**: `internal/vault`, `internal/ui/vault.go`, session password resolution

## 1. Code quality

- File format is versioned and self-describing (`version`, `algorithm`,
  `iterations`, salt, nonce, payload), so a future KDF change can be detected
  instead of silently mis-decrypting.
- Key derivation: PBKDF2-HMAC-SHA256, 210k iterations, 16-byte random salt per
  file; encryption AES-256-GCM, which authenticates the payload (a modified
  ciphertext fails with `ErrWrongPassword` instead of decrypting garbage).
- Writes are atomic (temp file `0600`, `Sync`, rename) and the tests assert the
  permissions *and* that the secret never appears in the file.
- Master password minimum is 8 characters; `Lock` wipes key, cipher and secrets
  and subsequent writes fail with `ErrLocked`.
- Session panels resolve passwords as: typed value, else stored secret for that
  profile id, else empty. The vault never receives a password implicitly: storing
  is an explicit button.

## 2. Performance

- Derivation is the only expensive step and runs once per unlock (210k
  iterations), not per session.

## 3. Architecture

- The vault is independent of the profile store: profiles keep no secret fields,
  so the plain JSON never gains credentials.
- `internal/vault` has no dependency on Fyne; the UI only orchestrates
  create/unlock/lock and the store button.
- External providers (Vault/OpenBao, 1Password, Delinea) are **not** implemented;
  the backlog keeps them under L9 with a provider interface still to be designed.
- Scope limit: SSH keys and RDP gateway credentials are not stored; only the
  session password.

## 4. Evidence

- `./scripts/check.sh`: green.
- New tests: vault round trip, owner-only permissions, no plaintext secret,
  wrong password, tampered payload detection, short password rejection, delete,
  lock semantics, missing file; UI tests for create/store/use/lock, typed
  password precedence, persistence through a fresh `vault.Open`, and wrong
  password reporting.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Manual check: create a vault, store an SSH password, reconnect without typing,
  and confirm the file content is ciphertext.
- Forgetting the master password means the file is unrecoverable: document this
  in the usability guide.
- External credential providers remain open.
