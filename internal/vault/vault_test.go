package vault

import (
	"encoding/json"
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

func TestOpen_FormatFields_AreValidated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.json")
	vault, err := Create(path, "master-password")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := vault.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(map[string]any){
		"unknown version":   func(f map[string]any) { f["version"] = 99 },
		"unknown algorithm": func(f map[string]any) { f["algorithm"] = "rot13" },
		"broken salt":       func(f map[string]any) { f["salt"] = "***" },
		"broken nonce":      func(f map[string]any) { f["nonce"] = "***" },
		"broken payload":    func(f map[string]any) { f["payload"] = "***" },
	}
	for name, mutate := range cases {
		var file map[string]any
		if err := json.Unmarshal(original, &file); err != nil {
			t.Fatal(err)
		}
		mutate(file)
		edited, _ := json.Marshal(file) // a map of plain values always marshals
		if err := os.WriteFile(path, edited, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(path, "master-password"); err == nil {
			t.Errorf("%s: Open succeeded, want an error", name)
		} else if errors.Is(err, ErrWrongPassword) && name != "broken payload" {
			t.Errorf("%s: error = %v, want a format error, not a password error", name, err)
		}
	}
}

func TestOpen_NotJSON_ReturnsDecodeError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path, "master-password"); err == nil {
		t.Fatal("Open accepted a file that is not JSON")
	}
}

func TestSave_AfterLock_ReturnsErrLocked(t *testing.T) {
	vault, err := Create(filepath.Join(t.TempDir(), "secrets.json"), "master-password")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	vault.Lock()
	if err := vault.Save(); !errors.Is(err, ErrLocked) {
		t.Fatalf("Save error = %v, want ErrLocked", err)
	}
}

func TestSet_EmptyID_IsRejected(t *testing.T) {
	vault, err := Create(filepath.Join(t.TempDir(), "secrets.json"), "master-password")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := vault.Set("", "x"); err == nil {
		t.Fatal("Set accepted an empty connection id")
	}
}

func TestSave_SecondSave_UsesAFreshNonce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.json")
	vault, err := Create(path, "master-password")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	nonces := map[string]bool{}
	for i := 0; i < 2; i++ {
		if err := vault.Save(); err != nil {
			t.Fatalf("Save: %v", err)
		}
		raw, _ := os.ReadFile(path) // the file was just written by Save
		var file fileFormat
		if err := json.Unmarshal(raw, &file); err != nil {
			t.Fatal(err)
		}
		if nonces[file.Nonce] {
			t.Fatal("two saves reused a nonce, which breaks AES-GCM")
		}
		nonces[file.Nonce] = true
	}
}

func TestSave_UnwritableDirectory_ReportsAndKeepsOldFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.json")
	vault, err := Create(path, "master-password")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := vault.Set("a", "1"); err != nil {
		t.Fatal(err)
	}
	if err := vault.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	before, _ := os.ReadFile(path) // written by the Save above

	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) // let TempDir remove it
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	if err := vault.Save(); err == nil {
		t.Fatal("Save succeeded in a read-only directory")
	}
	after, _ := os.ReadFile(path) // the failed Save must not have touched it
	if string(before) != string(after) {
		t.Error("a failed Save changed the existing vault file")
	}
}

func TestExists_FilePresence_IsReported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.json")
	if Exists(path) {
		t.Error("Exists = true for a missing file")
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if !Exists(path) {
		t.Error("Exists = false for a present file")
	}
}
