// Package passwordstate implements a client for Click Studios
// Passwordstate's REST API (single-password lookup by API key).
package passwordstate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/mRemoteNG/mremoteng-go/internal/credential"
)

// Client reads a single password record by ID using a Passwordstate API
// key -- either a Password List API key (scoped to one list) or the
// system-wide API key, Passwordstate does not distinguish between them
// at this endpoint.
type Client struct {
	// HTTPClient defaults to http.DefaultClient if nil.
	HTTPClient *http.Client
	// BaseURL is the Passwordstate root, e.g. "https://passwordstate.example.com".
	BaseURL string
	APIKey  string
}

func New(baseURL, apiKey string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), APIKey: apiKey}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

type passwordRecord struct {
	PasswordID int    `json:"PasswordID"`
	UserName   string `json:"UserName"`
	Password   string `json:"Password"`
}

// Password fetches passwordID. The API returns a JSON array with exactly
// one record for a single-ID lookup by design (its list-query form
// returns many; this client only implements the single-record lookup).
func (c *Client) Password(ctx context.Context, passwordID int) (credential.Credential, error) {
	reqURL := fmt.Sprintf("%s/api/passwords/%d?apikey=%s", c.BaseURL, passwordID, url.QueryEscape(c.APIKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return credential.Credential{}, fmt.Errorf("passwordstate: build request: %w", err)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return credential.Credential{}, fmt.Errorf("passwordstate: get password %d: %w", passwordID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return credential.Credential{}, fmt.Errorf("passwordstate: read response for password %d: %w", passwordID, err)
	}
	if resp.StatusCode != http.StatusOK {
		return credential.Credential{}, fmt.Errorf("passwordstate: get password %d: HTTP %d: %s", passwordID, resp.StatusCode, string(body))
	}

	var records []passwordRecord
	if err := json.Unmarshal(body, &records); err != nil {
		return credential.Credential{}, fmt.Errorf("passwordstate: parse response for password %d: %w", passwordID, err)
	}
	if len(records) == 0 {
		return credential.Credential{}, fmt.Errorf("passwordstate: password %d not found", passwordID)
	}

	return credential.Credential{Username: records[0].UserName, Password: records[0].Password}, nil
}
