package ui

import (
	"errors"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	"strings"
	"testing"
)

func TestSave_ValidProfile_AddsTreeEntryTabAndPersists(t *testing.T) {
	store := &fakeStore{}
	s := newTestScreen(t, store)
	fillConfig(s, "Production web", "Production/Web", "web.example.com", "22")
	test.Tap(s.props.save)

	if store.saves != 1 || len(store.profiles) != 1 {
		t.Fatalf("store = %d saves, %d profiles; want 1/1", store.saves, len(store.profiles))
	}
	saved := store.profiles[0]
	if saved.Name != "Production web" || saved.Folder != "Production/Web" || saved.Host != "web.example.com" {
		t.Fatalf("saved = %+v", saved)
	}
	if _, known := s.index.profiles[connectionUID(saved.ID)]; !known {
		t.Error("saved connection is missing from the tree")
	}
	if _, known := s.index.folders[folderUID("Production/Web")]; !known {
		t.Error("folder path is missing from the tree")
	}
	if children := s.index.children[folderUID("Production")]; len(children) != 1 || children[0] != folderUID("Production/Web") {
		t.Errorf("nested folder children = %v, want the Web folder under Production", children)
	}
	if tab, open := s.openTabs[saved.ID]; !open || tab.Text != saved.Name {
		t.Errorf("session tab = %v, want a tab named %q", open, saved.Name)
	}
	if s.selected != saved.ID || s.baseline != saved {
		t.Errorf("selection = %q, baseline = %+v", s.selected, s.baseline)
	}
	if s.isDirty() {
		t.Error("Config panel still dirty after a successful save")
	}
	if s.counts.Text != "1 / 2" {
		t.Errorf("counts = %q, want %q (connections / folders)", s.counts.Text, "1 / 2")
	}
}

func TestSave_InvalidInput_ReportsProblemWithoutWriting(t *testing.T) {
	tests := []struct {
		name     string
		nameText string
		folder   string
		host     string
		port     string
		want     string
	}{
		{name: "missing name", host: "web", port: "22", want: "name is required"},
		{name: "missing host", nameText: "web", port: "22", want: "host is required"},
		{name: "non numeric port", nameText: "web", host: "web", port: "ssh", want: "Port must be a whole number"},
		{name: "port out of range", nameText: "web", host: "web", port: "70000", want: "port must be between"},
		{name: "invalid folder", nameText: "web", folder: "../Secrets", host: "web", port: "22", want: "folder path is invalid"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeStore{}
			s := newTestScreen(t, store)
			fillConfig(s, tc.nameText, tc.folder, tc.host, tc.port)
			test.Tap(s.props.save)

			if store.saves != 0 {
				t.Fatalf("store saved %d times, want 0", store.saves)
			}
			if !strings.Contains(s.status.Text, tc.want) {
				t.Errorf("status = %q, want substring %q", s.status.Text, tc.want)
			}
			if s.status.Importance != widget.DangerImportance {
				t.Error("validation problem is not styled as an error")
			}
			if !s.isDirty() {
				t.Error("invalid edits were discarded")
			}
		})
	}
}

func TestSave_StoreFailure_KeepsEditsAndReports(t *testing.T) {
	store := &fakeStore{saveErr: errors.New("disk full")}
	s := newTestScreen(t, store)
	fillConfig(s, "Prod", "", "web", "22")
	test.Tap(s.props.save)

	if len(s.ws.profiles) != 0 {
		t.Fatalf("in-memory profiles = %d, want 0 after a failed save", len(s.ws.profiles))
	}
	if !strings.Contains(s.status.Text, "Could not save") || !strings.Contains(s.status.Text, "disk full") {
		t.Errorf("status = %q, want save error", s.status.Text)
	}
	if s.props.name.Text != "Prod" || !s.isDirty() {
		t.Error("edits were lost after a failed save")
	}
}

func TestFilter_HidesConnectionsAndEmptyFolders(t *testing.T) {
	store := &fakeStore{profiles: []profiles.Profile{
		sampleProfile(t, "Production web", "Production", "SSH", "web.example.com", 22),
		sampleProfile(t, "Office desktop", "", "RDP", "desk.example.com", 3389),
	}}
	s := newTestScreen(t, store)
	if s.index.connections != 2 {
		t.Fatalf("tree connections = %d, want 2", s.index.connections)
	}

	s.filter.SetText("desktop")
	if s.index.connections != 1 {
		t.Errorf("filtered connections = %d, want 1", s.index.connections)
	}
	if _, known := s.index.folders[folderUID("Production")]; known {
		t.Error("folder without matches is still in the tree")
	}

	s.filter.SetText("office.example")
	if s.index.connections != 0 {
		t.Errorf("filtered connections = %d, want 0 for an unknown host", s.index.connections)
	}
}

