package ui

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	rdpclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/rdp"
)

// storedRDPProfile is an RDP connection with every stored option set.
func storedRDPProfile(t *testing.T) profiles.Profile {
	t.Helper()
	profile := rdpProfile(t)
	profile.RDP = profiles.RDPOptions{
		Clipboard:         true,
		Printers:          true,
		Audio:             true,
		Drives:            []string{"share:/srv/share"},
		DynamicResolution: true,
		MultiMonitor:      true,
		LegacyTLS:         true,
		IgnoreCertificate: true,
		Gateway:           profiles.Gateway{Host: "gw.example.com", Port: 8443, UsageMethod: "direct"},
	}
	return profile
}

func TestRDPOptionsFromProfile_MapsEveryStoredField(t *testing.T) {
	options := rdpOptionsFromProfile(storedRDPProfile(t))

	if !options.Clipboard || !options.Printers || !options.Audio || !options.DynamicResolution || !options.MultiMonitor {
		t.Errorf("forwarding and display options were dropped: %+v", options)
	}
	if len(options.Drives) != 1 || options.Drives[0] != "share:/srv/share" {
		t.Errorf("drives = %v, want the stored share", options.Drives)
	}
	if options.Security != "rdp" {
		t.Errorf("security = %q, want the legacy mode from the stored toggle", options.Security)
	}
	if !options.IgnoreCertificate {
		t.Error("the stored certificate choice was dropped")
	}
	if options.Gateway.Host != "gw.example.com" || options.Gateway.Port != 8443 || options.Gateway.UsageMethod != "direct" {
		t.Errorf("gateway = %+v, want the stored hop", options.Gateway)
	}
	if options.Password != "" {
		t.Error("a password appeared in the stored options")
	}
}

func TestRDPOptionsToProfile_RoundTripsTheStoredChoices(t *testing.T) {
	profile := storedRDPProfile(t)
	again := rdpOptionsToProfile(rdpOptionsFromProfile(profile))
	if !sameRDPOptions(profile.RDP, again) {
		t.Errorf("round trip = %+v, want %+v", again, profile.RDP)
	}
	// A comma-separated entry with a trailing comma must not become a drive.
	trimmed := rdpOptionsToProfile(rdpclient.Options{Drives: []string{"share:/srv/share", " ", ""}})
	if len(trimmed.Drives) != 1 {
		t.Errorf("drives = %v, want the empties dropped", trimmed.Drives)
	}
	if trimmed.LegacyTLS {
		t.Error("legacy TLS appeared without being selected")
	}
}

// sameRDPOptions compares two option blocks, drives included.
func sameRDPOptions(left, right profiles.RDPOptions) bool {
	if len(left.Drives) != len(right.Drives) {
		return false
	}
	for i := range left.Drives {
		if left.Drives[i] != right.Drives[i] {
			return false
		}
	}
	left.Drives, right.Drives = nil, nil
	return reflect.DeepEqual(left, right)
}

func TestSessionPanel_SaveRDPOptions_WritesTheProfile(t *testing.T) {
	store := &fakeStore{profiles: []profiles.Profile{storedRDPProfile(t)}}
	s := newTestScreen(t, store)
	profile := store.profiles[0]
	panel := s.newSessionPanel(profile)

	options := profile.RDP
	options.Audio = false
	options.Gateway = profiles.Gateway{Host: "gw2.example.com"}
	if err := panel.saveRDPOptions(options); err != nil {
		t.Fatalf("saveRDPOptions: %v", err)
	}

	stored, known := s.profileByID(profile.ID)
	if !known {
		t.Fatal("the profile disappeared from the store")
	}
	if stored.RDP.Audio || stored.RDP.Gateway.Host != "gw2.example.com" {
		t.Errorf("stored options = %+v, want the new ones", stored.RDP)
	}
	if panel.profile.RDP.Gateway.Host != "gw2.example.com" {
		t.Error("the panel still holds the old options")
	}
	if store.saves == 0 {
		t.Error("nothing was written to the store")
	}
}

