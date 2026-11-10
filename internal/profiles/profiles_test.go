package profiles

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleProfile(t *testing.T) Profile {
	t.Helper()
	id, err := NewID()
	if err != nil {
		t.Fatalf("NewID: %v", err)
	}
	return Profile{ID: id, Name: "Prod web", Protocol: SSH, Host: "web.example.com", Port: 22, Username: "deploy"}
}

func TestNewID_RepeatedCalls_ReturnsDistinctValidIdentifiers(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id, err := NewID()
		if err != nil {
			t.Fatalf("NewID: %v", err)
		}
		if len(id) != 36 || strings.Count(id, "-") != 4 || id[14] != '4' {
			t.Fatalf("NewID() = %q, want RFC 4122 version 4", id)
		}
		if seen[id] {
			t.Fatalf("NewID() returned duplicate %q", id)
		}
		seen[id] = true
	}
}

func TestProfile_Validate_TableDrivenResults(t *testing.T) {
	t.Parallel()
	valid := sampleProfile(t)
	tests := []struct {
		name    string
		mutate  func(*Profile)
		wantErr error
	}{
		{name: "valid ssh", mutate: func(*Profile) {}},
		{name: "valid rdp", mutate: func(p *Profile) { p.Protocol = RDP; p.Port = 3389 }},
		{name: "name with spaces", mutate: func(p *Profile) { p.Name = "Prod database primary" }},
		{name: "ipv6 literal", mutate: func(p *Profile) { p.Host = "[2001:db8::1]" }},
		{name: "missing name", mutate: func(p *Profile) { p.Name = "   " }, wantErr: ErrMissingName},
		{name: "name with control rune", mutate: func(p *Profile) { p.Name = "bad\x00name" }, wantErr: ErrInvalidName},
		{name: "unknown protocol", mutate: func(p *Profile) { p.Protocol = "FTP" }, wantErr: ErrInvalidProtocol},
		{name: "missing host", mutate: func(p *Profile) { p.Host = "" }, wantErr: ErrMissingHost},
		{name: "host with space", mutate: func(p *Profile) { p.Host = "web example.com" }, wantErr: ErrInvalidHost},
		{name: "host with scheme", mutate: func(p *Profile) { p.Host = "ssh://web" }, wantErr: ErrInvalidHost},
		{name: "host with port", mutate: func(p *Profile) { p.Host = "web:22" }, wantErr: ErrInvalidHost},
		{name: "host with user", mutate: func(p *Profile) { p.Host = "deploy@web" }, wantErr: ErrInvalidHost},
		{name: "host option injection", mutate: func(p *Profile) { p.Host = "-oProxyCommand=sh" }, wantErr: ErrInvalidHost},
		{name: "port zero", mutate: func(p *Profile) { p.Port = 0 }, wantErr: ErrInvalidPort},
		{name: "port too high", mutate: func(p *Profile) { p.Port = 65536 }, wantErr: ErrInvalidPort},
		{name: "username with space", mutate: func(p *Profile) { p.Username = "de ploy" }, wantErr: ErrInvalidUsername},
		{name: "username option injection", mutate: func(p *Profile) { p.Username = "-lroot" }, wantErr: ErrInvalidUsername},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			profile := valid
			test.mutate(&profile)
			err := profile.Validate()
			if test.wantErr == nil {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Validate() = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestFolderSegments_TableDrivenResults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		folder  string
		want    []string
		wantErr bool
	}{
		{folder: "", want: nil},
		{folder: "Production", want: []string{"Production"}},
		{folder: "Production/Web", want: []string{"Production", "Web"}},
		{folder: "/Production", wantErr: true},
		{folder: "Production/", wantErr: true},
		{folder: "Production//Web", wantErr: true},
		{folder: "Production/../Secrets", wantErr: true},
		{folder: "./Production", wantErr: true},
		{folder: "Production/\x00bad", wantErr: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.folder, func(t *testing.T) {
			t.Parallel()
			got, err := FolderSegments(test.folder)
			if test.wantErr {
				if !errors.Is(err, ErrInvalidFolder) {
					t.Fatalf("FolderSegments(%q) error = %v, want ErrInvalidFolder", test.folder, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("FolderSegments(%q) error = %v", test.folder, err)
			}
			if len(got) != len(test.want) {
				t.Fatalf("FolderSegments(%q) = %v, want %v", test.folder, got, test.want)
			}
			for i := range got {
				if got[i] != test.want[i] {
					t.Fatalf("FolderSegments(%q) = %v, want %v", test.folder, got, test.want)
				}
			}
		})
	}
}

func TestResolve_ClosestAncestorWinsPerField(t *testing.T) {
	profile := Profile{Name: "web", Folder: "Team/Prod/Web", Protocol: SSH, Host: "web", Port: 22,
		InheritUsername: true, InheritPort: true}
	defaults := []FolderDefault{
		{Path: "Team", Username: "team-user", Port: 2200},
		{Path: "Team/Prod", Port: 2222},
	}
	resolved := Resolve(profile, defaults)
	if resolved.Username != "team-user" {
		t.Errorf("username = %q, want the Team default", resolved.Username)
	}
	if resolved.Port != 2222 {
		t.Errorf("port = %d, want the closer Prod default", resolved.Port)
	}
	if profile.Username != "" || profile.Port != 22 {
		t.Error("Resolve mutated the input profile")
	}
}

func TestResolve_FlagsOff_KeepsLocalValues(t *testing.T) {
	profile := Profile{Name: "web", Folder: "Team", Protocol: SSH, Host: "web", Port: 2222, Username: "local"}
	defaults := []FolderDefault{{Path: "Team", Username: "team-user", Port: 2200}}
	resolved := Resolve(profile, defaults)
	if resolved.Username != "local" || resolved.Port != 2222 {
		t.Errorf("resolved = %+v, want the local values", resolved)
	}
}

func TestProfile_Validate_InheritedPortMayBeUnset(t *testing.T) {
	profile := Profile{Name: "web", Protocol: SSH, Host: "web", Port: 0, InheritPort: true}
	if err := profile.Validate(); err != nil {
		t.Fatalf("Validate with InheritPort = %v, want nil", err)
	}
	profile.InheritPort = false
	if err := profile.Validate(); !errors.Is(err, ErrInvalidPort) {
		t.Fatalf("Validate = %v, want ErrInvalidPort", err)
	}
}

func TestStore_LoadVersion1_MigratesWithoutDefaults(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "profiles.json")
	data := `{"version":1,"folders":["Team"],"profiles":[{"id":"a","name":"web","protocol":"SSH","host":"web","port":22}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	store := NewStore(path)
	document, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(document.Profiles) != 1 || len(document.Defaults) != 0 {
		t.Fatalf("document = %+v, want the v1 data without defaults", document)
	}
	if err := store.Save(document); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(raw), `"version": 2`) {
		t.Errorf("rewritten store does not use version 2: %s", raw)
	}
}

func TestStore_DefaultsRoundTripAndValidation(t *testing.T) {
	t.Parallel()
	store := NewStore(filepath.Join(t.TempDir(), "profiles.json"))
	document := Document{Folders: []string{"Team"}, Defaults: []FolderDefault{{Path: "Team", Username: "team-user", Port: 2222}}}
	if err := store.Save(document); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Defaults) != 1 || loaded.Defaults[0].Username != "team-user" || loaded.Defaults[0].Port != 2222 {
		t.Fatalf("defaults = %+v", loaded.Defaults)
	}
	bad := Document{Folders: []string{"Team"}, Defaults: []FolderDefault{{Path: "/bad"}}}
	if err := store.Save(bad); !errors.Is(err, ErrInvalidFolder) {
		t.Fatalf("Save bad default = %v, want ErrInvalidFolder", err)
	}
}

func TestDefaultPort_SupportedProtocols_ReturnsConventionalPort(t *testing.T) {
	t.Parallel()
	if got := DefaultPort(SSH); got != 22 {
		t.Errorf("DefaultPort(SSH) = %d, want 22", got)
	}
	if got := DefaultPort(RDP); got != 3389 {
		t.Errorf("DefaultPort(RDP) = %d, want 3389", got)
	}
	if got := DefaultPort("FTP"); got != 0 {
		t.Errorf("DefaultPort(unknown) = %d, want 0", got)
	}
}

func TestStore_LoadMissingFile_ReturnsEmptyDocument(t *testing.T) {
	t.Parallel()
	store := NewStore(filepath.Join(t.TempDir(), "nested", "profiles.json"))
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got.Profiles) != 0 || len(got.Folders) != 0 {
		t.Fatalf("Load() = %+v, want an empty document", got)
	}
}

func TestStore_SaveThenLoad_RoundTripsProfiles(t *testing.T) {
	t.Parallel()
	store := NewStore(filepath.Join(t.TempDir(), "profiles.json"))
	want := []Profile{sampleProfile(t)}
	want[0].Name = "  Padded name  "
	want[0].Username = `DOMAIN\deploy`
	folders := []string{"Production", "Production/Web"}

	if err := store.Save(Document{Profiles: want, Folders: folders}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Profiles) != 1 {
		t.Fatalf("Load() returned %d profiles, want 1", len(got.Profiles))
	}
	if got.Profiles[0].Name != want[0].Name || got.Profiles[0].Host != want[0].Host || got.Profiles[0].Username != want[0].Username {
		t.Fatalf("round trip = %+v, want %+v", got.Profiles[0], want[0])
	}
	if len(got.Folders) != 2 || got.Folders[1] != "Production/Web" {
		t.Fatalf("folders = %v, want the saved paths", got.Folders)
	}
}

func TestStore_Save_CreatesOwnerOnlyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")
	if err := NewStore(path).Save(Document{Profiles: []Profile{sampleProfile(t)}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("store permissions = %o, want 600", perm)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("store directory contains %d entries, want only the store file", len(entries))
	}
}

func TestStore_SaveTwice_ReplacesPreviousContent(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "profiles.json")
	store := NewStore(path)
	first := sampleProfile(t)
	if err := store.Save(Document{Profiles: []Profile{first}}); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	if err := store.Save(Document{Profiles: []Profile{}}); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Profiles) != 0 {
		t.Fatalf("Load() = %v, want empty after replacement", got.Profiles)
	}
}

func TestStore_LoadCorruptFile_ReturnsError(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "profiles.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := NewStore(path).Load(); err == nil {
		t.Fatal("Load() error = nil, want decode error")
	}
}

func TestStore_LoadUnsupportedVersion_ReturnsError(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "profiles.json")
	if err := os.WriteFile(path, []byte(`{"version":99,"profiles":[]}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := NewStore(path).Load(); !errors.Is(err, ErrUnsupportedStore) {
		t.Fatalf("Load() error = %v, want ErrUnsupportedStore", err)
	}
}

func TestStore_SaveInvalidFolder_ReturnsError(t *testing.T) {
	t.Parallel()
	store := NewStore(filepath.Join(t.TempDir(), "profiles.json"))
	err := store.Save(Document{Folders: []string{"Production", "Production"}, Profiles: []Profile{}})
	if !errors.Is(err, ErrDuplicateID) {
		t.Fatalf("Save() error = %v, want ErrDuplicateID for duplicated folder", err)
	}
	if err := store.Save(Document{Folders: []string{"/bad"}, Profiles: []Profile{}}); !errors.Is(err, ErrInvalidFolder) {
		t.Fatalf("Save() error = %v, want ErrInvalidFolder", err)
	}
}

func TestStore_LoadInvalidProfile_ReturnsError(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "profiles.json")
	data := `{"version":1,"profiles":[{"id":"dup","name":"a","protocol":"SSH","host":"h","port":22},{"id":"dup","name":"b","protocol":"RDP","host":"h","port":3389}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := NewStore(path).Load(); !errors.Is(err, ErrInvalidStore) {
		t.Fatalf("Load() error = %v, want ErrInvalidStore", err)
	}
}

func TestStore_SaveInvalidProfiles_DoesNotWriteFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "profiles.json")
	store := NewStore(path)
	invalid := sampleProfile(t)
	invalid.Host = ""
	if err := store.Save(Document{Profiles: []Profile{invalid}}); !errors.Is(err, ErrMissingHost) {
		t.Fatalf("Save() error = %v, want ErrMissingHost", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("store file exists after rejected save (stat error %v)", err)
	}
}

func TestDefaultPath_UsesConfigurationDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if want := filepath.Join(dir, "mremoteng-go", "profiles.json"); path != want {
		t.Errorf("DefaultPath() = %q, want %q", path, want)
	}
}
