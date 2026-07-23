# Audit — Phase 3, Stage 3.7

- **Date (UTC)**: 2026-07-24
- **Agent**: claude-code
- **Audited stage**: External credential repositories
- **Commits covered**: uncommitted at audit time; see the stage 3.7
  implementation and closing-audit changelog fragments created alongside
  this file

## 1. Code quality

- New package `internal/credential` (five subpackages, one per
  provider): `vault` (HashiCorp Vault / OpenBao), `delinea` (Delinea
  Secret Server), `passwordstate` (Click Studios Passwordstate),
  `onepassword` (1Password Connect), `awsec2` (AWS EC2 address
  resolution). Each is a small, self-contained HTTP client with no
  shared base beyond the one `credential.Credential{Username, Password}`
  result type in the parent package — justified by how little these
  APIs actually share (different auth schemes: bearer token issued by
  OAuth2 password grant, static API key in a query string, static bearer
  token, a pre-issued Vault token, and hand-rolled AWS SigV4; different
  response shapes: nested JSON, a flat JSON array, a fields array, and
  XML).
- **`vault`**: maps directly onto the four engines
  `connection.VaultOpenbaoSecretEngine` already names (`Kv`,
  `LdapDynamic`, `LdapStatic`, `SSHOTP`) — `ReadKV2`/`ReadKV1`,
  `LDAPDynamicCredential`/`LDAPStaticCredential`, `SSHOTP`. KV v2's extra
  `data.data` response nesting (vs. KV v1's flat `data`) is unwrapped
  internally so callers don't need to know which version they're
  reading.
- **`delinea`**: OAuth2 password-grant authentication with token
  caching (`tokenFor`/`authenticate`, guarded by a mutex) and a
  single automatic re-authenticate-and-retry on a 401 from the secret
  endpoint, covering the case of a token that expired between calls.
  Field extraction matches `fieldName`/`slug` case-insensitively against
  "username"/"password" — Secret Server's own field naming isn't
  perfectly consistent across secret templates.
- **`passwordstate`**: the simplest client (one query-string API key,
  one JSON array response) — implements only the single-password lookup
  endpoint, not Passwordstate's separate list-query/search endpoints,
  since nothing in `connection.ConnectionValues` models a *list* lookup.
- **`onepassword`**: uses the 1Password **Connect** server's REST API,
  not the `op` CLI — deliberately, since Connect is what 1Password
  documents for unattended/service access (a CLI needs an
  interactively-unlocked vault session, which doesn't fit an app
  resolving a credential to open a remote session). Field extraction
  prefers Connect's own `purpose` tag (`USERNAME`/`PASSWORD`, set on
  fields it generates itself) and falls back to matching a field's
  `id`/`label` for items using a different template.
- **`awsec2`**: hand-rolled AWS Signature Version 4 signing
  (`crypto/hmac` + `crypto/sha256`, no SDK dependency — see the file's
  own doc comment for the "no new external dependency" justification)
  for a single, fixed EC2 Query API call, `DescribeInstances`.
  `InstanceAddress` prefers the instance's public IP, falling back to
  private — EC2 instances in a private subnet have no public IP, and
  mRemoteNG's own EC2 integration is documented as connecting to
  whichever address is actually reachable. The signing host is always
  derived from the actual request URL (`Endpoint` override or the real
  `ec2.{region}.amazonaws.com`), not a hardcoded string, so the
  canonical-headers `host` value is guaranteed consistent with whatever
  the request actually goes to — verified in
  `TestInstanceAddress_RequestIsSigV4Signed`, which asserts the exact
  `Credential=.../eu-central-1/ec2/aws4_request` scope and
  `SignedHeaders=host;x-amz-date` against a fixed clock, not just "no
  error".
- **The unresolved field-mapping question, stated plainly rather than
  guessed past**: `internal/connection`'s model has fields for the Vault
  provider specifically and for AWS address resolution, but *no*
  per-provider secret-identifier field for Delinea/Passwordstate/
  1Password — the original C# app's convention for which existing field
  (if any) is repurposed to hold that ID isn't recoverable from this Go
  repository. Every client in this package therefore takes its secret
  identifier as an explicit parameter rather than reading
  `connection.ConnectionInfo` directly; wiring that mapping is left to
  whoever integrates this package with real
  `connection.ExternalCredentialProvider` values, with the authoritative
  C# source in hand. Documented in `credential.go`'s package doc, not
  buried in a subpackage.
