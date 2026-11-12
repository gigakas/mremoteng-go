package ui

import (
	"fyne.io/fyne/v2"
	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	"strings"
	"testing"
)

func TestTreeSelect_Connection_OpensSessionTabAndLoadsConfig(t *testing.T) {
	existing := sampleProfile(t, "Production web", "Production", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{existing}})

	s.tree.Select(connectionUID(existing.ID))

	if s.selected != existing.ID {
		t.Fatalf("selection = %q, want %q", s.selected, existing.ID)
	}
	if tab, open := s.openTabs[existing.ID]; !open || tab.Text != existing.Name {
		t.Fatalf("session tab = %v, want one named %q", open, existing.Name)
	}
	if s.tabs.Selected() != s.openTabs[existing.ID] {
		t.Error("session tab is not selected after opening it")
	}
	if s.props.name.Text != existing.Name || s.props.host.Text != existing.Host {
		t.Errorf("Config panel = %q/%q, want the selected profile", s.props.name.Text, s.props.host.Text)
	}
}

func TestTreeSelect_Folder_ClearsSelectionAndExplainsGrouping(t *testing.T) {
	existing := sampleProfile(t, "Production web", "Production", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{existing}})

	s.tree.Select(folderUID("Production"))

	if s.selected != "" {
		t.Errorf("selection = %q, want no connection selected", s.selected)
	}
	if !strings.Contains(s.status.Text, "Folder") {
		t.Errorf("status = %q, want folder explanation", s.status.Text)
	}
}

func TestTreeSelect_UnsavedChanges_DeclinedKeepsEdits(t *testing.T) {
	first := sampleProfile(t, "First", "", "SSH", "first.example.com", 22)
	second := sampleProfile(t, "Second", "", "SSH", "second.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{first, second}})
	guard := &fakeConfirm{accept: false}
	s.confirm = guard.ask

	s.tree.Select(connectionUID(first.ID))
	s.props.name.SetText("Edited name")
	s.tree.Select(connectionUID(second.ID))

	if len(guard.asked) != 1 || guard.asked[0] != discardTitle {
		t.Fatalf("confirmations = %v, want one discard prompt", guard.asked)
	}
	if s.props.name.Text != "Edited name" || s.selected != first.ID {
		t.Errorf("edits = %q, selection = %q; want them preserved", s.props.name.Text, s.selected)
	}
}

func TestRowMenu_FolderOffersFolderActions(t *testing.T) {
	store := &fakeStore{folders: []string{"Production"}}
	s := newTestScreen(t, store)
	menu := s.rowMenu(folderUID("Production"))
	labels := make([]string, 0, len(menu.Items))
	for _, item := range menu.Items {
		labels = append(labels, item.Label)
	}
	want := []string{"Folder defaults", "New folder inside", "Rename folder", "Delete folder"}
	if len(labels) != len(want) {
		t.Fatalf("menu = %v, want %v", labels, want)
	}
	for i := range want {
		if labels[i] != want[i] {
			t.Fatalf("menu = %v, want %v", labels, want)
		}
	}
}

func TestRowTapped_OpensConnectionAndSelectsIt(t *testing.T) {
	connection := sampleProfile(t, "Web", "Production", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{connection}, folders: []string{"Production"}})
	row := newTreeRow(s)
	row.setConnection(connectionUID(connection.ID), connection)
	row.Tapped(&fyne.PointEvent{})

	if s.selected != connection.ID {
		t.Fatalf("selection = %q, want the tapped connection", s.selected)
	}
	if tab, open := s.openTabs[connection.ID]; !open || tab.Text != connection.Name {
		t.Fatalf("session tab = %v, want it opened by the click", open)
	}
}

func TestRowTapped_FolderSelectsIt(t *testing.T) {
	connection := sampleProfile(t, "Web", "Production", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{connection}, folders: []string{"Production"}})
	row := newTreeRow(s)
	row.setFolder(folderUID("Production"), "Production")
	row.Tapped(&fyne.PointEvent{})

	if s.selected != "" || s.selectedUID != folderUID("Production") {
		t.Fatalf("selection = %q/%q, want the folder", s.selected, s.selectedUID)
	}
}

func TestMoveToMenu_ListsRootAndFoldersAndMoves(t *testing.T) {
	connection := sampleProfile(t, "Web", "", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{
		profiles: []profiles.Profile{connection},
		folders:  []string{"Ops", "Production"},
	})
	menu := s.moveToMenu(connection)
	if len(menu.Items) != 3 {
		t.Fatalf("items = %d, want root plus two folders", len(menu.Items))
	}
	if menu.Items[0].Label != "(root)" || menu.Items[1].Label != "Ops" {
		t.Fatalf("items = %v", menu.Items)
	}
	menu.Items[1].Action()
	if s.ws.profiles[0].Folder != "Ops" {
		t.Fatalf("folder = %q, want Ops after the menu move", s.ws.profiles[0].Folder)
	}
	menu.Items[0].Action()
	if s.ws.profiles[0].Folder != "" {
		t.Fatalf("folder = %q, want the root after the menu move", s.ws.profiles[0].Folder)
	}
}

