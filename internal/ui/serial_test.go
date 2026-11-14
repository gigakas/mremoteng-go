package ui

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	serialclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/serial"
)

// openStream is a serial port that stays open: the real port blocks on Read until the
// device sends something, and a fake that reports EOF would close the session before
// the test can look at it.
type openStream struct {
	release chan struct{}
	written []byte
	closed  bool
}

func newOpenStream() *openStream { return &openStream{release: make(chan struct{})} }

func (s *openStream) Read(payload []byte) (int, error) {
	<-s.release
	return 0, io.EOF
}

func (s *openStream) Write(payload []byte) (int, error) {
	s.written = append(s.written, payload...)
	return len(payload), nil
}

func (s *openStream) Close() error { s.closed = true; close(s.release); return nil }

func serialProfile(t *testing.T) profiles.Profile {
	t.Helper()
	profile := sampleProfile(t, "Console", "", string(profiles.SERIAL), "/dev/ttyS0", 0)
	profile.Serial = profiles.SerialOptions{Baud: 115200, DataBits: 8, Parity: "none", StopBits: 1}
	return profile
}

func TestSessionPanel_ConnectSerial_OpensThePortAndInstallsTheTerminal(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	configs := make(chan serialclient.Config, 1)
	s.openSerial = func(config serialclient.Config) (streamSession, error) {
		configs <- config
		return newOpenStream(), nil
	}
	profile := serialProfile(t)
	s.openSessionTab(profile)
	panel := s.panels[profile.ID]

	panel.connectSerial()

	select {
	case config := <-configs:
		if config.Device != "/dev/ttyS0" || config.Baud != 115200 {
			t.Errorf("config = %+v, want the stored line settings", config)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the port was not opened")
	}
	// The terminal is installed before the status line is written, so wait for the
	// status: it is the last thing startStream does.
	if !waitFor(t, func() bool { return strings.Contains(s.status.Text, "session with /dev/ttyS0 connected") }) {
		t.Fatalf("status = %q, want the session reported", s.status.Text)
	}
	if panel.term == nil {
		t.Error("the terminal was not installed over the port")
	}
	if !strings.Contains(s.status.Text, "115200 8N1") {
		t.Errorf("status = %q, want the line settings named", s.status.Text)
	}
}

func TestSessionPanel_ConnectSerial_InvalidSettings_ReportsWithoutOpening(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	opened := false
	s.openSerial = func(serialclient.Config) (streamSession, error) {
		opened = true
		return &fakeStream{}, nil
	}
	profile := serialProfile(t)
	profile.Serial.Baud = 1234
	panel := s.newSessionPanel(profile)

	panel.connectSerial()

	if opened {
		t.Error("the port was opened with settings the device cannot use")
	}
	if !strings.Contains(s.status.Text, "Cannot open the serial port") {
		t.Errorf("status = %q, want the reason", s.status.Text)
	}
}

func TestSessionPanel_ConnectSerial_OpenFailure_IsReported(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.openSerial = func(serialclient.Config) (streamSession, error) {
		return nil, errors.New("permission denied")
	}
	profile := serialProfile(t)
	s.openSessionTab(profile)
	panel := s.panels[profile.ID]

	panel.connectSerial()

	if !waitFor(t, func() bool { return strings.Contains(s.status.Text, "permission denied") }) {
		t.Errorf("status = %q, want the failure", s.status.Text)
	}
	if panel.term != nil {
		t.Error("a terminal was installed over a port that did not open")
	}
}

func TestSerialOptionsFromFields_RejectsWhatThePortCannotDo(t *testing.T) {
	options, err := serialOptionsFromFields("9600", "8", "none", "1")
	if err != nil || options.Baud != 9600 || options.DataBits != 8 || options.Parity != "none" || options.StopBits != 1 {
		t.Fatalf("options = %+v, %v", options, err)
	}
	cases := map[string][4]string{
		"unknown baud":      {"1234", "8", "none", "1"},
		"nine bits":         {"9600", "9", "none", "1"},
		"bad parity":        {"9600", "8", "sometimes", "1"},
		"three stops":       {"9600", "8", "none", "3"},
		"not a number":      {"fast", "8", "none", "1"},
		"bits not a number": {"9600", "eight", "none", "1"},
	}
	for name, fields := range cases {
		if _, err := serialOptionsFromFields(fields[0], fields[1], fields[2], fields[3]); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestSessionPanel_SaveSerialOptions_ReachesTheProfile(t *testing.T) {
	profile := serialProfile(t)
	store := &fakeStore{profiles: []profiles.Profile{profile}}
	s := newTestScreen(t, store)
	panel := s.newSessionPanel(profile)

	if err := panel.saveSerialOptions(profiles.SerialOptions{Baud: 19200, DataBits: 7, Parity: "even", StopBits: 2}); err != nil {
		t.Fatalf("saveSerialOptions: %v", err)
	}

	stored, _ := s.profileByID(profile.ID)
	if stored.Serial.Baud != 19200 || stored.Serial.Parity != "even" {
		t.Errorf("stored = %+v, want the new line settings", stored.Serial)
	}
	if panel.profile.Serial.Baud != 19200 {
		t.Error("the panel still holds the old settings")
	}
	if !strings.Contains(panel.serialConfig().Description(), "19200 7E2") {
		t.Errorf("description = %q, want the saved settings", panel.serialConfig().Description())
	}
}

func TestSessionPanel_SaveSerialOptions_ReadOnlyStore_ReportsIt(t *testing.T) {
	profile := serialProfile(t)
	store := &fakeStore{profiles: []profiles.Profile{profile}}
	s := newTestScreen(t, store)
	panel := s.newSessionPanel(profile)
	store.saveErr = errors.New("disk full")

	if err := panel.saveSerialOptions(profiles.SerialOptions{Baud: 9600}); err == nil {
		t.Fatal("a rejected write was reported as success")
	}
	if panel.profile.Serial.Baud != 115200 {
		t.Error("the panel changed although nothing was stored")
	}
}

// TestSerialOptionsButton_OnlyOnSerialSessions keeps the control where it belongs.
func TestSerialOptionsButton_OnlyOnSerialSessions(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	serial := s.newSessionPanel(serialProfile(t))
	onBar := map[fyne.CanvasObject]bool{}
	collectObjects(serial.connectionBar(), onBar)
	found := false
	for object := range onBar {
		if button, ok := object.(*hintButton); ok && strings.Contains(button.tip, "Serial line settings") {
			found = true
		}
	}
	if !found {
		t.Error("the serial session bar has no line-settings control")
	}

	ssh := s.newSessionPanel(sshPanelProfile(t, "tester"))
	sshBar := map[fyne.CanvasObject]bool{}
	collectObjects(ssh.connectionBar(), sshBar)
	for object := range sshBar {
		if button, ok := object.(*hintButton); ok && strings.Contains(button.tip, "Serial line settings") {
			t.Error("an SSH session offers serial line settings")
		}
	}
}
