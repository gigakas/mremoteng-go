package passwordstate_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/credential/passwordstate"
)

func TestPassword_SendsAPIKeyAndReturnsFirstRecord(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("apikey")
		json.NewEncoder(w).Encode([]map[string]any{
			{"PasswordID": 7, "UserName": "svc-web1", "Password": "hunter2"},
		})
	}))
	defer srv.Close()

	c := passwordstate.New(srv.URL, "test-api-key")
	got, err := c.Password(context.Background(), 7)
	if err != nil {
		t.Fatalf("Password: %v", err)
	}
	if gotPath != "/api/passwords/7" {
		t.Errorf("request path = %q, want /api/passwords/7", gotPath)
	}
	if gotQuery != "test-api-key" {
		t.Errorf("apikey query param = %q, want %q", gotQuery, "test-api-key")
	}
	if got.Username != "svc-web1" || got.Password != "hunter2" {
		t.Errorf("Password = %+v, want Username=svc-web1 Password=hunter2", got)
	}
}

func TestPassword_EmptyArrayResponse_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	c := passwordstate.New(srv.URL, "test-api-key")
	if _, err := c.Password(context.Background(), 999); err == nil {
		t.Fatal("expected an error for an empty result array (password not found)")
	}
}

func TestPassword_NonOKStatus_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`Invalid API Key`))
	}))
	defer srv.Close()

	c := passwordstate.New(srv.URL, "wrong-key")
	if _, err := c.Password(context.Background(), 7); err == nil {
		t.Fatal("expected an error for a 401 response")
	}
}
