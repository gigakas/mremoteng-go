package ui_test

import (
	"path/filepath"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/ui"
)

func TestSaveThenLoadConnectionsFile_RoundTrips(t *testing.T) {
	root, err := connection.NewRootInfo()
	if err != nil {
		t.Fatalf("NewRootInfo: %v", err)
	}
	folder, err := connection.NewContainerInfo()
	if err != nil {
		t.Fatalf("NewContainerInfo: %v", err)
	}
	folder.Base().Raw.Name = "Servers"
	if err := root.AddChild(folder); err != nil {
		t.Fatalf("AddChild(folder): %v", err)
	}
	conn, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	conn.Raw.Name = "web1"
	conn.Raw.Hostname = "web1.example.com"
	conn.Raw.Protocol = connection.ProtocolSSH2
	conn.Raw.Username = "alice"
	conn.Raw.Password = "s3cret"
	if err := folder.AddChild(conn); err != nil {
		t.Fatalf("AddChild(conn): %v", err)
	}

	path := filepath.Join(t.TempDir(), "connections.xml")
	password := []byte("test-password")

	if err := ui.SaveConnectionsFile(path, root, password); err != nil {
		t.Fatalf("SaveConnectionsFile: %v", err)
	}

	loaded, err := ui.LoadConnectionsFile(path, password)
	if err != nil {
		t.Fatalf("LoadConnectionsFile: %v", err)
	}

	children := loaded.Children()
	if len(children) != 1 {
		t.Fatalf("loaded root has %d children, want 1", len(children))
	}
	loadedFolder, ok := children[0].(*connection.ContainerInfo)
	if !ok {
		t.Fatalf("loaded child = %T, want *connection.ContainerInfo", children[0])
	}
	if got := loadedFolder.Base().Raw.Name; got != "Servers" {
		t.Errorf("loaded folder name = %q, want %q", got, "Servers")
	}

	grandchildren := loadedFolder.Children()
	if len(grandchildren) != 1 {
		t.Fatalf("loaded folder has %d children, want 1", len(grandchildren))
	}
	loadedConn, ok := grandchildren[0].(*connection.ConnectionInfo)
	if !ok {
		t.Fatalf("loaded grandchild = %T, want *connection.ConnectionInfo", grandchildren[0])
	}
	if got := loadedConn.Raw.Hostname; got != "web1.example.com" {
		t.Errorf("loaded hostname = %q, want %q", got, "web1.example.com")
	}
	if got := loadedConn.Raw.Username; got != "alice" {
		t.Errorf("loaded username = %q, want %q", got, "alice")
	}
	if got := loadedConn.Raw.Password; got != "s3cret" {
		t.Errorf("loaded password = %q, want %q (round-tripped through real AES-256-GCM/PBKDF2 encryption, Phase 1)", got, "s3cret")
	}
}

func TestLoadConnectionsFile_WrongPassword_ReturnsError(t *testing.T) {
	root, err := connection.NewRootInfo()
	if err != nil {
		t.Fatalf("NewRootInfo: %v", err)
	}
	path := filepath.Join(t.TempDir(), "connections.xml")
	if err := ui.SaveConnectionsFile(path, root, []byte("right-password")); err != nil {
		t.Fatalf("SaveConnectionsFile: %v", err)
	}

	if _, err := ui.LoadConnectionsFile(path, []byte("wrong-password")); err == nil {
		t.Fatal("expected an error when loading with the wrong password")
	}
}

func TestLoadConnectionsFile_MissingFile_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.xml")
	if _, err := ui.LoadConnectionsFile(path, nil); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}
