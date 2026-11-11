---
timestamp: 2026-11-11T22:13:42Z
agent: mimocode-cred
files:
  - internal/credential/vault/vault.go
  - internal/credential/vault/vault_test.go
---

Trim a trailing slash from the vault client address in New

vault.New stored the address verbatim, so an address ending in a slash produced a doubled request path: the client concatenates Addr with paths that already start with one, giving GET //v1/secret/data/web1. delinea, passwordstate and onepassword already normalize their base URL the same way in New, so vault.New now applies strings.TrimRight(addr, "/") to match them; Addr keeps that normalized value, which is also what a caller configuring the client sees. New test TestNew_TrailingSlashAddr_RequestPathHasNoDoubleSlash builds a client from a server URL with a trailing slash, asserts Addr lost it, and asserts the observed request path is /v1/secret/data/web1 with no "//" (it fails with //v1/... before this change). No other provider needs the same normalization: delinea, passwordstate and onepassword already trim in New, and awsec2 never concatenates a base URL with a path -- it parses its Endpoint as a complete URL and only sets the query string, so a trailing slash there cannot double. Validated with go build ./internal/credential/... and go test ./internal/credential/... -count=1.
