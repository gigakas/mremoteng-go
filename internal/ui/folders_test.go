package ui

import (
	"fyne.io/fyne/v2/test"
	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	"strings"
	"testing"
)

func TestAddFolder_EmptyFolder_IsPersistedAndShown(t *testing.T) {
	store := &fakeStore{}
	s := newTestScreen(t, store)
	s.addFolder("Production/Web")

	if len(store.folders) != 2 || store.folders[1] != "Production/Web" {
		t.Fatalf("stored folders = %v, want the path and its parent", store.folders)
	}
	if _, known := s.index.folders[folderUID("Production/Web")]; !known {
		t.Error("empty folder is missing from the tree")
	}
	if !strings.Contains(s.status.Text, "created") {
		t.Errorf("status = %q, want a creation notice", s.status.Text)
	}
	if s.selectedUID == folderUID("Production/Web") {
		t.Error("new folder was auto-selected, which scrolls the tree")
	}
	s.addFolder("Production/Web")
	if len(store.folders) != 2 || !strings.Contains(s.status.Text, "already exists") {
		t.Errorf("duplicate folder was persisted: %v (%q)", store.folders, s.status.Text)
	}
}

func TestAddFolder_InvalidPath_ReportsProblemWithoutWriting(t *testing.T) {
	store := &fakeStore{}
	s := newTestScreen(t, store)
	s.addFolder("../escape")
	if store.saves != 0 {
		t.Fatalf("store saved %d times for an invalid folder", store.saves)
	}
	if !strings.Contains(s.status.Text, "Cannot create folder") {
		t.Errorf("status = %q, want a folder error", s.status.Text)
	}
}

func TestRenameFolder_MovesConnectionsAndDescendants(t *testing.T) {
	connection := sampleProfile(t, "Web", "Production/Web", "SSH", "web.example.com", 22)
	store := &fakeStore{
		profiles: []profiles.Profile{connection},
		folders:  []string{"Production", "Production/Web"},
	}
	s := newTestScreen(t, store)
	s.renameFolder("Production", "Clientes")

	if len(store.profiles) != 1 || store.profiles[0].Folder != "Clientes/Web" {
		t.Fatalf("profiles = %+v, want the folder rewritten", store.profiles)
	}
	if len(store.folders) != 2 || store.folders[0] != "Clientes" || store.folders[1] != "Clientes/Web" {
		t.Fatalf("folders = %v, want the whole subtree renamed", store.folders)
	}
	if _, known := s.index.folders[folderUID("Clientes/Web")]; !known {
		t.Error("renamed folder is missing from the tree")
	}
}

func TestRenameFolder_InvalidTargets_AreRejected(t *testing.T) {
	store := &fakeStore{folders: []string{"Production", "Ops"}}
	s := newTestScreen(t, store)
	s.renameFolder("Production", "Production/Inner")
	if !strings.Contains(s.status.Text, "into itself") {
		t.Errorf("status = %q, want a self-nesting rejection", s.status.Text)
	}
	s.renameFolder("Production", "Ops")
	if !strings.Contains(s.status.Text, "already exists") {
		t.Errorf("status = %q, want an existing-folder rejection", s.status.Text)
	}
	if store.saves != 0 {
		t.Fatalf("store saved %d times for rejected renames", store.saves)
	}
}

func TestDeleteFolder_RemovesConnectionsAndTabs(t *testing.T) {
	inside := sampleProfile(t, "Web", "Production/Web", "SSH", "web.example.com", 22)
	outside := sampleProfile(t, "Desk", "", "RDP", "desk.example.com", 3389)
	store := &fakeStore{
		profiles: []profiles.Profile{inside, outside},
		folders:  []string{"Production", "Production/Web"},
	}
	s := newTestScreen(t, store)
	s.confirm = (&fakeConfirm{accept: true}).ask
	s.tree.Select(connectionUID(inside.ID))
	s.tree.Select(folderUID("Production"))
	s.requestDeleteFolder()

	if len(store.profiles) != 1 || store.profiles[0].ID != outside.ID {
		t.Fatalf("profiles = %+v, want only the outside connection", store.profiles)
	}
	if len(store.folders) != 0 {
		t.Fatalf("folders = %v, want the subtree removed", store.folders)
	}
	if _, open := s.openTabs[inside.ID]; open {
		t.Error("removed connection kept its session tab")
	}
}

func TestDeleteFolder_Declined_KeepsEverything(t *testing.T) {
	connection := sampleProfile(t, "Web", "Production", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{connection}, folders: []string{"Production"}}
	s := newTestScreen(t, store)
	s.confirm = (&fakeConfirm{accept: false}).ask
	s.tree.Select(folderUID("Production"))
	s.requestDeleteFolder()
	if store.saves != 0 || len(store.profiles) != 1 {
		t.Fatalf("declined delete changed the store: %+v", store.profiles)
	}
}

