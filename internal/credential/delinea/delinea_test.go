package delinea_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/credential/delinea"
)

func newTestServer(t *testing.T, wantUsername, wantPassword string) (*httptest.Server, *int32) {
	t.Helper()
	var authCalls int32
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&authCalls, 1)
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse token request form: %v", err)
		}
		if r.Form.Get("username") != wantUsername || r.Form.Get("password") != wantPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"access_token": "test-bearer-token"})
	})
	mux.HandleFunc("/api/v1/secrets/42", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-bearer-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"id": 42,
			"items": []map[string]any{
				{"fieldName": "Username", "itemValue": "svc-web1", "isPassword": false},
				{"fieldName": "Password", "itemValue": "hunter2", "isPassword": true},
				{"fieldName": "Notes", "itemValue": "unrelated", "isPassword": false},
			},
		})
	})
	srv := httptest.NewServer(mux)
	return srv, &authCalls
}

func TestSecret_AuthenticatesThenFetchesAndExtractsUsernamePassword(t *testing.T) {
	srv, authCalls := newTestServer(t, "svc-account", "svc-password")
	defer srv.Close()

	c := delinea.New(srv.URL, "svc-account", "svc-password")
	got, err := c.Secret(context.Background(), 42)
	if err != nil {
		t.Fatalf("Secret: %v", err)
	}
	if got.Username != "svc-web1" || got.Password != "hunter2" {
		t.Errorf("Secret = %+v, want Username=svc-web1 Password=hunter2", got)
	}
	if atomic.LoadInt32(authCalls) != 1 {
		t.Errorf("authenticate was called %d times, want 1", atomic.LoadInt32(authCalls))
	}
}

func TestSecret_ReusesCachedToken_DoesNotReauthenticateOnSecondCall(t *testing.T) {
	srv, authCalls := newTestServer(t, "svc-account", "svc-password")
	defer srv.Close()

	c := delinea.New(srv.URL, "svc-account", "svc-password")
	if _, err := c.Secret(context.Background(), 42); err != nil {
		t.Fatalf("first Secret call: %v", err)
	}
	if _, err := c.Secret(context.Background(), 42); err != nil {
		t.Fatalf("second Secret call: %v", err)
	}
	if got := atomic.LoadInt32(authCalls); got != 1 {
		t.Errorf("authenticate was called %d times across two Secret calls, want 1 (token should be cached)", got)
	}
}

func TestSecret_WrongCredentials_ReturnsError(t *testing.T) {
	srv, _ := newTestServer(t, "svc-account", "svc-password")
	defer srv.Close()

	c := delinea.New(srv.URL, "svc-account", "wrong-password")
	if _, err := c.Secret(context.Background(), 42); err == nil {
		t.Fatal("expected an error when the OAuth2 token request is rejected")
	}
}

func TestNew_TrimsTrailingSlashFromBaseURL(t *testing.T) {
	c := delinea.New("https://secretserver.example.com/SecretServer/", "u", "p")
	if got := c.BaseURL; got != "https://secretserver.example.com/SecretServer" {
		t.Errorf("BaseURL = %q, want no trailing slash", got)
	}
}
