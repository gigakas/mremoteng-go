package vault_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/credential/vault"
)

func TestReadKV2_UnwrapsDataDataAndSendsToken(t *testing.T) {
	var gotPath, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("X-Vault-Token")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"data": map[string]string{"username": "alice", "password": "s3cret"},
			},
		})
	}))
	defer srv.Close()

	c := vault.New(srv.URL, "test-token")
	got, err := c.ReadKV2(context.Background(), "secret", "web1")
	if err != nil {
		t.Fatalf("ReadKV2: %v", err)
	}

	if gotPath != "/v1/secret/data/web1" {
		t.Errorf("request path = %q, want %q", gotPath, "/v1/secret/data/web1")
	}
	if gotToken != "test-token" {
		t.Errorf("X-Vault-Token = %q, want %q", gotToken, "test-token")
	}
	if got["username"] != "alice" || got["password"] != "s3cret" {
		t.Errorf("ReadKV2 = %+v, want username=alice password=s3cret", got)
	}
}

func TestReadKV1_NoExtraDataNesting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/secret/web1" {
			t.Errorf("request path = %q, want /v1/secret/web1", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{"username": "bob", "password": "hunter2"},
		})
	}))
	defer srv.Close()

	c := vault.New(srv.URL, "tok")
	got, err := c.ReadKV1(context.Background(), "secret", "web1")
	if err != nil {
		t.Fatalf("ReadKV1: %v", err)
	}
	if got["username"] != "bob" {
		t.Errorf("ReadKV1[username] = %q, want %q", got["username"], "bob")
	}
}

func TestLDAPDynamicCredential_ReturnsGeneratedCredential(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ldap/creds/dba" {
			t.Errorf("request path = %q, want /v1/ldap/creds/dba", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{"username": "v-token-dba-abc123", "password": "generated-pw"},
		})
	}))
	defer srv.Close()

	c := vault.New(srv.URL, "tok")
	got, err := c.LDAPDynamicCredential(context.Background(), "ldap", "dba")
	if err != nil {
		t.Fatalf("LDAPDynamicCredential: %v", err)
	}
	if got.Username != "v-token-dba-abc123" || got.Password != "generated-pw" {
		t.Errorf("LDAPDynamicCredential = %+v", got)
	}
}

func TestLDAPStaticCredential_ReturnsCurrentPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ldap/static-cred/svc-account" {
			t.Errorf("request path = %q, want /v1/ldap/static-cred/svc-account", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{"username": "svc-account", "password": "current-rotated-pw"},
		})
	}))
	defer srv.Close()

	c := vault.New(srv.URL, "tok")
	got, err := c.LDAPStaticCredential(context.Background(), "ldap", "svc-account")
	if err != nil {
		t.Fatalf("LDAPStaticCredential: %v", err)
	}
	if got.Password != "current-rotated-pw" {
		t.Errorf("LDAPStaticCredential.Password = %q, want %q", got.Password, "current-rotated-pw")
	}
}

func TestSSHOTP_PostsUsernameAndIPAndReturnsKey(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{"key": "123456"},
		})
	}))
	defer srv.Close()

	c := vault.New(srv.URL, "tok")
	otp, err := c.SSHOTP(context.Background(), "ssh", "otp_key_role", "deploy", "10.0.0.5")
	if err != nil {
		t.Fatalf("SSHOTP: %v", err)
	}
	if otp != "123456" {
		t.Errorf("SSHOTP = %q, want %q", otp, "123456")
	}
	if gotBody["username"] != "deploy" || gotBody["ip"] != "10.0.0.5" {
		t.Errorf("SSHOTP request body = %+v", gotBody)
	}
}

func TestReadKV2_NonOKStatus_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"errors":["permission denied"]}`))
	}))
	defer srv.Close()

	c := vault.New(srv.URL, "bad-token")
	if _, err := c.ReadKV2(context.Background(), "secret", "web1"); err == nil {
		t.Fatal("expected an error for a 403 response")
	}
}

func TestReadKV2_ResponseErrorsField_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"errors": []string{"no such secret"}})
	}))
	defer srv.Close()

	c := vault.New(srv.URL, "tok")
	if _, err := c.ReadKV2(context.Background(), "secret", "missing"); err == nil {
		t.Fatal("expected an error when the response body carries an errors array")
	}
}
