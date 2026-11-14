package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/container"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

// liveTestSession marks a panel as carrying an open session without dialling
// anywhere, which is what the disconnect question looks at.
func liveTestSession(panel *sessionPanel) {
	panel.shell = &fakeShell{output: strings.NewReader("")}
}

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

// TestRequestDisconnect_AnotherSessionInTheGroup_SkipsTheQuestion covers the
// reported annoyance: a split tab holds several sessions of the same group, and
// disconnecting one of them leaves the group on screen, so there is nothing to
// approve.
func TestRequestDisconnect_AnotherSessionInTheGroup_SkipsTheQuestion(t *testing.T) {
	s, panel := openSplitTestTab(t)
	s.splitPane(panel, true)
	sibling := s.tabPanels(panel.profile.ID)[1]
	guard := &fakeConfirm{accept: true}
	s.confirm = guard.ask
	liveTestSession(panel)
	liveTestSession(sibling)

	panel.requestDisconnect()

	if len(guard.asked) != 0 {
		t.Errorf("confirmations = %v, want none while the group still has a session", guard.asked)
	}
	if panel.shell != nil {
		t.Error("the session of the pane stayed open")
	}
	if sibling.shell == nil {
		t.Error("the sibling session was disconnected as well")
	}
	if panels := s.tabPanels(panel.profile.ID); len(panels) != 1 || panels[0] != sibling {
		t.Errorf("panes after the disconnect = %d, want the sibling alone", len(panels))
	}
}

// TestRequestDisconnect_SiblingPaneWithoutASession_StillAsks keeps the rule about
// sessions and not about panes: an empty pane is not another session, so the
// question is still due.
func TestRequestDisconnect_SiblingPaneWithoutASession_StillAsks(t *testing.T) {
	s, panel := openSplitTestTab(t)
	s.splitPane(panel, true)
	guard := &fakeConfirm{accept: true}
	s.confirm = guard.ask
	liveTestSession(panel)

	panel.requestDisconnect()

	if len(guard.asked) != 1 || guard.asked[0] != "Disconnect session?" {
		t.Fatalf("confirmations = %v, want one disconnect question", guard.asked)
	}
	if panel.shell != nil {
		t.Error("accepting the question did not end the session")
	}
	if panels := s.tabPanels(panel.profile.ID); len(panels) != 1 {
		t.Errorf("panes after the disconnect = %d, want the closed one gone", len(panels))
	}
}

// TestRequestDisconnect_Declined_KeepsTheSession is the reason the question
// exists: a mistaken click on the X must not take the session away.
func TestRequestDisconnect_Declined_KeepsTheSession(t *testing.T) {
	s, panel := openSplitTestTab(t)
	guard := &fakeConfirm{accept: false}
	s.confirm = guard.ask
	liveTestSession(panel)

	panel.requestDisconnect()

	if len(guard.asked) != 1 {
		t.Fatalf("confirmations = %v, want one disconnect question", guard.asked)
	}
	if panel.shell == nil {
		t.Error("declining the question ended the session anyway")
	}
}

// TestRequestDisconnect_LastPaneOfATab_KeepsThePane pins the other half of the
// rule: a tab always shows one pane, so the last one stays after its session ends
// and Connect can open the next session in place.
func TestRequestDisconnect_LastPaneOfATab_KeepsThePane(t *testing.T) {
	s, panel := openSplitTestTab(t)
	guard := &fakeConfirm{accept: true}
	s.confirm = guard.ask
	liveTestSession(panel)

	panel.requestDisconnect()

	if panels := s.tabPanels(panel.profile.ID); len(panels) != 1 || panels[0] != panel {
		t.Fatalf("panes after the disconnect = %d, want the pane to stay", len(panels))
	}
	if len(panel.slot.Objects) != 1 {
		t.Fatalf("slot objects = %d, want the closed placeholder", len(panel.slot.Objects))
	}
	if s.panels[panel.profile.ID] != panel {
		t.Error("the tab still points at the closed pane")
	}
}

// TestCloseSelf_SplitPane_SurvivorKeepsTheKeyboard guards the hand-over: removing
// the pane that had focus must leave the sibling ready to type into.
func TestCloseSelf_SplitPane_SurvivorKeepsTheKeyboard(t *testing.T) {
	s, panel := openSplitTestTab(t)
	s.splitPane(panel, true)
	sibling := s.tabPanels(panel.profile.ID)[1]
	sibling.term = newTermWidget(20, 5, func([]byte) {}, func(int, int) {})
	sibling.installTerminal()

	panel.closeSelf()

	if sibling.term == nil {
		t.Fatal("the surviving pane lost its terminal")
	}
	if len(s.tabPanels(panel.profile.ID)) != 1 {
		t.Error("closing the pane left the split behind")
	}
}

