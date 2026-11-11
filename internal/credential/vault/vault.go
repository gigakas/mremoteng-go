// Package vault implements a client for the four secret engines
// connection.VaultOpenbaoSecretEngine names (Kv, LdapDynamic, LdapStatic,
// SSHOTP), against either HashiCorp Vault or OpenBao — the two are wire
// compatible, hence the model's combined "VaultOpenbao" naming.
package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mRemoteNG/mremoteng-go/internal/credential"
)

// Client talks to a Vault/OpenBao HTTP API using a pre-issued token — this
// package does not implement any of Vault's own login/auth methods, only
// secret reads once a caller already holds a token.
type Client struct {
	// HTTPClient defaults to http.DefaultClient if nil.
	HTTPClient *http.Client
	// Addr is the server's base URL, e.g. "https://vault.example.com:8200".
	Addr string
	// Token is sent as the X-Vault-Token header on every request.
	Token string
	// InsecureSkipVerify disables TLS certificate verification -- for
	// self-signed internal deployments, matching the original app's own
	// "accept untrusted cert" option for these integrations. Off by
	// default.
	InsecureSkipVerify bool
}

// New builds a client for the Vault/OpenBao server at addr. A trailing
// slash is trimmed because do concatenates Addr with a path that already
// starts with one, which would otherwise request "//v1/...".
func New(addr, token string) *Client {
	return &Client{Addr: strings.TrimRight(addr, "/"), Token: token}
}

func (c *Client) httpClient() *http.Client {
	if c.InsecureSkipVerify && c.HTTPClient == nil {
		return credential.InsecureHTTPClient()
	}
	return credential.HTTPClient(c.HTTPClient)
}

type apiResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []string        `json:"errors"`
}

func (c *Client) do(ctx context.Context, method, path string, body any) (json.RawMessage, error) {
	url := c.Addr + path

	var reqBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("vault: encode request body: %w", err)
		}
		reqBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("vault: build request: %w", err)
	}
	req.Header.Set("X-Vault-Token", c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	opContext := fmt.Sprintf("%s %s", method, path)
	respBody, status, err := credential.Do(c.httpClient(), req, "vault: request "+opContext, "vault: read response body")
	if err != nil {
		return nil, err
	}
	if err := credential.CheckStatus(status, http.StatusOK, respBody, "vault: "+opContext); err != nil {
		return nil, err
	}

	var parsed apiResponse
	if err := credential.DecodeJSON(respBody, &parsed, "vault: parse response"); err != nil {
		return nil, err
	}
	if len(parsed.Errors) > 0 {
		return nil, fmt.Errorf("vault: %s %s: %v", method, path, parsed.Errors)
	}
	return parsed.Data, nil
}

// ReadKV2 reads a KV version 2 secret at mount/path (engine
// connection.VaultEngineKV), returning its string-valued keys. KV v2
// nests the actual secret one level under "data" in the response
// ("data":{"data":{...},"metadata":{...}}) — that nesting is handled
// here so callers don't need to know about it.
func (c *Client) ReadKV2(ctx context.Context, mount, path string) (map[string]string, error) {
	data, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/%s/data/%s", mount, path), nil)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Data map[string]string `json:"data"`
	}
	if err := credential.DecodeJSON(data, &wrapper, fmt.Sprintf("vault: parse KV v2 secret at %s/%s", mount, path)); err != nil {
		return nil, err
	}
	return wrapper.Data, nil
}

// ReadKV1 reads a KV version 1 secret at mount/path -- no extra "data"
// nesting, unlike KV v2.
func (c *Client) ReadKV1(ctx context.Context, mount, path string) (map[string]string, error) {
	data, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/%s/%s", mount, path), nil)
	if err != nil {
		return nil, err
	}
	var values map[string]string
	if err := credential.DecodeJSON(data, &values, fmt.Sprintf("vault: parse KV v1 secret at %s/%s", mount, path)); err != nil {
		return nil, err
	}
	return values, nil
}

// LDAPDynamicCredential requests a freshly generated LDAP credential for
// role (engine connection.VaultEngineLDAPDynamic) -- each call returns a
// new, distinct password with its own lease.
func (c *Client) LDAPDynamicCredential(ctx context.Context, mount, role string) (credential.Credential, error) {
	data, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/%s/creds/%s", mount, role), nil)
	if err != nil {
		return credential.Credential{}, err
	}
	return decodeCredential(data, "vault: parse LDAP dynamic credential for role "+role)
}

// LDAPStaticCredential reads the current value of a static LDAP account's
// password (engine connection.VaultEngineLDAPStatic) -- the account
// itself is rotated on a schedule managed by Vault/OpenBao, not by this
// call.
func (c *Client) LDAPStaticCredential(ctx context.Context, mount, role string) (credential.Credential, error) {
	data, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/%s/static-cred/%s", mount, role), nil)
	if err != nil {
		return credential.Credential{}, err
	}
	return decodeCredential(data, "vault: parse LDAP static credential for role "+role)
}

// SSHOTP requests a one-time-use SSH password (engine
// connection.VaultEngineSSHOTP) for username@ip via role -- the target
// host must have the corresponding Vault SSH helper installed to accept
// it; this call only obtains the OTP itself.
func (c *Client) SSHOTP(ctx context.Context, mount, role, username, ip string) (string, error) {
	data, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/v1/%s/creds/%s", mount, role), map[string]string{
		"username": username,
		"ip":       ip,
	})
	if err != nil {
		return "", err
	}
	var wrapper struct {
		Key string `json:"key"`
	}
	if err := credential.DecodeJSON(data, &wrapper, fmt.Sprintf("vault: parse SSH OTP for role %s", role)); err != nil {
		return "", err
	}
	if wrapper.Key == "" {
		return "", fmt.Errorf("vault: SSH OTP response for role %s had no key", role)
	}
	return wrapper.Key, nil
}

func decodeCredential(data json.RawMessage, errContext string) (credential.Credential, error) {
	var wrapper struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := credential.DecodeJSON(data, &wrapper, errContext); err != nil {
		return credential.Credential{}, err
	}
	return credential.Credential{Username: wrapper.Username, Password: wrapper.Password}, nil
}
