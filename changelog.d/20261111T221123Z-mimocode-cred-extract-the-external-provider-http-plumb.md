---
timestamp: 2026-11-11T22:11:23Z
agent: mimocode-cred
files:
  - internal/credential/http.go
  - internal/credential/fields.go
  - internal/credential/http_test.go
  - internal/credential/fields_test.go
  - internal/credential/credential.go
  - internal/credential/vault/vault.go
  - internal/credential/delinea/delinea.go
  - internal/credential/passwordstate/passwordstate.go
  - internal/credential/onepassword/onepassword.go
  - internal/credential/awsec2/awsec2.go
---

Extract the external-provider HTTP plumbing into shared internal/credential helpers

The five REST clients under internal/credential each carried their own copy of the same request plumbing: the nil-http.Client fallback to http.DefaultClient, the send/ReadAll/status-check/unmarshal block, and (in delinea and onepassword) a byte-identical fieldIs predicate. They now call five helpers in internal/credential (http.go, fields.go): HTTPClient(given *http.Client) *http.Client, InsecureHTTPClient() *http.Client, Do(client *http.Client, req *http.Request, doErrContext, readErrContext string) ([]byte, int, error), CheckStatus(status, want int, body []byte, errContext string) error, DecodeJSON(body []byte, out any, errContext string) error and FieldIs(first, second, want string) bool. This is a pure refactor: every provider keeps its exact error wording, its own status-code policy (vault, delinea token auth, passwordstate and onepassword accept only 200; delinea's secret GET returns body+status to its caller for the 401 re-auth retry; awsec2 still parses the EC2 XML error envelope) and its own request building. InsecureSkipVerify stays isolated: only vault.Client.httpClient calls InsecureHTTPClient, gated on the existing documented field, so no other provider gains the capability. Do is given a per-provider transport-error and read-error context string precisely so the previous messages survive verbatim. Helpers are additive; no exported provider signature changed and no existing provider test was edited. New unit tests cover the helper contracts (nil fallback, insecure transport wiring, both error contexts, status/JSON formatting, FieldIs). Validated with go build ./internal/credential/... and go test ./internal/credential/... -count=1.
