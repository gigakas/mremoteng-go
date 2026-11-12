package ui

import (
	"fyne.io/fyne/v2"
	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	telnetclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/telnet"
	"strings"
	"testing"
)

func TestSessionPanel_TelnetConnect_ShowsTerminalAndSendsInput(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	stream := &fakeStream{}
	var gotMode telnetclient.Mode
	s.dialStream = func(_ string, config telnetclient.Config) (streamSession, error) {
		gotMode = config.Mode
		return stream, nil
	}
	profile := sampleProfile(t, "Router", "", "TELNET", "router.example.com", 23)
	panel := s.newSessionPanel(profile)

	panel.startStream(stream, telnetclient.ModeTelnet)
	if gotMode != "" {
		t.Fatal("dialer ran before startStream")
	}
	if panel.term == nil || len(panel.slot.Objects) != 1 {
		t.Fatal("terminal view was not installed")
	}
	if panel.term.onCopy == nil {
		t.Error("telnet terminal is not wired for clipboard copy")
	}
	if panel.term.zoom == nil {
		t.Error("telnet terminal has no grid-level zoom theme")
	}
	panel.term.TypedRune('a')
	panel.term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnter})
	if string(stream.written) != "a\r" {
		t.Errorf("written = %q, want the typed input", stream.written)
	}
	panel.disconnectShell()
	if !stream.closed {
		t.Error("disconnect did not close the stream")
	}
}

func TestStreamMode_MapsProtocols(t *testing.T) {
	cases := map[profiles.Protocol]telnetclient.Mode{
		profiles.TELNET: telnetclient.ModeTelnet,
		profiles.RAW:    telnetclient.ModeRaw,
		profiles.RLOGIN: telnetclient.ModeRlogin,
	}
	for protocol, want := range cases {
		if got, ok := streamMode(protocol); !ok || got != want {
			t.Errorf("streamMode(%s) = %q, %v", protocol, got, ok)
		}
	}
	if _, ok := streamMode(profiles.SSH); ok {
		t.Error("SSH mapped to a stream protocol")
	}
}

func TestHTTPConnect_OpensBrowser(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	var opened string
	s.openBrowser = func(url string) error { opened = url; return nil }
	profile := sampleProfile(t, "Intranet", "", "HTTP", "intranet.example.com", 80)
	panel := s.newSessionPanel(profile)
	panel.connectHTTP()
	if opened != "http://intranet.example.com" {
		t.Fatalf("opened %q, want the address without :80", opened)
	}
	if !strings.Contains(s.status.Text, "browser") {
		t.Errorf("status = %q, want a browser notice", s.status.Text)
	}
}

func TestQuickConnect_OpensTemporaryTabWithoutSaving(t *testing.T) {
	store := &fakeStore{}
	s := newTestScreen(t, store)
	s.quickConnect("lab.example.com", string(profiles.SSH), "", "demo")

	if store.saves != 0 || len(store.profiles) != 0 {
		t.Fatalf("quick connect wrote %d profiles to the store", len(store.profiles))
	}
	if len(s.openTabs) != 1 || len(s.panels) != 1 {
		t.Fatalf("tabs = %d, want one temporary session", len(s.openTabs))
	}
	for _, panel := range s.panels {
		if panel.profile.Host != "lab.example.com" || panel.profile.Port != 22 || panel.profile.Username != "demo" {
			t.Errorf("profile = %+v", panel.profile)
		}
		if !strings.Contains(panel.profile.Name, "Quick:") {
			t.Errorf("name = %q, want a Quick prefix", panel.profile.Name)
		}
	}
}

func TestQuickConnect_InvalidInput_DoesNotOpenTab(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.quickConnect("", string(profiles.SSH), "", "")
	if len(s.openTabs) != 0 {
		t.Fatal("empty host opened a session")
	}
	s.quickConnect("lab.example.com", string(profiles.SSH), "not-a-port", "")
	if len(s.openTabs) != 0 {
		t.Fatal("invalid port opened a session")
	}
	if !strings.Contains(s.status.Text, "must be a number") {
		t.Errorf("status = %q, want a port error", s.status.Text)
	}
}
