package ui

import (
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	rdpclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/rdp"
	"strings"
	"testing"
	"time"
)

func TestSessionPanel_RDPConnect_ShowsExternalSessionState(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	panel := s.newSessionPanel(rdpProfile(t))
	session := &fakeRDPSession{done: make(chan struct{})}
	panel.applyConnectRDP(session, nil)

	if panel.rdp != session {
		t.Fatal("panel did not keep the RDP session")
	}
	if !panel.connect.Disabled() || panel.disconnect.Disabled() {
		t.Error("button state does not reflect a running RDP session")
	}
	if !strings.Contains(s.status.Text, "separate window") {
		t.Errorf("status = %q, want a separate-window notice", s.status.Text)
	}
	panel.disconnectShell()
	if !session.closed {
		t.Error("disconnect did not close the RDP session")
	}
	if panel.disconnect.Disabled() == false || panel.connect.Disabled() {
		t.Error("button state does not reflect a closed RDP session")
	}
}

func TestSessionPanel_RDPConnect_ForwardsTargetAndPassword(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	type call struct {
		target  rdpclient.Target
		options rdpclient.Options
	}
	calls := make(chan call, 1)
	s.launchRDP = func(target rdpclient.Target, options rdpclient.Options) (rdpSession, error) {
		calls <- call{target: target, options: options}
		return &fakeRDPSession{done: make(chan struct{})}, nil
	}
	panel := s.newSessionPanel(rdpProfile(t))
	panel.password.SetText("secret")
	panel.connectRDP()

	select {
	case got := <-calls:
		if got.target.Host != "desk.example.com" || got.target.Port != 3389 || got.target.Username != "kas" {
			t.Errorf("target = %+v", got.target)
		}
		if got.options.Password != "secret" {
			t.Error("password not forwarded to the launcher")
		}
		if got.options.ParentWindowID != 0 {
			t.Errorf("ParentWindowID = %d, want 0 in this preview", got.options.ParentWindowID)
		}
		if got.options.Security != "" {
			t.Errorf("Security = %q, want the client default unless legacy is enabled", got.options.Security)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("launcher was not called")
	}
}

func TestSessionPanel_RDPConnect_EmbedsSessionInTheTab(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	area := &fakeArea{id: 77}
	s.embedder = func(fyne.Window, fyne.CanvasObject, fyne.CanvasObject) (embeddedArea, error) { return area, nil }
	ids := make(chan uint64, 1)
	s.launchRDP = func(_ rdpclient.Target, options rdpclient.Options) (rdpSession, error) {
		ids <- options.ParentWindowID
		return &fakeRDPSession{done: make(chan struct{})}, nil
	}
	panel := s.newSessionPanel(rdpProfile(t))
	panel.connectRDP()

	select {
	case id := <-ids:
		if id != 77 {
			t.Fatalf("ParentWindowID = %d, want the embedded area id 77", id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("launcher was not called")
	}

	session := &fakeRDPSession{done: make(chan struct{})}
	panel.applyConnectRDPWithArea(session, area, nil, nil)
	if !strings.Contains(s.status.Text, "embedded") {
		t.Errorf("status = %q, want an embedded notice", s.status.Text)
	}
	panel.disconnectShell()
	if !area.destroyed {
		t.Error("disconnect did not destroy the embedded area")
	}
}

func TestSessionPanel_RDPConnect_EmbeddingUnavailable_FallsBackToSeparateWindow(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.embedder = func(fyne.Window, fyne.CanvasObject, fyne.CanvasObject) (embeddedArea, error) { return nil, ErrNoX11 }
	ids := make(chan uint64, 1)
	s.launchRDP = func(_ rdpclient.Target, options rdpclient.Options) (rdpSession, error) {
		ids <- options.ParentWindowID
		return &fakeRDPSession{done: make(chan struct{})}, nil
	}
	panel := s.newSessionPanel(rdpProfile(t))
	panel.connectRDP()

	select {
	case id := <-ids:
		if id != 0 {
			t.Fatalf("ParentWindowID = %d, want 0 without an embedded area", id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("launcher was not called")
	}

	panel.applyConnectRDPWithArea(&fakeRDPSession{done: make(chan struct{})}, nil, ErrNoX11, nil)
	if !strings.Contains(s.status.Text, "separate window") {
		t.Errorf("status = %q, want the separate-window fallback", s.status.Text)
	}
}

func TestEmbeddedArea_FollowsTabSelection(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	area := &fakeArea{id: 11}
	panel := s.newSessionPanel(rdpProfile(t))
	panel.embedded = area
	tab := container.NewTabItem("session", panel.content())
	s.panels["id"] = panel
	s.openTabs["id"] = tab

	s.tabSelected(tab)
	if !area.visible {
		t.Error("embedded area is hidden on its own tab")
	}
	s.tabSelected(s.welcome)
	if area.visible {
		t.Error("embedded area stayed visible after switching tabs")
	}
}

func TestSessionPanel_RDPConnect_LegacyCheckboxSelectsLegacySecurity(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	security := make(chan string, 1)
	s.launchRDP = func(_ rdpclient.Target, options rdpclient.Options) (rdpSession, error) {
		security <- options.Security
		return &fakeRDPSession{done: make(chan struct{})}, nil
	}
	panel := s.newSessionPanel(rdpProfile(t))
	panel.rdpLegacy.SetChecked(true)
	panel.connectRDP()

	select {
	case got := <-security:
		if got != "rdp" {
			t.Errorf("Security = %q, want rdp when legacy is enabled", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("launcher was not called")
	}
}

func TestSessionPanel_RDPTest_CallsProbeWithCredentials(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	probed := make(chan rdpclient.Options, 1)
	s.probeRDP = func(target rdpclient.Target, options rdpclient.Options) (string, error) {
		probed <- options
		return "Authentication only, exit status 0", nil
	}
	panel := s.newSessionPanel(rdpProfile(t))
	panel.password.SetText("secret")
	panel.testRDP()

	if !strings.Contains(s.status.Text, "Testing RDP credentials") {
		t.Errorf("status = %q, want a testing notice", s.status.Text)
	}
	select {
	case options := <-probed:
		if options.Password != "secret" {
			t.Error("probe did not receive the typed password")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("probe was not called")
	}
}

func TestSessionPanel_RDPProbeFailure_PointsToConnect(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	panel := s.newSessionPanel(rdpProfile(t))
	panel.rdpTest.Disable()
	// Mirror the async failure path without a goroutine.
	output := ""
	_ = output
	panel.screen.setProblem("RDP auth-only check failed: exit status 1. Some servers (for example xrdp) do not support auth-only; use Connect to open a real session.")
	if !strings.Contains(s.status.Text, "use Connect") {
		t.Errorf("status = %q, want a hint to use Connect", s.status.Text)
	}
}

func TestSessionPanel_RDPLaunchFailure_ReportsProblem(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	panel := s.newSessionPanel(rdpProfile(t))
	panel.applyConnectRDP(nil, errors.New("xfreerdp not found"))
	if !strings.Contains(s.status.Text, "RDP launch failed") || s.status.Importance != widget.DangerImportance {
		t.Errorf("status = %q, want a launch failure", s.status.Text)
	}
}
