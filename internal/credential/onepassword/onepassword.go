// Package onepassword implements a client for the 1Password Connect
// server's REST API (a self-hosted read proxy in front of a 1Password
// account) rather than shelling out to the "op" CLI: the Connect API is
// the integration path 1Password documents for unattended/service
// access, where a CLI would need an interactively-unlocked session.
package onepassword

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mRemoteNG/mremoteng-go/internal/credential"
)

// Client talks to a 1Password Connect server using a Connect API token
// scoped to the vault(s) it needs to read.
type Client struct {
	// HTTPClient defaults to http.DefaultClient if nil.
	HTTPClient *http.Client
	// ConnectHost is the Connect server's base URL, e.g.
	// "https://connect.example.com".
	ConnectHost string
	Token       string
}

func New(connectHost, token string) *Client {
	return &Client{ConnectHost: strings.TrimRight(connectHost, "/"), Token: token}
}

func (c *Client) httpClient() *http.Client {
	return credential.HTTPClient(c.HTTPClient)
}

type itemResponse struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Fields []struct {
		ID      string `json:"id"`
		Label   string `json:"label"`
		Value   string `json:"value"`
		Purpose string `json:"purpose"`
	} `json:"fields"`
}

// Item fetches itemID from vaultID and extracts its username/password
// fields. 1Password's Connect API marks the login fields it generates
// itself with a "purpose" of USERNAME/PASSWORD; that takes priority, and
// a field labeled (or id'd) "username"/"password" is matched as a
// fallback for items using a different template.
func (c *Client) Item(ctx context.Context, vaultID, itemID string) (credential.Credential, error) {
	reqURL := fmt.Sprintf("%s/v1/vaults/%s/items/%s", c.ConnectHost, url.PathEscape(vaultID), url.PathEscape(itemID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return credential.Credential{}, fmt.Errorf("onepassword: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	body, status, err := credential.Do(c.httpClient(), req,
		fmt.Sprintf("onepassword: get item %s/%s", vaultID, itemID),
		fmt.Sprintf("onepassword: read response for item %s/%s", vaultID, itemID),
	)
	if err != nil {
		return credential.Credential{}, err
	}
	if err := credential.CheckStatus(status, http.StatusOK, body, fmt.Sprintf("onepassword: get item %s/%s", vaultID, itemID)); err != nil {
		return credential.Credential{}, err
	}

	var parsed itemResponse
	if err := credential.DecodeJSON(body, &parsed, fmt.Sprintf("onepassword: parse item %s/%s", vaultID, itemID)); err != nil {
		return credential.Credential{}, err
	}

	var out credential.Credential
	var usernameByLabel, passwordByLabel string
	for _, field := range parsed.Fields {
		switch strings.ToUpper(field.Purpose) {
		case "USERNAME":
			out.Username = field.Value
		case "PASSWORD":
			out.Password = field.Value
		}
		if credential.FieldIs(field.ID, field.Label, "username") {
			usernameByLabel = field.Value
		}
		if credential.FieldIs(field.ID, field.Label, "password") {
			passwordByLabel = field.Value
		}
	}
	if out.Username == "" {
		out.Username = usernameByLabel
	}
	if out.Password == "" {
		out.Password = passwordByLabel
	}
	return out, nil
}