// TestSessionPanel_SaveRDPOptions_StartsFromTheStoredProfile covers the rename
// race: the Config panel may have renamed the connection since the tab opened, and
// saving only the options must not write the old name back.
func TestSessionPanel_SaveRDPOptions_StartsFromTheStoredProfile(t *testing.T) {
	store := &fakeStore{profiles: []profiles.Profile{storedRDPProfile(t)}}
	s := newTestScreen(t, store)
	profile := store.profiles[0]
	panel := s.newSessionPanel(profile)

	renamed := profile
	renamed.Name = "Renamed desk"
	if err := s.ws.putConnection(renamed); err != nil {
		t.Fatalf("putConnection: %v", err)
	}
	options := profile.RDP
	options.Printers = false
	if err := panel.saveRDPOptions(options); err != nil {
		t.Fatalf("saveRDPOptions: %v", err)
	}

	stored, _ := s.profileByID(profile.ID)
	if stored.Name != "Renamed desk" {
		t.Errorf("name = %q, want the rename to survive the options write", stored.Name)
	}
}

func TestSessionPanel_SaveRDPOptions_RejectedWrite_KeepsTheOldOptions(t *testing.T) {
	store := &fakeStore{profiles: []profiles.Profile{storedRDPProfile(t)}}
	s := newTestScreen(t, store)
	profile := store.profiles[0]
	panel := s.newSessionPanel(profile)
	store.saveErr = errors.New("disk full")

	options := profile.RDP
	options.Audio = false
	if err := panel.saveRDPOptions(options); err == nil {
		t.Fatal("a rejected write was reported as success")
	}
	if !panel.profile.RDP.Audio {
		t.Error("the panel changed although nothing was stored")
	}
}

func TestSessionPanel_RDPToggle_PersistsAndRevertsOnFailure(t *testing.T) {
	store := &fakeStore{profiles: []profiles.Profile{storedRDPProfile(t)}}
	s := newTestScreen(t, store)
	profile := store.profiles[0]
	panel := s.newSessionPanel(profile)
	if !panel.rdpLegacy.Checked || !panel.rdpCert.Checked {
		t.Fatal("the boxes do not show the stored options")
	}

	panel.rdpCert.SetChecked(false)
	stored, _ := s.profileByID(profile.ID)
	if stored.RDP.IgnoreCertificate {
		t.Error("unticking Ignore cert did not reach the store")
	}

	store.saveErr = errors.New("read-only store")
	panel.rdpCert.SetChecked(true)
	if panel.rdpCert.Checked {
		t.Error("the box stayed ticked although the write failed")
	}
	if !strings.Contains(s.status.Text, "Could not save") {
		t.Errorf("status = %q, want a failure notice", s.status.Text)
	}
}

func TestSessionPanel_RDPConnect_UsesTheStoredOptions(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	calls := make(chan rdpclient.Options, 1)
	s.launchRDP = func(_ rdpclient.Target, options rdpclient.Options) (rdpSession, error) {
		calls <- options
		return &fakeRDPSession{done: make(chan struct{})}, nil
	}
	panel := s.newSessionPanel(storedRDPProfile(t))
	panel.connectRDP()

	select {
	case options := <-calls:
		if !options.Clipboard || !options.Printers || !options.Audio {
			t.Errorf("forwarding options = %+v, want the stored ones", options)
		}
		if options.Security != "rdp" || !options.IgnoreCertificate {
			t.Errorf("security = %q ignoreCert = %v, want the stored choices", options.Security, options.IgnoreCertificate)
		}
		if options.Gateway.Host != "gw.example.com" || options.Gateway.Port != 8443 {
			t.Errorf("gateway = %+v, want the stored hop", options.Gateway)
		}
		if options.Password != "" {
			t.Error("a password was taken from the store")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("launcher was not called")
	}
}
