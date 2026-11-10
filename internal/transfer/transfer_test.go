package transfer

import (
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	connectionxml "github.com/mRemoteNG/mremoteng-go/internal/serialize/xml"
)

// sampleXML builds a real v2.8 file with one folder and one SSH connection.
func sampleXML(t *testing.T, password string) []byte {
	t.Helper()
	root, err := connection.NewRootInfo()
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	folder, err := connection.NewContainerInfo()
	if err != nil {
		t.Fatalf("folder: %v", err)
	}
	folder.Base().Raw.Name = "Production"
	if err := root.AddChild(folder); err != nil {
		t.Fatalf("add folder: %v", err)
	}
	info, err := connection.NewConnectionInfoWithID("22222222-2222-4222-8222-222222222222")
	if err != nil {
		t.Fatalf("connection: %v", err)
	}
	info.Raw.Name = "web01"
	info.Raw.Hostname = "web01.example.com"
	info.Raw.Port = 22
	info.Raw.Username = "deploy"
	info.Raw.Password = "secret"
	info.Raw.Protocol = connection.ProtocolSSH2
	if err := folder.AddChild(info); err != nil {
		t.Fatalf("add connection: %v", err)
	}
	data, err := connectionxml.Serialize(&connectionxml.Document{Root: root}, connectionxml.SerializeOptions{Password: []byte(password)})
	if err != nil {
		t.Fatalf("serialize fixture: %v", err)
	}
	return data
}

func TestReadXML_FolderTree_MapsSupportedConnections(t *testing.T) {
	preview, err := ReadXML(sampleXML(t, ""), "")
	if err != nil {
		t.Fatalf("ReadXML: %v", err)
	}
	if len(preview.Profiles) != 1 {
		t.Fatalf("profiles = %d, want 1", len(preview.Profiles))
	}
	profile := preview.Profiles[0]
	if profile.Name != "web01" || profile.Host != "web01.example.com" || profile.Port != 22 {
		t.Errorf("profile = %+v", profile)
	}
	if profile.Protocol != profiles.SSH {
		t.Errorf("protocol = %q, want SSH", profile.Protocol)
	}
	if profile.Folder != "Production" {
		t.Errorf("folder = %q, want Production", profile.Folder)
	}
	if profile.Username != "deploy" {
		t.Errorf("username = %q", profile.Username)
	}
	if profile.ID == "" {
		t.Error("imported profile has no ID")
	}
	if len(preview.Folders) != 1 || preview.Folders[0] != "Production" {
		t.Errorf("folders = %v", preview.Folders)
	}
}

func TestReadXML_PasswordIsCountedAndNeverCarried(t *testing.T) {
	preview, err := ReadXML(sampleXML(t, ""), "")
	if err != nil {
		t.Fatalf("ReadXML: %v", err)
	}
	if preview.Dropped != 1 {
		t.Errorf("dropped = %d, want the source password counted", preview.Dropped)
	}
	if preview.Profiles[0].Username == "secret" {
		t.Error("password leaked into another field")
	}
}

func TestReadXML_UnsupportedProtocolIsSkipped(t *testing.T) {
	root, _ := connection.NewRootInfo()
	info, _ := connection.NewConnectionInfoWithID("55555555-5555-4555-8555-555555555555")
	info.Raw.Name = "vnc01"
	info.Raw.Hostname = "vnc01.example.com"
	info.Raw.Port = 5900
	info.Raw.Protocol = connection.ProtocolVNC
	_ = root.AddChild(info)
	data, err := connectionxml.Serialize(&connectionxml.Document{Root: root}, connectionxml.SerializeOptions{})
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	preview, err := ReadXML(data, "")
	if err != nil {
		t.Fatalf("ReadXML: %v", err)
	}
	if len(preview.Profiles) != 0 || preview.Unsupported != 1 {
		t.Fatalf("preview = %+v, want the VNC row skipped", preview)
	}
}

func TestReadXML_ProtectedFileNeedsPassword(t *testing.T) {
	_, err := ReadXML(sampleXML(t, "master-password"), "")
	if err == nil {
		t.Fatal("ReadXML succeeded on a protected file without a password")
	}
	preview, err := ReadXML(sampleXML(t, "master-password"), "master-password")
	if err != nil {
		t.Fatalf("ReadXML with password: %v", err)
	}
	if len(preview.Profiles) != 1 {
		t.Fatalf("profiles = %d, want 1", len(preview.Profiles))
	}
}

func TestWriteXML_RoundTripsThroughTheSerializer(t *testing.T) {
	profile := profiles.Profile{ID: "33333333-3333-4333-8333-333333333333", Name: "web01", Folder: "Production",
		Protocol: profiles.SSH, Host: "web01.example.com", Port: 22, Username: "deploy"}
	data, err := WriteXML([]profiles.Profile{profile}, []string{"Production"}, "")
	if err != nil {
		t.Fatalf("WriteXML: %v", err)
	}
	document, err := connectionxml.Deserialize(data, connectionxml.Options{})
	if err != nil {
		t.Fatalf("deserialize export: %v", err)
	}
	if document.Metadata.ConfVersion != "2.8" {
		t.Errorf("version = %q, want 2.8", document.Metadata.ConfVersion)
	}
	if nodes := document.Root.Descendants(); len(nodes) != 2 {
		t.Fatalf("nodes = %d, want a folder and a connection", len(nodes))
	}
	preview, err := ReadXML(data, "")
	if err != nil {
		t.Fatalf("ReadXML of our own export: %v", err)
	}
	if len(preview.Profiles) != 1 || preview.Profiles[0].Folder != "Production" {
		t.Fatalf("round trip = %+v", preview)
	}
}

func TestWriteCSV_RoundTripsThroughTheSerializer(t *testing.T) {
	profile := profiles.Profile{ID: "44444444-4444-4444-8444-444444444444", Name: "desk01",
		Protocol: profiles.RDP, Host: "desk01.example.com", Port: 3389}
	data, err := WriteCSV([]profiles.Profile{profile}, nil)
	if err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	preview, err := ReadCSV(data)
	if err != nil {
		t.Fatalf("ReadCSV of our own export: %v", err)
	}
	if len(preview.Profiles) != 1 || preview.Profiles[0].Name != "desk01" {
		t.Fatalf("preview = %+v", preview)
	}
	if preview.Profiles[0].Protocol != profiles.RDP {
		t.Errorf("protocol = %q, want RDP", preview.Profiles[0].Protocol)
	}
}

func TestSanitizeSegment_RejectsPathsAndKeepsNames(t *testing.T) {
	if got := sanitizeSegment(" Prod/Web "); got != "Prod-Web" {
		t.Errorf("sanitizeSegment = %q", got)
	}
	if got := sanitizeSegment(".."); got != "Imported folder" {
		t.Errorf("sanitizeSegment = %q, want the fallback", got)
	}
}
