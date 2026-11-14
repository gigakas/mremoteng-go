package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	rdpclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/rdp"
)

func TestSpecialChords_PutModifiersFirstAndEndWithTheKey(t *testing.T) {
	if len(specialChords) < 3 {
		t.Fatalf("chords = %d, want the set users cannot type", len(specialChords))
	}
	if specialChords[0].Label != "Ctrl+Alt+Del" {
		t.Errorf("first chord = %q, want the one the bar button sends", specialChords[0].Label)
	}
	for _, chord := range specialChords {
		if len(chord.Keysyms) < 2 {
			t.Errorf("%s has %d keysyms, want a modifier and a key", chord.Label, len(chord.Keysyms))
		}
		if chord.Keysyms[len(chord.Keysyms)-1] == keysymControlL || chord.Keysyms[len(chord.Keysyms)-1] == keysymAltL {
			t.Errorf("%s ends with a modifier", chord.Label)
		}
		if strings.TrimSpace(chord.Label) == "" {
			t.Error("a chord has no label")
		}
	}
}

// TestSendSpecialKey_TypesTheChordIntoTheSession is the contract of the feature: the
// chord reaches the embedded window instead of the local desktop.
func TestSendSpecialKey_TypesTheChordIntoTheSession(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	area := &fakeArea{id: 31}
	panel := s.newSessionPanel(sampleProfile(t, "desk", "", string(profiles.RDP), "desk.example.com", 3389))
	panel.embedded = area

	panel.sendSpecialKey(specialChords[0])

	if len(area.sent) != 1 {
		t.Fatalf("sent = %v, want one chord", area.sent)
	}
	if len(area.sent[0]) != 3 || area.sent[0][2] != keysymDelete {
		t.Errorf("chord = %x, want Ctrl+Alt+Delete", area.sent[0])
	}
	if !strings.Contains(s.status.Text, "Sent Ctrl+Alt+Del") {
		t.Errorf("status = %q, want the chord reported", s.status.Text)
	}
}

func TestSendSpecialKey_WithoutAnEmbeddedSession_ExplainsWhy(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	panel := s.newSessionPanel(sampleProfile(t, "desk", "", string(profiles.RDP), "desk.example.com", 3389))

	panel.sendSpecialKey(specialChords[0])

	if !strings.Contains(s.status.Text, "embedded in the tab") {
		t.Errorf("status = %q, want the reason", s.status.Text)
	}
}

func TestSendSpecialKey_ClientFailure_IsReported(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	area := &fakeArea{id: 32, fail: errors.New("XTEST unavailable")}
	panel := s.newSessionPanel(sampleProfile(t, "desk", "", string(profiles.RDP), "desk.example.com", 3389))
	panel.embedded = area

	panel.sendSpecialKey(specialChords[0])

	if !strings.Contains(s.status.Text, "Could not send Ctrl+Alt+Del") {
		t.Errorf("status = %q, want the failure", s.status.Text)
	}
	if s.notifications.count() == 0 {
		t.Error("the failure was not recorded as a notification")
	}
}

// TestSpecialKeysButton_IsOnTheRDPBar keeps the control where the user can reach it: a
// chord cannot be typed while the session owns the keyboard.
func TestSpecialKeysButton_IsOnTheRDPBar(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	panel := s.newSessionPanel(sampleProfile(t, "desk", "", string(profiles.RDP), "desk.example.com", 3389))
	onBar := map[fyne.CanvasObject]bool{}
	collectObjects(panel.connectionBar(), onBar)

	found := false
	for object := range onBar {
		if button, ok := object.(*hintButton); ok && strings.Contains(button.tip, "Ctrl+Alt+Del") {
			found = true
		}
	}
	if !found {
		t.Error("the RDP session bar has no special-keys control")
	}

	ssh := s.newSessionPanel(sshPanelProfile(t, "tester"))
	sshBar := map[fyne.CanvasObject]bool{}
	collectObjects(ssh.connectionBar(), sshBar)
	for object := range sshBar {
		if button, ok := object.(*hintButton); ok && strings.Contains(button.tip, "Ctrl+Alt+Del") {
			t.Error("an SSH session offers RDP special keys")
		}
	}
}

func TestSendChordToSelected_ActsOnTheOpenTab(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	profile := sampleProfile(t, "desk", "", string(profiles.RDP), "desk.example.com", 3389)
	s.openSessionTab(profile)
	area := &fakeArea{id: 33}
	s.panels[profile.ID].embedded = area

	s.sendChordToSelected(specialChords[1])

	if len(area.sent) != 1 || area.sent[0][len(area.sent[0])-1] != keysymEnd {
		t.Fatalf("sent = %v, want Ctrl+Alt+End", area.sent)
	}
}

func TestToggleFullScreen_TogglesTheWindowAndReportsIt(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})

	s.toggleFullScreen()
	if !s.win.FullScreen() {
		t.Fatal("the window did not go full screen")
	}
	if !strings.Contains(s.status.Text, "Full screen") {
		t.Errorf("status = %q, want the state reported", s.status.Text)
	}
	if item := s.fullScreenItem(); !item.Checked {
		t.Error("the menu item does not show the window state")
	}

	s.toggleFullScreen()
	if s.win.FullScreen() {
		t.Error("the window stayed full screen")
	}
	if item := s.fullScreenItem(); item.Checked {
		t.Error("the menu item still claims full screen")
	}
}

// TestConnectRDP_MultiMonitor_DoesNotEmbed covers the flag's meaning: a session that
// spans monitors needs FreeRDP's own window, so it must not be confined to the tab.
func TestConnectRDP_MultiMonitor_DoesNotEmbed(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	parent := make(chan uint64, 1)
	s.embedder = func(fyne.Window, fyne.CanvasObject, fyne.CanvasObject) (embeddedArea, error) {
		return &fakeArea{id: 41}, nil
	}
	s.launchRDP = func(_ rdpclient.Target, options rdpclient.Options) (rdpSession, error) {
		parent <- options.ParentWindowID
		return &fakeRDPSession{done: make(chan struct{})}, nil
	}
	profile := sampleProfile(t, "desk", "", string(profiles.RDP), "desk.example.com", 3389)
	profile.RDP.MultiMonitor = true
	panel := s.newSessionPanel(profile)

	panel.connectRDP()

	select {
	case window := <-parent:
		if window != 0 {
			t.Errorf("ParentWindowID = %d, want 0 so FreeRDP opens its own window", window)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the launcher was not called")
	}
	if !waitFor(t, func() bool { return strings.Contains(s.status.Text, "separate window") }) {
		t.Errorf("status = %q, want the session reported as a separate window", s.status.Text)
	}
}

// TestFullScreenItem_IsInTheViewMenu keeps a way out of full screen.
func TestFullScreenItem_IsInTheViewMenu(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	var view *fyne.Menu
	for _, menu := range s.menu().Items {
		if menu.Label == "View" {
			view = menu
		}
	}
	if view == nil {
		t.Fatal("no View menu")
	}
	labels := make([]string, 0, len(view.Items))
	for _, item := range view.Items {
		labels = append(labels, item.Label)
	}
	joined := strings.Join(labels, ",")
	for _, want := range []string{"Full screen", "Send keys"} {
		if !strings.Contains(joined, want) {
			t.Errorf("View menu = %v, want %q", labels, want)
		}
	}
	var send *fyne.MenuItem
	for _, item := range view.Items {
		if item.Label == "Send keys" {
			send = item
		}
	}
	if send == nil || send.ChildMenu == nil || len(send.ChildMenu.Items) != len(specialChords) {
		t.Errorf("Send keys has no chord list: %+v", send)
	}
}