func TestDelete_Confirmed_RemovesProfileTabAndTreeEntry(t *testing.T) {
	existing := sampleProfile(t, "Production web", "", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{existing}}
	s := newTestScreen(t, store)
	s.confirm = (&fakeConfirm{accept: true}).ask

	s.tree.Select(connectionUID(existing.ID))
	test.Tap(s.props.delete)

	if len(store.profiles) != 0 {
		t.Fatalf("store = %+v, want no profiles", store.profiles)
	}
	if len(s.ws.profiles) != 0 || s.index.connections != 0 {
		t.Errorf("ui still holds %d profiles", len(s.ws.profiles))
	}
	if _, open := s.openTabs[existing.ID]; open {
		t.Error("session tab survived the deletion")
	}
	if !strings.Contains(s.status.Text, "Deleted") {
		t.Errorf("status = %q, want deletion confirmation", s.status.Text)
	}
}

func TestDelete_NotConfirmed_KeepsProfile(t *testing.T) {
	existing := sampleProfile(t, "Production web", "", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{existing}}
	s := newTestScreen(t, store)
	s.confirm = (&fakeConfirm{accept: false}).ask

	s.tree.Select(connectionUID(existing.ID))
	test.Tap(s.props.delete)

	if store.saves != 0 || len(store.profiles) != 1 {
		t.Fatalf("store = %+v after declined delete, want unchanged", store.profiles)
	}
}

func TestLoadFailure_DisablesWritesAndReportsProblem(t *testing.T) {
	store := &fakeStore{loadErr: errors.New("permission denied")}
	s := newTestScreen(t, store)
	if !s.props.save.Disabled() || !s.props.delete.Disabled() {
		t.Error("Config buttons stay enabled although the store could not be read")
	}
	if !strings.Contains(s.status.Text, "Could not read") {
		t.Errorf("status = %q, want load failure", s.status.Text)
	}

	fillConfig(s, "Prod", "", "web", "22")
	test.Tap(s.props.save)
	if store.saves != 0 {
		t.Fatalf("store saved %d times despite a load failure", store.saves)
	}
}

func TestProtocolChange_UntouchedPort_UsesProtocolDefault(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.props.proto.SetSelected(string(profiles.RDP))
	if s.props.port.Text != "3389" {
		t.Errorf("port = %q after switching to RDP, want 3389", s.props.port.Text)
	}

	s.props.port.SetText("3390")
	s.props.proto.SetSelected(string(profiles.SSH))
	if s.props.port.Text != "3390" {
		t.Errorf("port = %q after switching to SSH, want the typed port", s.props.port.Text)
	}
}

func TestRevert_UnsavedChanges_RestoresSelectedProfile(t *testing.T) {
	existing := sampleProfile(t, "Production web", "", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{existing}})
	s.tree.Select(connectionUID(existing.ID))
	s.props.name.SetText("Changed")
	test.Tap(s.props.revert)

	if s.props.name.Text != existing.Name || s.isDirty() {
		t.Errorf("Config = %q, dirty = %v; want the stored values back", s.props.name.Text, s.isDirty())
	}
}

func TestSave_ProfileFolder_IsPersistedWithTheTree(t *testing.T) {
	store := &fakeStore{}
	s := newTestScreen(t, store)
	fillConfig(s, "Production web", "Production/Web", "web.example.com", "22")
	test.Tap(s.props.save)

	if len(store.folders) != 2 {
		t.Fatalf("stored folders = %v, want Production and Production/Web", store.folders)
	}
}

func TestRequireWritable_ReadOnlyStore_ReportsProblemAndBlocks(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.ws.writable = false
	if s.requireWritable("Saving") {
		t.Fatal("requireWritable allowed a write on a read-only store")
	}
	want := "Saving is disabled because the existing file could not be read."
	if s.status.Text != want {
		t.Errorf("status = %q, want %q", s.status.Text, want)
	}
}

func TestRequireWritable_WritableStore_AllowsAndKeepsStatus(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.setStatus("ready")
	if !s.requireWritable("Saving") {
		t.Fatal("requireWritable blocked a writable store")
	}
	if s.status.Text != "ready" {
		t.Errorf("status = %q, want it left untouched", s.status.Text)
	}
}