func TestMoveConnection_MovesIntoNewFolderAndBackToRoot(t *testing.T) {
	connection := sampleProfile(t, "Web", "", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{connection}}
	s := newTestScreen(t, store)
	s.tree.Select(connectionUID(connection.ID))

	s.moveConnection(connection, "Ops/Oncall")
	if store.profiles[0].Folder != "Ops/Oncall" {
		t.Fatalf("folder = %q, want Ops/Oncall", store.profiles[0].Folder)
	}
	if len(store.folders) != 2 {
		t.Fatalf("folders = %v, want Ops and Ops/Oncall", store.folders)
	}

	s.moveConnection(store.profiles[0], "")
	if store.profiles[0].Folder != "" {
		t.Fatalf("folder = %q, want the root", store.profiles[0].Folder)
	}
	if !strings.Contains(s.status.Text, "root") {
		t.Errorf("status = %q, want a root notice", s.status.Text)
	}
}

func TestMoveConnection_InvalidFolder_IsRejected(t *testing.T) {
	connection := sampleProfile(t, "Web", "", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{connection}}
	s := newTestScreen(t, store)
	s.moveConnection(connection, "../escape")
	if store.saves != 0 {
		t.Fatalf("store saved %d times for an invalid move", store.saves)
	}
	if !strings.Contains(s.status.Text, "Cannot move") {
		t.Errorf("status = %q, want a move error", s.status.Text)
	}
}

func TestWithinFolder_AndPrefixReplacement(t *testing.T) {
	if !withinFolder("Production/Web", "Production") || withinFolder("Production2", "Production") {
		t.Error("withinFolder is wrong")
	}
	if got := replaceFolderPrefix("Production/Web", "Production", "Clientes"); got != "Clientes/Web" {
		t.Errorf("replaceFolderPrefix = %q, want Clientes/Web", got)
	}
	if got := withinFolder("", ""); !got {
		t.Error("the root should be within the root")
	}
}

func TestRequestRenameConnection_UpdatesNameAndTab(t *testing.T) {
	connection := sampleProfile(t, "Web", "", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{connection}}
	s := newTestScreen(t, store)
	s.tree.Select(connectionUID(connection.ID))
	s.selected = connection.ID
	s.requestRenameConnection()

	// The dialog is modal, so drive the rename path directly.
	s.renameConnection(connection, "Web primary")
	if store.profiles[0].Name != "Web primary" {
		t.Fatalf("name = %q, want Web primary", store.profiles[0].Name)
	}
	if tab, open := s.openTabs[connection.ID]; !open || tab.Text != "Web primary" {
		t.Errorf("tab label not updated: %v", tab)
	}
}

func TestSaveFolderDefaults_PersistsAndClears(t *testing.T) {
	connection := sampleProfile(t, "web", "Team", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{connection}, folders: []string{"Team"}}
	s := newTestScreen(t, store)

	s.saveFolderDefaults("Team", "team-user", "2222")
	document, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(document.Defaults) != 1 || document.Defaults[0].Username != "team-user" || document.Defaults[0].Port != 2222 {
		t.Fatalf("defaults = %+v", document.Defaults)
	}

	s.saveFolderDefaults("Team", "", "")
	document, err = store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(document.Defaults) != 0 {
		t.Fatalf("defaults = %+v, want the entry cleared", document.Defaults)
	}
}

func TestSaveFolderDefaults_InvalidPortIsRejected(t *testing.T) {
	store := &fakeStore{folders: []string{"Team"}}
	s := newTestScreen(t, store)
	s.saveFolderDefaults("Team", "team-user", "not-a-port")
	if store.saves != 0 {
		t.Fatalf("store saved %d times for an invalid port", store.saves)
	}
	if !strings.Contains(s.status.Text, "must be a number") {
		t.Errorf("status = %q, want a port error", s.status.Text)
	}
}

func TestInheritanceToggles_ShowEffectiveValuesAndPersist(t *testing.T) {
	connection := sampleProfile(t, "web", "Team", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{connection}, folders: []string{"Team"},
		defaults: []profiles.FolderDefault{{Path: "Team", Username: "team-user", Port: 2222}}}
	s := newTestScreen(t, store)
	s.tree.Select(connectionUID(connection.ID))

	s.props.inheritUser.SetChecked(true)
	s.props.inheritPort.SetChecked(true)
	if !s.props.user.Disabled() || !s.props.port.Disabled() {
		t.Error("inherited fields stay editable")
	}
	if s.props.user.PlaceHolder != "team-user" || s.props.port.PlaceHolder != "2222" {
		t.Errorf("placeholders = %q/%q, want the folder values", s.props.user.PlaceHolder, s.props.port.PlaceHolder)
	}
	test.Tap(s.props.save)
	if !store.profiles[0].InheritUsername || !store.profiles[0].InheritPort {
		t.Errorf("saved profile = %+v, want the inheritance flags", store.profiles[0])
	}
	if s.resolve(store.profiles[0]).Port != 2222 {
		t.Error("resolved port does not come from the folder")
	}
}
