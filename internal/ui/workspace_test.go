package ui

import (
	"errors"
	"slices"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

func profileIn(t *testing.T, name, folder string) profiles.Profile {
	t.Helper()
	return sampleProfile(t, name, folder, string(profiles.SSH), "192.0.2.10", 22)
}

func folderOf(list []string) func(string) bool {
	return func(folder string) bool { return slices.Contains(list, folder) }
}

func TestWorkspace_LoadFailure_LeavesReadOnlyAndEmpty(t *testing.T) {
	store := &fakeStore{loadErr: errors.New("unreadable")}
	w := newWorkspace(store)

	if err := w.load(); err == nil {
		t.Fatal("load did not report the unreadable store")
	}
	if w.writable {
		t.Error("workspace is writable after a failed load")
	}
	if len(w.profiles) != 0 || len(w.folders) != 0 || len(w.defaults) != 0 || len(w.collapsed) != 0 {
		t.Errorf("state not cleared: %d profiles, %d folders, %d defaults, %d collapsed",
			len(w.profiles), len(w.folders), len(w.defaults), len(w.collapsed))
	}
	// A read-only workspace must not write, and must not report an error either:
	// the window stays usable and the repair path is the only way out.
	if err := w.save(); err != nil {
		t.Errorf("save on a read-only workspace returned %v, want nil", err)
	}
	if store.saves != 0 {
		t.Errorf("store received %d writes from a read-only workspace, want 0", store.saves)
	}
}

func TestWorkspace_Load_PopulatesStateAndCollapsedSet(t *testing.T) {
	web := profileIn(t, "web", "Production/Web")
	store := &fakeStore{
		profiles:  []profiles.Profile{web},
		folders:   []string{"Production", "Production/Web"},
		defaults:  []profiles.FolderDefault{{Path: "Production", Port: 22}},
		collapsed: []string{"Production"},
	}
	w := newWorkspace(store)

	if err := w.load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	if !w.writable {
		t.Error("workspace is read-only after a successful load")
	}
	if len(w.profiles) != 1 || w.profiles[0].Name != "web" {
		t.Errorf("profiles = %+v, want the stored connection", w.profiles)
	}
	if !folderOf(w.folders)("Production/Web") {
		t.Errorf("folders = %v, want the stored folders", w.folders)
	}
	if len(w.defaults) != 1 || w.defaults[0].Path != "Production" {
		t.Errorf("defaults = %+v, want the stored defaults", w.defaults)
	}
	if !w.collapsed["Production"] {
		t.Errorf("collapsed = %v, want Production", w.collapsed)
	}
}

func TestWorkspace_RejectedWrite_KeepsState(t *testing.T) {
	web := profileIn(t, "web", "")
	store := &fakeStore{profiles: []profiles.Profile{web}}
	w := newWorkspace(store)
	if err := w.load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	store.saveErr = errors.New("disk full")

	if err := w.putProfile(profileIn(t, "api", "")); err == nil {
		t.Fatal("putProfile reported success although the store rejected the write")
	}
	if len(w.profiles) != 1 || w.profiles[0].ID != web.ID {
		t.Errorf("profiles = %+v, want the single original connection", w.profiles)
	}
}

func TestWorkspace_PutProfile_AddsConnectionAndItsFolders(t *testing.T) {
	w := newWorkspace(&fakeStore{})
	if err := w.load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	if err := w.putProfile(profileIn(t, "web", "Production/Web")); err != nil {
		t.Fatalf("putProfile: %v", err)
	}
	if len(w.profiles) != 1 {
		t.Fatalf("profiles = %d, want 1", len(w.profiles))
	}
	// addFolderPath records every parent, not only the leaf.
	for _, want := range []string{"Production", "Production/Web"} {
		if !folderOf(w.folders)(want) {
			t.Errorf("folders = %v, want it to include %q", w.folders, want)
		}
	}

	// Saving the same connection again replaces it instead of duplicating it.
	updated := w.profiles[0]
	updated.Name = "web renamed"
	if err := w.putProfile(updated); err != nil {
		t.Fatalf("putProfile (update): %v", err)
	}
	if len(w.profiles) != 1 || w.profiles[0].Name != "web renamed" {
		t.Errorf("profiles = %+v, want a single updated connection", w.profiles)
	}
}

func TestWorkspace_DeleteProfile_RemovesOnlyThatConnection(t *testing.T) {
	web := profileIn(t, "web", "")
	api := profileIn(t, "api", "")
	w := newWorkspace(&fakeStore{profiles: []profiles.Profile{web, api}})
	if err := w.load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	if err := w.deleteProfile(web.ID); err != nil {
		t.Fatalf("deleteProfile: %v", err)
	}
	if len(w.profiles) != 1 || w.profiles[0].ID != api.ID {
		t.Errorf("profiles = %+v, want only api", w.profiles)
	}
}

func TestWorkspace_AddFolder_AppendsEachPathOnce(t *testing.T) {
	w := newWorkspace(&fakeStore{})
	if err := w.load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	if err := w.addFolder("Production/Web"); err != nil {
		t.Fatalf("addFolder: %v", err)
	}
	count := len(w.folders)
	if err := w.addFolder("Production/Web"); err != nil {
		t.Fatalf("addFolder (repeat): %v", err)
	}
	if len(w.folders) != count {
		t.Errorf("folders = %v, want no duplicates on a repeated add", w.folders)
	}
}

func TestWorkspace_SetFolderDefaults_StoresAndClearsAnEntry(t *testing.T) {
	w := newWorkspace(&fakeStore{})
	if err := w.load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	if err := w.setFolderDefaults("Production", "tester", 2222); err != nil {
		t.Fatalf("setFolderDefaults: %v", err)
	}
	if len(w.defaults) != 1 || w.defaults[0].Username != "tester" || w.defaults[0].Port != 2222 {
		t.Fatalf("defaults = %+v, want the stored entry", w.defaults)
	}

	// An empty username and port removes the entry instead of storing blanks.
	if err := w.setFolderDefaults("Production", "", 0); err != nil {
		t.Fatalf("setFolderDefaults (clear): %v", err)
	}
	if len(w.defaults) != 0 {
		t.Errorf("defaults = %+v, want the entry removed", w.defaults)
	}
}

func TestWorkspace_RenameFolder_RewritesTheWholeSubtree(t *testing.T) {
	web := profileIn(t, "web", "Production/Web")
	root := profileIn(t, "root", "")
	store := &fakeStore{
		profiles: []profiles.Profile{web, root},
		folders:  []string{"Production", "Production/Web"},
	}
	w := newWorkspace(store)
	if err := w.load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	if err := w.renameFolder("Production", "Platform"); err != nil {
		t.Fatalf("renameFolder: %v", err)
	}
	if folderOf(w.folders)("Production") || folderOf(w.folders)("Production/Web") {
		t.Errorf("folders = %v, want the old paths gone", w.folders)
	}
	for _, want := range []string{"Platform", "Platform/Web"} {
		if !folderOf(w.folders)(want) {
			t.Errorf("folders = %v, want %q", w.folders, want)
		}
	}
	if w.profiles[0].Folder != "Platform/Web" {
		t.Errorf("connection folder = %q, want it rewritten", w.profiles[0].Folder)
	}
	if w.profiles[1].Folder != "" {
		t.Errorf("connection outside the subtree changed to %q", w.profiles[1].Folder)
	}
}

func TestWorkspace_DeleteFolder_DropsContainedConnectionsAndFolders(t *testing.T) {
	web := profileIn(t, "web", "Production/Web")
	root := profileIn(t, "root", "")
	store := &fakeStore{
		profiles: []profiles.Profile{web, root},
		folders:  []string{"Production", "Production/Web", "Other"},
	}
	w := newWorkspace(store)
	if err := w.load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	if err := w.deleteFolder("Production"); err != nil {
		t.Fatalf("deleteFolder: %v", err)
	}
	if len(w.profiles) != 1 || w.profiles[0].ID != root.ID {
		t.Errorf("profiles = %+v, want only the connection outside the folder", w.profiles)
	}
	if folderOf(w.folders)("Production") || folderOf(w.folders)("Production/Web") {
		t.Errorf("folders = %v, want the subtree removed", w.folders)
	}
	if !folderOf(w.folders)("Other") {
		t.Errorf("folders = %v, want the unrelated folder kept", w.folders)
	}
}

func TestWorkspace_PutConnection_AddsFoldersButRenameDoesNot(t *testing.T) {
	moved := profileIn(t, "web", "Staging")
	store := &fakeStore{profiles: []profiles.Profile{moved}}
	w := newWorkspace(store)
	if err := w.load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	// A rename keeps the folder list untouched.
	if err := w.renameProfile(moved); err != nil {
		t.Fatalf("renameProfile: %v", err)
	}
	if folderOf(w.folders)("Staging") {
		t.Errorf("folders = %v, want a rename to leave the list alone", w.folders)
	}

	// A move records the destination folder.
	if err := w.putConnection(moved); err != nil {
		t.Fatalf("putConnection: %v", err)
	}
	if !folderOf(w.folders)("Staging") {
		t.Errorf("folders = %v, want the destination recorded", w.folders)
	}
}

func TestWorkspace_ImportMerge_SkipsDuplicatesAndAddsFolders(t *testing.T) {
	web := profileIn(t, "web", "Production")
	store := &fakeStore{profiles: []profiles.Profile{web}, folders: []string{"Production"}}
	w := newWorkspace(store)
	if err := w.load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	duplicate := profileIn(t, "web", "Production")
	incoming := profileIn(t, "api", "Production/Api")
	added, skipped, err := w.importInto([]profiles.Profile{duplicate, incoming}, []string{"Production/Api"}, false)
	if err != nil {
		t.Fatalf("importInto: %v", err)
	}
	if added != 1 || skipped != 1 {
		t.Errorf("added/skipped = %d/%d, want 1/1", added, skipped)
	}
	if len(w.profiles) != 2 {
		t.Errorf("profiles = %d, want the existing plus the imported one", len(w.profiles))
	}
	if !folderOf(w.folders)("Production/Api") {
		t.Errorf("folders = %v, want the imported folder", w.folders)
	}
}

func TestWorkspace_ImportReplace_KeepsEverythingIncoming(t *testing.T) {
	web := profileIn(t, "web", "Production")
	store := &fakeStore{profiles: []profiles.Profile{web}, folders: []string{"Production"}}
	w := newWorkspace(store)
	if err := w.load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	// Same name and folder as the existing one: replace mode drops the old data
	// first, so nothing is a duplicate and both incoming profiles enter.
	same := profileIn(t, "web", "Production")
	other := profileIn(t, "api", "Staging")
	added, skipped, err := w.importInto([]profiles.Profile{same, other}, []string{"Staging"}, true)
	if err != nil {
		t.Fatalf("importInto: %v", err)
	}
	if added != 2 || skipped != 0 {
		t.Errorf("added/skipped = %d/%d, want 2/0", added, skipped)
	}
	if len(w.profiles) != 2 {
		t.Errorf("profiles = %d, want exactly the imported set", len(w.profiles))
	}
	if folderOf(w.folders)("Production") {
		t.Errorf("folders = %v, want the old folder dropped", w.folders)
	}
}

func TestWorkspace_Reset_ReEnablesWritesAfterAFailedLoad(t *testing.T) {
	store := &fakeStore{loadErr: errors.New("unreadable")}
	w := newWorkspace(store)
	if err := w.load(); err == nil {
		t.Fatal("load did not report the unreadable store")
	}
	store.loadErr = nil

	if err := w.reset(); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if !w.writable {
		t.Error("workspace is still read-only after a reset")
	}
	if store.saves != 1 {
		t.Errorf("store received %d writes, want the single empty document", store.saves)
	}
	if len(store.profiles) != 0 {
		t.Errorf("stored profiles = %d, want an empty document", len(store.profiles))
	}
}