// countPanes reports how many panes a stored layout describes.
func countPanes(layout profiles.PaneLayout) int {
	if !layout.Split {
		return 1
	}
	total := 0
	if layout.A != nil {
		total += countPanes(*layout.A)
	}
	if layout.B != nil {
		total += countPanes(*layout.B)
	}
	return total
}

// TestScreen_SplitLayout_IsPersistedAndRestored covers the half of session restore
// the first version left out: a divided tab comes back divided, in the same shape.
func TestScreen_SplitLayout_IsPersistedAndRestored(t *testing.T) {
	profile := sshPanelProfile(t, "")
	// The profile has to be in the store, or the tab list would not record it and
	// there would be nothing to restore.
	store := &fakeStore{profiles: []profiles.Profile{profile}}
	s := newTestScreen(t, store)
	s.openSessionTab(profile)
	panel := s.panels[profile.ID]
	s.splitPane(panel, true)
	second := s.tabPanels(profile.ID)[1]
	s.splitPane(second, false)

	stored, divided := store.layouts[profile.ID]
	if !divided {
		t.Fatal("the divided tab was not stored")
	}
	if !stored.Split || !stored.SideBySide {
		t.Fatalf("root = %+v, want a side-by-side split", stored)
	}
	if got := countPanes(stored); got != 3 {
		t.Fatalf("stored panes = %d, want 3", got)
	}
	if stored.B == nil || !stored.B.Split || stored.B.SideBySide {
		t.Errorf("second split = %+v, want a stacked split", stored.B)
	}

	again := newTestScreen(t, store)
	panes := again.tabPanels(profile.ID)
	if len(panes) != 3 {
		t.Fatalf("restored panes = %d, want 3", len(panes))
	}
	for _, pane := range panes {
		if pane.hasSession() || pane.term != nil {
			t.Error("a restored pane was connected")
		}
		if pane.connect.Disabled() {
			t.Error("a restored pane cannot be connected")
		}
	}
	root := panes[0].pane.root()
	if !root.sideBySide {
		t.Error("the restored tab lost the side-by-side orientation")
	}
}

// TestScreen_RestoringPanes_KeepsTheTabPointingAtTheRoot keeps closePane and tab
// bookkeeping working after a restore.
func TestScreen_RestoringPanes_KeepsTheTabPointingAtTheRoot(t *testing.T) {
	profile := sshPanelProfile(t, "")
	store := &fakeStore{profiles: []profiles.Profile{profile}}
	s := newTestScreen(t, store)
	s.openSessionTab(profile)
	s.splitPane(s.panels[profile.ID], true)

	again := newTestScreen(t, store)

	if survivor := again.tabPanels(profile.ID)[0]; again.panels[profile.ID] != survivor {
		t.Error("the tab does not point at a restored pane")
	}
	again.closePane(again.tabPanels(profile.ID)[1])
	if got := len(again.tabPanels(profile.ID)); got != 1 {
		t.Errorf("panes after closing one = %d, want 1", got)
	}
	if _, divided := store.layouts[profile.ID]; divided {
		t.Error("the single-pane tab still has a stored layout")
	}
}

// TestScreen_RestoreSessions_IgnoresALayoutTheProtocolCannotSplit covers a store that
// claims an RDP tab was divided: the tab must come back as one pane, not fail.
func TestScreen_RestoreSessions_IgnoresALayoutTheProtocolCannotSplit(t *testing.T) {
	store := &fakeStore{}
	profile := sampleProfile(t, "desk", "", string(profiles.RDP), "desk.example.com", 3389)
	store.profiles = []profiles.Profile{profile}
	store.openTabs = []string{profile.ID}
	store.layouts = map[string]profiles.PaneLayout{
		profile.ID: {Split: true, SideBySide: true, A: &profiles.PaneLayout{}, B: &profiles.PaneLayout{}},
	}

	s := newTestScreen(t, store)

	if got := len(s.tabPanels(profile.ID)); got != 1 {
		t.Errorf("panes = %d, want the RDP tab to stay single", got)
	}
}