- No duplication, no function over ~50 lines, no discarded errors.

## 2. Performance

Not applicable: each client makes one (or two, for Delinea's
authenticate-then-fetch) HTTP round trip per call — network-bound,
called only when a connection actually needs to resolve a credential,
never a hot path.

## 3. Architecture

- `internal/credential` is fully disjoint from `internal/ui` and
  `internal/protocol`, exactly as the blueprint's parallelism note
  describes — no changes to either package in this stage, no UI wiring
  (explicitly out of scope; see the blueprint's "picker in the
  properties panel happens inside 3.4, not in 3.7" note — 3.4 is
  already closed without adding one, an honest gap recorded in that
  stage's own audit).
- No new external dependencies: all five clients use only
  `net/http`, `encoding/json`, `encoding/xml`, and (for `awsec2`)
  `crypto/hmac`/`crypto/sha256` from the standard library.
- No impact on any other package's public contracts.

## 4. Evidence

- `./scripts/check.sh`: **green**.
- `./scripts/smoke.sh`: **green**.
- Unlike every other Phase 3 stage, **this one has no visual-verification
  gap at all** — it has no UI. All behavior (request shape, auth
  headers, response parsing, error handling) is directly assertable in a
  headless test without Fyne involved.
- New tests, all against real `httptest.Server` instances driving actual
  HTTP round trips (not mocked at the transport level):
  - `internal/credential/vault` (7 tests): KV v2's `data.data` unwrap,
    KV v1's flat shape, both LDAP credential kinds, SSHOTP's POST
    body and response, a non-200 error, and a 200-with-`errors`-array
    response.
  - `internal/credential/delinea` (4 tests): full
    authenticate-then-fetch flow, **token caching verified by asserting
    the auth endpoint is hit exactly once across two `Secret` calls**,
    wrong-credentials error, and the `BaseURL` trailing-slash trim.
  - `internal/credential/passwordstate` (3 tests): API key sent as a
    query param, empty-array-means-not-found, non-200 error.
  - `internal/credential/onepassword` (3 tests): `purpose`-tag priority,
    label-match fallback when no purpose tags are present, non-200
    error.
  - `internal/credential/awsec2` (6 tests): public-IP preference,
    private-IP fallback, AWS XML error-response parsing, **the SigV4
    Authorization header's exact credential scope and signed-headers
    list against a fixed clock** (not just "request succeeded"),
    session-token inclusion in both the header and the signed-headers
    list, and the empty-reservation-set not-found case.

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- **No UI wiring** — by design (see section 3); a picker in the
  properties panel and the actual `connection.ConnectionInfo` field
  mapping described in section 1 are both future work.
- **The Delinea/Passwordstate/1Password field-mapping gap** (section 1)
  is the main open question for whoever does that wiring — needs the
  original C# `ExternalConnectors` source to confirm, not another guess
  from this repository alone.
- **`vault`'s `InsecureSkipVerify` option and `delinea`/`passwordstate`/
  `onepassword`'s lack of one** — Vault/OpenBao self-signed internal
  deployments are common enough that the original app exposes a "trust
  this certificate" toggle for it; the other three providers are more
  commonly deployed behind a real CA-issued certificate (Secret Server
  Cloud, Passwordstate's own recommended reverse-proxy setup, 1Password
  Connect typically behind a proper ingress), so this was added only
  where it seemed likely to matter, not uniformly across all five — a
  judgment call, not a confirmed requirement.
- **`vault`'s SSHOTP client obtains the OTP but doesn't install/verify
  the corresponding Vault SSH helper on the target host** — matches the
  scope of "obtain a credential", the same as every other client here;
  actually *using* an OTP is a `internal/protocol/ssh` concern, not
  this package's.
- **This phase (Phase 3) is now all 7 stages done** — Phase 3's exit
  criteria and the top-level README are addressed in a separate wrap-up
  commit, not folded into this stage's own audit.
- Commit the working tree — not done without explicit request.
