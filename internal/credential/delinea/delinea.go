// Package delinea implements a client for Delinea Secret Server
// (formerly Thycotic Secret Server): OAuth2 password-grant
// authentication followed by a REST secret lookup.
package delinea

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/mRemoteNG/mremoteng-go/internal/credential"
)

// Client authenticates to a Secret Server instance with a service
// account's username/password and caches the resulting bearer token for
// reuse across Secret calls, re-authenticating once it's rejected.
type Client struct {
	// HTTPClient defaults to http.DefaultClient if nil.
	HTTPClient *http.Client
	// BaseURL is the Secret Server root, e.g.
	// "https://secretserver.example.com/SecretServer".
	BaseURL  string
	Username string
	Password string

	mu    sync.Mutex
	token string
}

func New(baseURL, username, password string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Username: username, Password: password}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// authenticate obtains and caches a bearer token via OAuth2 password
// grant. Not safe to call concurrently with itself -- callers go through
// tokenFor, which holds c.mu.
func (c *Client) authenticate(ctx context.Context) (string, error) {
	form := url.Values{
		"username":   {c.Username},
		"password":   {c.Password},
		"grant_type": {"password"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("delinea: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("delinea: token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("delinea: read token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("delinea: authenticate: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("delinea: parse token response: %w", err)
	}
	if parsed.AccessToken == "" {
		return "", fmt.Errorf("delinea: token response had no access_token")
	}
	return parsed.AccessToken, nil
}

func (c *Client) tokenFor(ctx context.Context, forceRefresh bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && !forceRefresh {
		return c.token, nil
	}
	token, err := c.authenticate(ctx)
	if err != nil {
		return "", err
	}
	c.token = token
	return token, nil
}

type secretResponse struct {
	ID    int `json:"id"`
	Items []struct {
		FieldName  string `json:"fieldName"`
		Slug       string `json:"slug"`
		ItemValue  string `json:"itemValue"`
		IsPassword bool   `json:"isPassword"`
	} `json:"items"`
}

// Secret fetches secretID and extracts its Username/Password fields --
// matched case-insensitively against each item's field name or slug,
// since Secret Server templates name these fields consistently but not
// with a single fixed casing across template types.
func (c *Client) Secret(ctx context.Context, secretID int) (credential.Credential, error) {
	body, status, err := c.getSecret(ctx, secretID, false)
	if err != nil {
		return credential.Credential{}, err
	}
	if status == http.StatusUnauthorized {
		body, status, err = c.getSecret(ctx, secretID, true)
		if err != nil {
			return credential.Credential{}, err
		}
	}
	if status != http.StatusOK {
		return credential.Credential{}, fmt.Errorf("delinea: get secret %d: HTTP %d: %s", secretID, status, string(body))
	}

	var parsed secretResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return credential.Credential{}, fmt.Errorf("delinea: parse secret %d: %w", secretID, err)
	}

	var out credential.Credential
	for _, item := range parsed.Items {
		switch {
		case fieldIs(item.FieldName, item.Slug, "username"):
			out.Username = item.ItemValue
		case fieldIs(item.FieldName, item.Slug, "password"):
			out.Password = item.ItemValue
		}
	}
	return out, nil
}

func fieldIs(fieldName, slug, want string) bool {
	return strings.EqualFold(fieldName, want) || strings.EqualFold(slug, want)
}

func (c *Client) getSecret(ctx context.Context, secretID int, forceRefresh bool) ([]byte, int, error) {
	token, err := c.tokenFor(ctx, forceRefresh)
	if err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/v1/secrets/%d", c.BaseURL, secretID), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("delinea: build secret request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("delinea: get secret %d: %w", secretID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("delinea: read secret %d response: %w", secretID, err)
	}
	return body, resp.StatusCode, nil
}
