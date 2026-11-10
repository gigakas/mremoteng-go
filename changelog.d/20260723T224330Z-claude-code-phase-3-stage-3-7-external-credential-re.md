---
timestamp: 2026-07-23T22:43:30Z
agent: claude-code
files:
  - auditory/phase3-stage7-20260724-claude-code.md
  - blueprint/phase-3-ui.md
  - internal/credential/awsec2/awsec2.go
  - internal/credential/awsec2/awsec2_test.go
  - internal/credential/credential.go
  - internal/credential/delinea/delinea.go
  - internal/credential/delinea/delinea_test.go
  - internal/credential/onepassword/onepassword.go
  - internal/credential/onepassword/onepassword_test.go
  - internal/credential/passwordstate/passwordstate.go
  - internal/credential/passwordstate/passwordstate_test.go
  - internal/credential/vault/vault.go
  - internal/credential/vault/vault_test.go
---

Phase 3 stage 3.7: external credential repositories

Add internal/credential (Credential{Username,Password} result type) and five provider subpackages ported from mRemoteNG's ExternalConnectors: vault (HashiCorp Vault/OpenBao -- KV v1/v2, LDAP dynamic/static, SSH OTP, matching connection.VaultOpenbaoSecretEngine's four engines), delinea (Secret Server, OAuth2 password grant with cached/auto-refreshed bearer token), passwordstate (Click Studios, API-key single-password lookup), onepassword (1Password Connect REST API, not the op CLI, since Connect is what 1Password documents for unattended access), and awsec2 (EC2 DescribeInstances address resolution via hand-rolled AWS SigV4 signing, no SDK dependency). No UI wiring: connection.ConnectionInfo has no per-provider secret-identifier field for Delinea/Passwordstate/1Password recoverable from this repo, so every client takes its identifier as an explicit parameter rather than guessing a field mapping; documented as a follow-up in credential.go's package doc. Fully disjoint from internal/ui and internal/protocol, no new external dependencies (stdlib net/http, encoding/json, encoding/xml, crypto/hmac, crypto/sha256 only).
