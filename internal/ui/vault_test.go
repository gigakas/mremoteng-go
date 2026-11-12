package ui

import (
	"github.com/mRemoteNG/mremoteng-go/internal/vault"
	"path/filepath"
	"strings"
	"testing"
)

func TestVault_StoreUseAndLockSecrets(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.vaultPath = filepath.Join(t.TempDir(), "secrets.json")
	s.createVault("master-password")

	profile := sampleProfile(t, "web", "", "SSH", "web.example.com", 22)
	panel := s.newSessionPanel(profile)
	panel.password.SetText("s3cr3t")
	panel.requestStorePassword()
	if !strings.Contains(s.status.Text, "stored") {
		t.Fatalf("status = %q, want a storage confirmation", s.status.Text)
	}

	panel.password.SetText("")
	if got := panel.passwordFor(); got != "s3cr3t" {
		t.Errorf("passwordFor = %q, want the stored secret", got)
	}
	panel.password.SetText("typed")
	if got := panel.passwordFor(); got != "typed" {
		t.Errorf("passwordFor = %q, want the typed password to win", got)
	}

	reopened, err := vault.Open(s.vaultPath, "master-password")
	if err != nil {
		t.Fatalf("reopen vault: %v", err)
	}
	if secret, ok := reopened.Get(profile.ID); !ok || secret != "s3cr3t" {
		t.Fatalf("stored secret = %q/%v", secret, ok)
	}

	s.lockVault()
	panel.password.SetText("")
	if got := panel.passwordFor(); got != "" {
		t.Errorf("passwordFor = %q after locking, want empty", got)
	}
	panel.requestStorePassword()
	if !strings.Contains(s.status.Text, "Unlock the credential vault") {
		t.Errorf("status = %q, want an unlock hint", s.status.Text)
	}
}

func TestVault_WrongPasswordIsReported(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.vaultPath = filepath.Join(t.TempDir(), "secrets.json")
	s.createVault("master-password")
	s.unlockVault("nope")
	if !strings.Contains(s.status.Text, "Could not unlock") {
		t.Errorf("status = %q, want an unlock error", s.status.Text)
	}
}
