package ui

import (
	"strings"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

// orderScreen opens a screen with three connections and one folder in the root.
func orderScreen(t *testing.T) (*screen, *fakeStore, []profiles.Profile) {
	t.Helper()
	alpha := sampleProfile(t, "Alpha", "", string(profiles.SSH), "a.example.com", 22)
	beta := sampleProfile(t, "Beta", "", string(profiles.SSH), "b.example.com", 22)
	gamma := sampleProfile(t, "Gamma", "", string(profiles.SSH), "g.example.com", 22)
	store := &fakeStore{
		profiles: []profiles.Profile{alpha, beta, gamma},
		folders:  []string{"Archive"},
	}
	s := newTestScreen(t, store)
	return s, store, []profiles.Profile{alpha, beta, gamma}
}

// rootLabels lists the root rows as the tree shows them.
func rootLabels(s *screen) []string {
	labels := make([]string, 0, 4)
	for _, uid := range s.index.childUIDs("") {
		labels = append(labels, s.index.label(uid))
	}
	return labels
}

func TestMoveInOrder_SwapsWithTheNextConnectionAndPersists(t *testing.T) {
	s, store, profiles := orderScreen(t)
	if got := strings.Join(rootLabels(s), ","); got != "Archive,Alpha,Beta,Gamma" {
		t.Fatalf("initial order = %q, want folders first then by name", got)
	}

	s.moveInOrder(connectionUID(profiles[2].ID), -1) // Gamma up

	if got := strings.Join(rootLabels(s), ","); got != "Archive,Alpha,Gamma,Beta" {
		t.Errorf("order after moving up = %q", got)
	}
	if len(store.order) == 0 {
		t.Fatal("the new order was not stored")
	}
	stored := strings.Join(store.order, ",")
	if strings.Index(stored, connectionUID(profiles[2].ID)) > strings.Index(stored, connectionUID(profiles[1].ID)) {
		t.Errorf("stored order = %q, want Gamma before Beta", stored)
	}
	if !strings.Contains(s.status.Text, "Moved Gamma up") {
		t.Errorf("status = %q, want the move reported", s.status.Text)
	}
}

// TestMoveInOrder_NeverCrossesIntoTheFolderGroup keeps navigation predictable: folders
// stay ahead of connections however the connections are arranged.
func TestMoveInOrder_NeverCrossesIntoTheFolderGroup(t *testing.T) {
	s, _, profiles := orderScreen(t)

	s.moveInOrder(connectionUID(profiles[0].ID), -1) // Alpha is already first

	if got := strings.Join(rootLabels(s), ","); got != "Archive,Alpha,Beta,Gamma" {
		t.Errorf("order = %q, want it unchanged", got)
	}
	if !strings.Contains(s.status.Text, "already the first connection") {
		t.Errorf("status = %q, want it to explain why nothing moved", s.status.Text)
	}
}

func TestMoveInOrder_LastConnection_ReportsTheEdge(t *testing.T) {
	s, _, profiles := orderScreen(t)

	s.moveInOrder(connectionUID(profiles[2].ID), 1)

	if !strings.Contains(s.status.Text, "already the last connection") {
		t.Errorf("status = %q, want the edge explained", s.status.Text)
	}
}

func TestMoveInOrder_Folder_ReordersAmongFolders(t *testing.T) {
	store := &fakeStore{folders: []string{"Archive", "Zeta"}}
	s := newTestScreen(t, store)

	s.moveInOrder(folderUID("Zeta"), -1)

	if got := strings.Join(rootLabels(s), ","); got != "Zeta,Archive" {
		t.Errorf("folders = %q, want the move applied", got)
	}
	if !strings.Contains(strings.Join(store.order, ","), folderUID("Zeta")) {
		t.Errorf("stored order = %v, want the folder in it", store.order)
	}
}

// TestManualOrder_SurvivesARestart is the point of storing it at all.
func TestManualOrder_SurvivesARestart(t *testing.T) {
	s, store, profiles := orderScreen(t)
	s.moveInOrder(connectionUID(profiles[2].ID), -1)

	again := newTestScreen(t, store)

	if got := strings.Join(rootLabels(again), ","); got != "Archive,Alpha,Gamma,Beta" {
		t.Errorf("order after restart = %q, want the arranged one", got)
	}
}

// TestManualOrder_UnknownKeysAreIgnored covers a store that names nodes this tree no
// longer has: the rest of the order still applies.
func TestManualOrder_UnknownKeysAreIgnored(t *testing.T) {
	alpha := sampleProfile(t, "Alpha", "", string(profiles.SSH), "a.example.com", 22)
	beta := sampleProfile(t, "Beta", "", string(profiles.SSH), "b.example.com", 22)
	store := &fakeStore{
		profiles: []profiles.Profile{alpha, beta},
		order:    []string{"c:gone", connectionUID(beta.ID), connectionUID(alpha.ID)},
	}

	s := newTestScreen(t, store)

	if got := strings.Join(rootLabels(s), ","); got != "Beta,Alpha" {
		t.Errorf("order = %q, want the stored arrangement with the missing node skipped", got)
	}
}

// TestOrderMenu_IsDisabledAtTheEdges makes the context menu state what is possible.
func TestOrderMenu_IsDisabledAtTheEdges(t *testing.T) {
	s, _, profiles := orderScreen(t)

	first := s.orderItem("Move up", connectionUID(profiles[0].ID), -1)
	if !first.Disabled {
		t.Error("Move up is enabled for the first connection")
	}
	last := s.orderItem("Move down", connectionUID(profiles[2].ID), 1)
	if !last.Disabled {
		t.Error("Move down is enabled for the last connection")
	}
	middle := s.orderItem("Move up", connectionUID(profiles[1].ID), -1)
	if middle.Disabled {
		t.Error("Move up is disabled for a middle connection")
	}
}

// TestMoveInOrder_WhileFiltered_KeepsTheHiddenNodesInPlace covers reordering what a
// filter left on screen: the rows it hides must keep their arrangement, or clearing
// the filter would shuffle them back to alphabetical order.
func TestMoveInOrder_WhileFiltered_KeepsTheHiddenNodesInPlace(t *testing.T) {
	alpha := sampleProfile(t, "Alpha", "", string(profiles.SSH), "alpha.example.com", 22)
	bravo := sampleProfile(t, "Bravo", "", string(profiles.SSH), "bravo.example.com", 22)
	charlie := sampleProfile(t, "Charlie", "", string(profiles.SSH), "charlie.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{alpha, bravo, charlie}}
	s := newTestScreen(t, store)
	// Arrange Charlie, Alpha, Bravo first, so the hidden node has a position to keep.
	s.moveInOrder(connectionUID(charlie.ID), -1)
	s.moveInOrder(connectionUID(charlie.ID), -1)
	if got := strings.Join(rootLabels(s), ","); got != "Charlie,Alpha,Bravo" {
		t.Fatalf("arranged order = %q", got)
	}

	s.filter.SetText("ha") // Alpha and Charlie match; Bravo does not
	if got := strings.Join(rootLabels(s), ","); got != "Charlie,Alpha" {
		t.Fatalf("filtered order = %q, want the arranged one", got)
	}
	s.moveInOrder(connectionUID(alpha.ID), -1) // Alpha above Charlie, both visible

	s.filter.SetText("")
	if got := strings.Join(rootLabels(s), ","); got != "Alpha,Charlie,Bravo" {
		t.Errorf("order after clearing the filter = %q, want Bravo to have kept its place", got)
	}
}
