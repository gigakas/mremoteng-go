package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

// This file holds how the window opens, closes and restores its session tabs.

// tabSelected keeps an embedded RDP window visible only on its own tab; Fyne
// tabs share one X11 window, so an orphaned child would cover the whole app.
func (s *screen) tabSelected(item *container.TabItem) {
	s.applyEmbeddedVisibility(item)
	s.applyTerminalVisibility(item)
	for id, panel := range s.panels {
		if s.openTabs[id] == item && panel.term != nil {
			// The terminal of the selected tab owns the keyboard.
			panel.term.focus()
		}
	}
	s.rememberSessions()
}

func (s *screen) tabClosed(item *container.TabItem) {
	for id, tab := range s.openTabs {
		if tab == item {
			// Closing a session tab ends its session, which removes the embedded
			// window instead of leaving it behind.
			for _, panel := range s.tabPanels(id) {
				panel.disconnectShell()
			}
			delete(s.openTabs, id)
			delete(s.panels, id)
		}
	}
	if len(s.tabs.Items) == 0 {
		s.tabs.Append(s.welcome)
		s.tabs.Select(s.welcome)
	}
	s.rememberSessions()
}

func (s *screen) openSessionTab(profile profiles.Profile) {
	if tab, ok := s.openTabs[profile.ID]; ok {
		tab.Text = profile.Name
		if panel, known := s.panels[profile.ID]; known {
			panel.profile = profile
			// Rebuild the bar with the new name/endpoint; the slot keeps the live
			// terminal widget, so an open session survives a rename. A tab that
			// was split keeps its layout as it is.
			if panel.pane == nil {
				tab.Content = panel.content()
			} else if root := panel.pane.root(); root.panel != nil {
				root.view.Objects = []fyne.CanvasObject{panel.content()}
				root.view.Refresh()
			}
		}
		s.tabs.Refresh()
		s.tabs.Select(tab)
		s.rememberSessions()
		return
	}
	panel := s.newSessionPanel(profile)
	tab := container.NewTabItem(profile.Name, newRootPane(panel).view)
	s.openTabs[profile.ID] = tab
	s.panels[profile.ID] = panel
	s.tabs.Append(tab)
	s.tabs.Select(tab)
	s.rememberSessions()
}

// rememberSessions stores the tabs the window has open, in tab order, and the
// selected one. The write is best effort and its error is dropped on purpose: this
// is window state, and a read-only or full disk must not interrupt the click that
// changed a tab.
func (s *screen) rememberSessions() {
	if s.restoring || s.ws == nil || s.tabs == nil {
		return
	}
	byTab := make(map[*container.TabItem]string, len(s.openTabs))
	for id, tab := range s.openTabs {
		byTab[tab] = id
	}
	ids := make([]string, 0, len(byTab))
	kept := map[string]bool{}
	for _, item := range s.tabs.Items {
		id, known := byTab[item]
		if !known {
			continue
		}
		// A quick-connect tab has no stored profile, so reopening it would show a tab
		// for a connection that cannot come back. Leave it out.
		if _, stored := s.profileByID(id); !stored {
			continue
		}
		ids = append(ids, id)
		kept[id] = true
	}
	layouts := make(map[string]profiles.PaneLayout, len(ids))
	for _, id := range ids {
		panel := s.panels[id]
		if panel == nil || panel.pane == nil {
			continue
		}
		if layout := layoutOf(panel.pane.root()); layout != nil {
			layouts[id] = *layout
		}
	}
	selected, _ := byTab[s.tabs.Selected()]
	if !kept[selected] {
		selected = ""
	}
	// Nothing to record and nothing recorded: skip the write, so a quick-connect tab
	// or a window with no sessions does not touch the file on every click.
	if selected == s.ws.active && sameSessions(ids, s.ws.sessions) && sameLayouts(layouts, s.ws.layouts) {
		return
	}
	_ = s.ws.setSessions(ids, selected, layouts)
}

// sameSessions reports whether two tab lists are equal and in the same order.
func sameSessions(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// restoreSessions reopens the tabs the previous run left open, unconnected: no
// credential is stored, so each tab waits with Connect enabled for the user. Tabs
// whose connection no longer exists are dropped, and the pruned list is written
// back so the store stops carrying them.
func (s *screen) restoreSessions() {
	if !s.prefs.reopenTabs() || len(s.ws.sessions) == 0 {
		return
	}
	s.restoring = true
	for _, id := range s.ws.sessions {
		profile, known := s.profileByID(id)
		if !known {
			continue
		}
		s.openSessionTab(profile)
		if layout, divided := s.ws.layouts[id]; divided {
			if panel, open := s.panels[id]; open {
				s.restorePanes(panel, &layout)
			}
		}
	}
	s.restoring = false
	if active, known := s.profileByID(s.ws.active); known {
		if tab, open := s.openTabs[active.ID]; open {
			s.tabs.Select(tab)
		}
	}
	s.rememberSessions()
}

// applyTerminalVisibility draws the terminals of the selected tab only. Every
// terminal keeps consuming its session's output; the hidden ones just skip the
// repaint work until their tab comes back.
func (s *screen) applyTerminalVisibility(selected *container.TabItem) {
	for id := range s.panels {
		hidden := s.openTabs[id] != selected
		for _, panel := range s.tabPanels(id) {
			if panel.term != nil {
				panel.term.setPaused(hidden)
			}
		}
	}
}

// panelShown reports whether the panel is in the selected tab. A panel that no
// tab owns yet counts as shown, so a terminal never starts out frozen.
func (s *screen) panelShown(panel *sessionPanel) bool {
	if s.tabs == nil {
		return true
	}
	for id := range s.panels {
		for _, candidate := range s.tabPanels(id) {
			if candidate == panel {
				return s.openTabs[id] == s.tabs.Selected()
			}
		}
	}
	return true
}
