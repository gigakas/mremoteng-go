---
timestamp: 2026-11-14T22:25:34Z
agent: mimocode-e1
files:
  - internal/credential
  - blueprint/backlog.md
  - blueprint/linux-preview.md
  - auditory/linux-l3-20261115-mimocode-a1.md
  - docs/linux-preview-usability.md
---

Delete the credential providers nothing imported

internal/credential held five external providers (Vault/OpenBao, 1Password, Delinea, PasswordSTATE, AWS EC2) plus shared HTTP plumbing: 1655 lines, unit-tested in isolation and imported by nothing since the day they were written. L9's remaining item was to decide between wiring them and deleting them. Wiring needs a resolution step at connect time, a picker per connection and a threat model, and no real provider is reachable from this machine to verify any of it, which is the rule this repository applies to security-relevant code. They are deleted instead, and the stage now says what it ships: an encrypted local vault, with a password stored only when the user asks. The packages stay in git history if they are ever wanted. The L3 audit, the usability runbook and the backlog were updated in the same wave. Validated with ./scripts/check.sh.
