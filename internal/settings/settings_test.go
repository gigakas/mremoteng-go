package settings_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/settings"
)

func TestLoad_MissingFile_ReturnsDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.json")

	got, err := settings.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := settings.Default()
	if *got != *want {
		t.Errorf("Load(missing) = %+v, want Default() = %+v", got, want)
	}
}

func TestSaveThenLoad_RoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "settings.json")

	s := settings.Default()
	s.WindowWidth = 1280
	s.WindowHeight = 800
	s.LastConnectionsFile = "C:/connections.xml"
	s.Theme = "dark"

	if err := s.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := settings.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if *got != *s {
		t.Errorf("Load after Save = %+v, want %+v", got, s)
	}
}

func TestLoad_MalformedFile_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := settings.Load(path); err == nil {
		t.Fatal("expected an error for a malformed settings file")
	}
}

func TestDefaultPath_ReturnsANonEmptyPathUnderTheUserConfigDir(t *testing.T) {
	path, err := settings.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if path == "" {
		t.Error("DefaultPath returned an empty path")
	}
	if filepath.Base(path) != "settings.json" {
		t.Errorf("DefaultPath = %q, want it to end in settings.json", path)
	}
}
