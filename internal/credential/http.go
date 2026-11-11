package credential

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// The helpers in this file are the request plumbing the provider
// subpackages share. They are implementation detail for those clients,
// not part of what a caller outside the credential tree configures:
// Credential remains the package's reason for existing.

// HTTPClient returns c, or http.DefaultClient when c is nil. It backs the
// optional "HTTPClient defaults to http.DefaultClient if nil" field every
// provider exposes.
func HTTPClient(c *http.Client) *http.Client {
	if c != nil {
		return c
	}
	return http.DefaultClient
}

// InsecureHTTPClient returns a client that skips TLS certificate
// verification. Only a provider whose configuration carries an explicit
// "accept untrusted certificates" opt-in may call this (currently only
// vault.Client.InsecureSkipVerify); every other provider must keep the
// verifying default.
func InsecureHTTPClient() *http.Client {
	return &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}} //nolint:gosec // opt-in by the calling provider, documented on its Client field.
}

// Do sends req with client and reads the whole response body, returning
// the body and the status code: whether a status is an error stays the
// caller's decision, since providers disagree on the accepted code. A
// transport failure is wrapped with doErrContext and a body read failure
// with readErrContext, so each provider keeps its own error wording.
func Do(client *http.Client, req *http.Request, doErrContext, readErrContext string) ([]byte, int, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", doErrContext, err)
	}
	// The body is read to completion below, so a Close error could only
	// report a connection-teardown problem that does not affect the data
	// the caller receives; a genuine read failure is reported instead.
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", readErrContext, err)
	}
	return body, resp.StatusCode, nil
}

// CheckStatus returns nil when status is want and otherwise an error
// formatted "<errContext>: HTTP <status>: <raw body>" — the shape every
// provider already reported rather than a typed error, because the raw
// body is the only diagnostic some of these APIs return.
func CheckStatus(status, want int, body []byte, errContext string) error {
	if status == want {
		return nil
	}
	return fmt.Errorf("%s: HTTP %d: %s", errContext, status, string(body))
}

// DecodeJSON unmarshals body into out, wrapping a failure as
// "<errContext>: <cause>".
func DecodeJSON(body []byte, out any, errContext string) error {
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%s: %w", errContext, err)
	}
	return nil
}
