---
timestamp: 2026-11-14T23:17:50Z
agent: claude-code
files:
  - go.mod
  - go.sum
  - internal/vault/vault.go
  - internal/vault/vault_test.go
---

Tidy go.mod and cover vault format and failure paths

pkg/sftp is a direct dependency, so go mod tidy moves it out of the indirect block. New vault tests cover format validation, locked saves, nonce reuse, unwritable directories and Exists.
