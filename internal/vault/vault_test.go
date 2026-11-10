package vault

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVault_SaveThenOpen_RoundTripsSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.json")
	created, err := Create(path, "master-password")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := created.Set("conn-1", "s3cr3t"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := created.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	opened, err := Open(path, "master-password")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if secret, ok := opened.Get("conn-1"); !ok || secret != "s3cr3t" {
		t.Fatalf("Get = %q/%v, want s3cr3t", secret, ok)
	}
}

func TestVault_File_IsOwnerOnlyAndNotPlaintext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.json")
	created, _ := Create(path, "master-password")
	_ = created.Set("conn-1", "s3cr3t")
	if err := created.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permissions = %o, want 600", perm)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(raw), "s3cr3t") {
		t.Error("secret appears in the file in plaintext")
	}
}

func TestOpen_WrongPassword_ReturnsErrWrongPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.json")
	created, _ := Create(path, "master-password")
	_ = created.Set("conn-1", "s3cr3t")
	if err := created.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := Open(path, "wrong-password"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("Open error = %v, want ErrWrongPassword", err)
	}
}

func TestOpen_TamperedPayload_IsDetected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.json")
	created, _ := Create(path, "master-password")
	_ = created.Set("conn-1", "s3cr3t")
	if err := created.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, _ := os.ReadFile(path)
	tampered := strings.Replace(string(raw), `"payload": "`, `"payload": "AAAA`, 1)
	if err := os.WriteFile(path, []byte(tampered), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := Open(path, "master-password"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("Open error = %v, want ErrWrongPassword for tampered data", err)
	}
}

func TestCreate_ShortPassword_IsRejected(t *testing.T) {
	if _, err := Create(filepath.Join(t.TempDir(), "secrets.json"), "short"); err == nil {
		t.Fatal("Create accepted a short master password")
	}
}

func TestDelete_RemovesSecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.json")
	created, _ := Create(path, "master-password")
	_ = created.Set("conn-1", "s3cr3t")
	created.Delete("conn-1")
	if _, ok := created.Get("conn-1"); ok {
		t.Fatal("secret was not deleted")
	}
}

func TestLock_ForgetsSecretsAndRejectsWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.json")
	created, _ := Create(path, "master-password")
	_ = created.Set("conn-1", "s3cr3t")
	created.Lock()
	if !created.Locked() {
		t.Error("Locked = false after Lock")
	}
	if _, ok := created.Get("conn-1"); ok {
		t.Error("Get returned a secret after Lock")
	}
	if err := created.Set("conn-2", "x"); !errors.Is(err, ErrLocked) {
		t.Errorf("Set after Lock = %v, want ErrLocked", err)
	}
}

func TestOpen_MissingFile_ReturnsError(t *testing.T) {
	if _, err := Open(filepath.Join(t.TempDir(), "absent.json"), "master-password"); err == nil {
		t.Fatal("Open succeeded without a file")
	}
}
