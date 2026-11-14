package ui

import (
	"errors"
	"strings"

	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	rdpclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/rdp"
)

// rdpOptionsFromProfile turns the stored per-connection RDP settings into the
// options a launch needs. Credentials, the parent window and the requested size
// stay with the session: they are never stored.
func rdpOptionsFromProfile(profile profiles.Profile) rdpclient.Options {
	stored := profile.RDP
	return rdpclient.Options{
		Clipboard:         stored.Clipboard,
		Printers:          stored.Printers,
		Audio:             stored.Audio,
		Drives:            append([]string(nil), stored.Drives...),
		DynamicResolution: stored.DynamicResolution,
		MultiMonitor:      stored.MultiMonitor,
		Security:          legacySecurity(stored.LegacyTLS),
		IgnoreCertificate: stored.IgnoreCertificate,
		Gateway: rdpclient.Gateway{
			Host:        stored.Gateway.Host,
			Port:        stored.Gateway.Port,
			UsageMethod: stored.Gateway.UsageMethod,
		},
	}
}

// rdpOptionsToProfile is the other direction, used when the options dialog saves.
// It keeps the fields the dialog does not edit.
func rdpOptionsToProfile(options rdpclient.Options) profiles.RDPOptions {
	return profiles.RDPOptions{
		Clipboard:         options.Clipboard,
		Printers:          options.Printers,
		Audio:             options.Audio,
		Drives:            trimmedDrives(options.Drives),
		DynamicResolution: options.DynamicResolution,
		MultiMonitor:      options.MultiMonitor,
		LegacyTLS:         options.Security == "rdp",
		IgnoreCertificate: options.IgnoreCertificate,
		Gateway: profiles.Gateway{
			Host:        strings.TrimSpace(options.Gateway.Host),
			Port:        options.Gateway.Port,
			UsageMethod: options.Gateway.UsageMethod,
		},
	}
}

// legacySecurity maps the stored toggle to FreeRDP's pre-TLS security mode.
func legacySecurity(legacy bool) string {
	if legacy {
		return "rdp"
	}
	return ""
}

// trimmedDrives drops the empties a comma-separated entry leaves behind, so a
// trailing comma is not stored as a drive.
func trimmedDrives(drives []string) []string {
	kept := make([]string, 0, len(drives))
	for _, drive := range drives {
		if trimmed := strings.TrimSpace(drive); trimmed != "" {
			kept = append(kept, trimmed)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}

// saveRDPOptions writes the options into the profile that owns them and keeps the
// panel's copy in step, so the next Connect uses what was just chosen. Nothing is
// changed when the store rejects the write: the dialog reports it and stays open.
func (p *sessionPanel) saveRDPOptions(options profiles.RDPOptions) error {
	if !p.screen.requireWritable("Saving RDP options") {
		return errors.New("the profile store is read-only")
	}
	// Start from the stored profile, not the panel's copy: the Config panel may have
	// renamed or re-addressed the connection since this tab opened, and writing the
	// old copy back would silently undo that edit.
	updated, known := p.screen.profileByID(p.profile.ID)
	if !known {
		updated = p.profile
	}
	updated.RDP = options
	if err := updated.Validate(); err != nil {
		return err
	}
	if err := p.screen.ws.putConnection(updated); err != nil {
		return err
	}
	p.profile = updated
	return nil
}

// persistRDPToggle saves one checkbox of the session bar. A rejected write puts the
// box back where it was: the stored options are the truth, and a tick that does not
// survive the next Connect is worse than no tick at all.
func (p *sessionPanel) persistRDPToggle(label string, box *widget.Check, checked bool, change func(*profiles.RDPOptions)) {
	options := p.profile.RDP
	change(&options)
	if err := p.saveRDPOptions(options); err != nil {
		box.Checked = !checked
		box.Refresh()
		p.setProblem("Could not save " + label + ": " + err.Error())
		return
	}
	p.setStatus("Saved " + label + " for " + p.profile.Name + ".")
}
