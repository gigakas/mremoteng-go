package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

// storeWithProfiles returns a fake store holding one profile per name.
func storeWithProfiles(t *testing.T, names ...string) (*fakeStore, []profiles.Profile) {
	t.Helper()
	store := &fakeStore{}
	for _, name := range names {
		profile := sampleProfile(t, name, "", string(profiles.SSH), "host.example.com", 22)
		profile.Username = "tester"
		store.profiles = append(store.profiles, profile)
	}
	return store, store.profiles
}

func tabIDs(s *screen) []string {
	byTab := map[any]string{}
	for id, tab := range s.openTabs {
		byTab[tab] = id
	}
	ids := make([]string, 0, len(byTab))
	for _, item := range s.tabs.Items {
		if id, known := byTab[item]; known {
			ids = append(ids, id)
		}
	}
	return ids
}

func TestScreen_OpenAndCloseTabs_ArePersisted(t *testing.T) {
	store, saved := storeWithProfiles(t, "First", "Second")
	s := newTestScreen(t, store)

	s.openSessionTab(saved[0])
	s.openSessionTab(saved[1])
	if got := store.openTabs; len(got) != 2 || got[0] != saved[0].ID || got[1] != saved[1].ID {
		t.Fatalf("stored tabs = %v, want both in tab order", got)
	}
	if store.activeTab != saved[1].ID {
		t.Errorf("active tab = %q, want the selected one", store.activeTab)
	}

	s.closeSessionTab(saved[1].ID)

	if got := store.openTabs; len(got) != 1 || got[0] != saved[0].ID {
		t.Errorf("stored tabs after closing = %v, want only the first", got)
	}
}

func TestScreen_RestoreSessions_ReopensTheTabsUnconnected(t *testing.T) {
	store, saved := storeWithProfiles(t, "First", "Second")
	store.openTabs = []string{saved[0].ID, saved[1].ID}
	store.activeTab = saved[0].ID

	s := newTestScreen(t, store)

	if got := tabIDs(s); len(got) != 2 || got[0] != saved[0].ID || got[1] != saved[1].ID {
		t.Fatalf("restored tabs = %v, want both in order", got)
	}
	for _, profile := range saved {
		panel := s.panels[profile.ID]
		if panel == nil {
			t.Fatalf("no panel for %q", profile.Name)
		}
		if panel.hasSession() || panel.term != nil {
			t.Errorf("tab %q was restored connected", profile.Name)
		}
		if panel.connect.Disabled() {
			t.Errorf("tab %q cannot be connected: Connect is disabled", profile.Name)
		}
	}
	if s.tabs.Selected() != s.openTabs[saved[0].ID] {
		t.Error("the previously active tab is not the selected one")
	}
}

func TestScreen_RestoreSessions_SkipsConnectionsThatAreGone(t *testing.T) {
	store, saved := storeWithProfiles(t, "Only")
	store.openTabs = []string{"deleted-id", saved[0].ID}

	s := newTestScreen(t, store)

	if got := tabIDs(s); len(got) != 1 || got[0] != saved[0].ID {
		t.Fatalf("restored tabs = %v, want only the connection that still exists", got)
	}
	if len(store.openTabs) != 1 {
		t.Errorf("stored tabs = %v, want the missing id pruned", store.openTabs)
	}
}

func TestScreen_RestoreSessions_PreferenceOff_OpensNoTabs(t *testing.T) {
	store, saved := storeWithProfiles(t, "First")
	store.openTabs = []string{saved[0].ID}

	screen := newTestScreenWithPreferences(t, store, func(prefs fyne.Preferences) {
		prefs.SetBool(prefReopenTabs, false)
	})

	if got := tabIDs(screen); len(got) != 0 {
		t.Errorf("restored tabs = %v, want none when the preference is off", got)
	}
}

func TestScreen_SelectedTab_IsRememberedAcrossRestarts(t *testing.T) {
	store, saved := storeWithProfiles(t, "First", "Second")
	s := newTestScreen(t, store)
	s.openSessionTab(saved[0])
	s.openSessionTab(saved[1])
	s.tabs.Select(s.openTabs[saved[0].ID])

	again := newTestScreen(t, store)

	if again.tabs.Selected() != again.openTabs[saved[0].ID] {
		t.Errorf("selected tab after restart is not the one that was active; stored = %q", store.activeTab)
	}
	if !strings.Contains(store.activeTab, saved[0].ID) {
		t.Errorf("stored active tab = %q, want the first connection", store.activeTab)
	}
}

// TestScreen_OtherSaves_KeepTheOpenTabs guards the shared write path: renaming a
// connection rewrites the whole document, and it must carry the tabs with it
// instead of forgetting them.
func TestScreen_OtherSaves_KeepTheOpenTabs(t *testing.T) {
	store, saved := storeWithProfiles(t, "First")
	s := newTestScreen(t, store)
	s.openSessionTab(saved[0])

	renamed := saved[0]
	renamed.Name = "Renamed"
	if err := s.ws.renameProfile(renamed); err != nil {
		t.Fatalf("renameProfile: %v", err)
	}

	if len(store.openTabs) != 1 || store.openTabs[0] != saved[0].ID {
		t.Errorf("open tabs after an unrelated save = %v, want the tab kept", store.openTabs)
	}
}
