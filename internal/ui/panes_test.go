package ui

import (
	"testing"

	"fyne.io/fyne/v2/container"
)

// openSplitTestTab opens a tab whose profile has no username: connecting a
// duplicate then fails synchronously instead of dialling on a goroutine, which
// the test driver would run concurrently with the test body.
func openSplitTestTab(t *testing.T) (*screen, *sessionPanel) {
	t.Helper()
	s := newTestScreen(t, &fakeStore{})
	profile := sshPanelProfile(t, "")
	s.openSessionTab(profile)
	return s, s.panels[profile.ID]
}

func TestSplitPane_SideBySide_AddsADuplicatePaneInTheSameTab(t *testing.T) {
	s, panel := openSplitTestTab(t)

	s.splitPane(panel, true)

	panels := s.tabPanels(panel.profile.ID)
	if len(panels) != 2 {
		t.Fatalf("panes = %d, want 2", len(panels))
	}
	if panels[1].profile.ID != panel.profile.ID {
		t.Errorf("the new pane should duplicate the connection, got %q", panels[1].profile.Name)
	}
	root := panel.pane.root()
	if _, ok := root.view.Objects[0].(*container.Split); !ok {
		t.Errorf("tab content = %T, want a split", root.view.Objects[0])
	}
	if len(s.openTabs) != 1 {
		t.Errorf("tabs = %d, want the split to stay in one tab", len(s.openTabs))
	}
}

func TestSplitPane_Nested_BuildsAThreePaneLayout(t *testing.T) {
	s, panel := openSplitTestTab(t)

	s.splitPane(panel, true)
	second := s.tabPanels(panel.profile.ID)[1]
	s.splitPane(second, false)

	if got := len(s.tabPanels(panel.profile.ID)); got != 3 {
		t.Fatalf("panes = %d, want 3", got)
	}
}

func TestClosePane_Sibling_TakesTheFreedSpace(t *testing.T) {
	s, panel := openSplitTestTab(t)
	s.splitPane(panel, true)
	second := s.tabPanels(panel.profile.ID)[1]

	s.closePane(second)

	panels := s.tabPanels(panel.profile.ID)
	if len(panels) != 1 || panels[0] != panel {
		t.Fatalf("panes after close = %d, want only the original", len(panels))
	}
	if panel.pane.root().panel != panel {
		t.Error("the remaining pane should be the root again")
	}
	if _, split := panel.pane.root().view.Objects[0].(*container.Split); split {
		t.Error("the tab still shows a split after closing the other pane")
	}
}

func TestClosePane_OriginalPane_KeepsTheTabPointingAtASurvivor(t *testing.T) {
	s, panel := openSplitTestTab(t)
	s.splitPane(panel, false)
	duplicate := s.tabPanels(panel.profile.ID)[1]

	s.closePane(panel)

	if s.panels[panel.profile.ID] != duplicate {
		t.Error("the tab should point at the surviving pane")
	}
}

func TestClosePane_LastPane_Refuses(t *testing.T) {
	s, panel := openSplitTestTab(t)

	s.closePane(panel)

	if len(s.tabPanels(panel.profile.ID)) != 1 {
		t.Error("the only pane of a tab must not be closable")
	}
}

func TestSplitPane_RDP_IsRefused(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	profile := sampleProfile(t, "Desktop", "", "RDP", "rdp.example.com", 3389)
	s.openSessionTab(profile)

	s.splitPane(s.panels[profile.ID], true)

	if got := len(s.tabPanels(profile.ID)); got != 1 {
		t.Errorf("panes = %d, want RDP to stay unsplit", got)
	}
}
