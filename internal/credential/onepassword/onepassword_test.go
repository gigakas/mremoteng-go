package onepassword_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/credential/onepassword"
)

func TestItem_PurposeTaggedFields_TakePriority(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "item1",
			"title": "web1",
			"fields": []map[string]any{
				{"id": "username", "label": "username", "value": "alice", "purpose": "USERNAME"},
				{"id": "password", "label": "password", "value": "s3cret", "purpose": "PASSWORD"},
				{"id": "notesPlain", "label": "notes", "value": "unrelated", "purpose": "NOTES"},
			},
		})
	}))
	defer srv.Close()

	c := onepassword.New(srv.URL, "connect-token")
	got, err := c.Item(context.Background(), "vault1", "item1")
	if err != nil {
		t.Fatalf("Item: %v", err)
	}

	if gotPath != "/v1/vaults/vault1/items/item1" {
		t.Errorf("request path = %q, want /v1/vaults/vault1/items/item1", gotPath)
	}
	if gotAuth != "Bearer connect-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer connect-token")
	}
	if got.Username != "alice" || got.Password != "s3cret" {
		t.Errorf("Item = %+v, want Username=alice Password=s3cret", got)
	}
}

func TestItem_NoPurposeTags_FallsBackToLabelMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id": "item1",
			"fields": []map[string]any{
				{"id": "f1", "label": "Username", "value": "bob"},
				{"id": "f2", "label": "Password", "value": "hunter2"},
			},
		})
	}))
	defer srv.Close()

	c := onepassword.New(srv.URL, "tok")
	got, err := c.Item(context.Background(), "vault1", "item1")
	if err != nil {
		t.Fatalf("Item: %v", err)
	}
	if got.Username != "bob" || got.Password != "hunter2" {
		t.Errorf("Item = %+v, want Username=bob Password=hunter2 (via label fallback)", got)
	}
}

func TestItem_NonOKStatus_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"item not found"}`))
	}))
	defer srv.Close()

	c := onepassword.New(srv.URL, "tok")
	if _, err := c.Item(context.Background(), "vault1", "missing"); err == nil {
		t.Fatal("expected an error for a 404 response")
	}
}