func TestCreateFolderInside_UsesUniqueNamesWithoutDialog(t *testing.T) {
	store := &fakeStore{}
	s := newTestScreen(t, store)
	s.createFolderInside("")
	s.createFolderInside("")
	if len(store.folders) != 2 || store.folders[0] != "New folder" || store.folders[1] != "New folder 2" {
		t.Fatalf("folders = %v, want New folder and New folder 2", store.folders)
	}
	s.createFolderInside("New folder")
	if store.folders[len(store.folders)-1] != "New folder/New folder" {
		t.Fatalf("folders = %v, want a nested default name", store.folders)
	}
}

func TestRowMenu_ConnectionOffersOpenMoveDelete(t *testing.T) {
	connection := sampleProfile(t, "Web", "Production", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{connection}, folders: []string{"Production"}})
	menu := s.rowMenu(connectionUID(connection.ID))
	if len(menu.Items) != 4 {
		t.Fatalf("menu items = %d, want 4", len(menu.Items))
	}
	if menu.Items[0].Label != "Rename" || menu.Items[2].Label != "Move to" || menu.Items[3].Label != "Delete connection" {
		t.Errorf("unexpected menu order: %v", menu.Items)
	}
	// The delete item must select the row before acting on it.
	menu.Items[3].Action()
	if s.selected != connection.ID {
		t.Errorf("selection = %q, want the menu row", s.selected)
	}
}

func TestFolderAt_ReturnsContainingRow(t *testing.T) {
	rects := map[string]rect{
		folderUID("Production"): {x: 10, y: 10, width: 100, height: 20},
		folderUID("Ops"):        {x: 10, y: 40, width: 100, height: 20},
	}
	if got := folderAt(fyne.NewPos(50, 45), rects); got != folderUID("Ops") {
		t.Errorf("folderAt = %q, want the Ops folder", got)
	}
	if got := folderAt(fyne.NewPos(500, 500), rects); got != "" {
		t.Errorf("folderAt = %q, want no folder", got)
	}
}

func TestTreeRow_VNCBadge_UsesItsOwnLetterAndColour(t *testing.T) {
	row := newTreeRow(nil)
	row.setConnection("c:vnc", sampleProfile(t, "Desktop", "", "VNC", "vnc.example.com", 5900))
	if row.badge.letter != "V" || row.badge.bg != vncBadgeColor {
		t.Fatalf("badge = %q %v, want V with the VNC colour", row.badge.letter, row.badge.bg)
	}
}

func TestTreeRow_Connection_ShowsCircularProtocolBadge(t *testing.T) {
	row := newTreeRow(nil)
	row.setConnection("c:test", sampleProfile(t, "Production web", "", "SSH", "web.example.com", 22))
	if row.badge.letter != "S" || row.badge.Hidden {
		t.Fatalf("badge = %q hidden=%v, want a visible SSH badge", row.badge.letter, row.badge.Hidden)
	}
	if row.badge.bg != sshBadgeColor {
		t.Error("SSH badge does not use the SSH colour")
	}

	row.setConnection("c:test", sampleProfile(t, "Office desktop", "", "RDP", "desk.example.com", 3389))
	if row.badge.letter != "R" {
		t.Fatalf("badge = %q, want R", row.badge.letter)
	}
	if row.badge.bg != rdpBadgeColor {
		t.Error("RDP badge does not use the RDP colour")
	}
}

func TestTreeRow_Folder_HidesProtocolBadge(t *testing.T) {
	row := newTreeRow(nil)
	row.setFolder("f:Production", "Production")
	if row.name.Text != "Production" {
		t.Errorf("row = %q, want the folder name", row.name.Text)
	}
	if !row.badge.Hidden {
		t.Error("folder row shows a protocol badge")
	}
}

func TestTreeHighlighted_ReportsEndpointInStatusBar(t *testing.T) {
	existing := sampleProfile(t, "Production web", "Production", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{existing}})
	s.treeHighlighted(connectionUID(existing.ID))
	if !strings.Contains(s.status.Text, "web.example.com:22") {
		t.Errorf("status = %q, want the hovered endpoint", s.status.Text)
	}
	s.treeHighlighted(folderUID("Production"))
	if !strings.Contains(s.status.Text, "Production") {
		t.Errorf("status = %q, want the highlighted folder", s.status.Text)
	}
}
