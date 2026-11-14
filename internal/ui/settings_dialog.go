package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

// protocolNames is the protocol picker's list, in the order the Config panel shows
// it, so the two cannot disagree.
var protocolNames = []string{
	string(profiles.SSH), string(profiles.RDP), string(profiles.VNC),
	string(profiles.TELNET), string(profiles.RAW), string(profiles.RLOGIN), string(profiles.HTTP),
}

// settingsValues is what the dialog collects. It is separate from the widgets so the
// validation and the write can be tested without a window.
type settingsValues struct {
	Theme    string
	Protocol string
	// Port is the port a new connection of that protocol starts with; zero asks for
	// the protocol's own default.
	Port       int
	ReopenTabs bool
}

// applySettings validates the values and stores them. Nothing is written when a
// value is refused, so a rejected dialog leaves the previous settings in place.
func applySettings(prefs appPreferences, values settingsValues) error {
	if values.Protocol != "" && !isProtocolName(values.Protocol) {
		return fmt.Errorf("rdp: %q is not a protocol this build connects", values.Protocol)
	}
	if values.Port != 0 && (values.Port < 1 || values.Port > 65535) {
		return fmt.Errorf("the default port %d is outside 1-65535", values.Port)
	}
	prefs.setThemeVariant(values.Theme)
	prefs.setNewConnectionDefault(values.Protocol, values.Port)
	prefs.setReopenTabs(values.ReopenTabs)
	return nil
}

// isProtocolName reports whether name is one of the protocols the window offers.
func isProtocolName(name string) bool {
	for _, known := range protocolNames {
		if known == name {
			return true
		}
	}
	return false
}

// parseSettingsPort reads the port field: an empty field means the protocol's own
// default.
func parseSettingsPort(text string) (int, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0, nil
	}
	port, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("the default port %q is not a number", trimmed)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("the default port %d is outside 1-65535", port)
	}
	return port, nil
}

// preferredProtocol is the protocol a new connection starts with.
func (s *screen) preferredProtocol() profiles.Protocol {
	choice := s.prefs.defaultProtocol()
	if isProtocolName(choice) {
		return profiles.Protocol(choice)
	}
	return profiles.SSH
}

// newConnectionProfile is the empty profile the Config panel starts a new connection
// with, honouring the defaults chosen in Settings.
func (s *screen) newConnectionProfile() profiles.Profile {
	protocol := s.preferredProtocol()
	port := profiles.DefaultPort(protocol)
	if stored := s.prefs.defaultPort(); stored > 0 && stored <= 65535 {
		port = stored
	}
	return profiles.Profile{Protocol: protocol, Port: port}
}

// requestSettings edits the application preferences: the colour variant, the default
// for a new connection, and whether the last tabs come back at startup.
func (s *screen) requestSettings() {
	themes := widget.NewSelect([]string{themeVariantSystem, themeVariantLight, themeVariantDark}, nil)
	themes.SetSelected(s.prefs.themeVariant())
	protocols := widget.NewSelect(protocolNames, nil)
	protocols.SetSelected(string(s.preferredProtocol()))
	port := widget.NewEntry()
	port.SetPlaceHolder("protocol default")
	if stored := s.prefs.defaultPort(); stored > 0 {
		port.SetText(strconv.Itoa(stored))
	}
	reopen := widget.NewCheck("Reopen the tabs of the last session at startup", nil)
	reopen.SetChecked(s.prefs.reopenTabs())
	problem := widget.NewLabel("")
	problem.Importance = widget.DangerImportance
	problem.Wrapping = fyne.TextWrapWord

	var popup *modal
	content := container.NewVBox(
		widget.NewLabelWithStyle("Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Colours", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		themes,
		widget.NewLabel("system follows the desktop; light and dark pin the window."),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("New connections", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		protocols, port,
		widget.NewLabel("Empty port: the protocol's own default."),
		widget.NewSeparator(),
		reopen, problem,
		container.NewHBox(
			widget.NewButton("Apply", func() {
				chosenPort, err := parseSettingsPort(port.Text)
				if err != nil {
					problem.SetText(err.Error())
					return
				}
				values := settingsValues{
					Theme:      themes.Selected,
					Protocol:   protocols.Selected,
					Port:       chosenPort,
					ReopenTabs: reopen.Checked,
				}
				if err := applySettings(s.prefs, values); err != nil {
					problem.SetText(err.Error())
					return
				}
				applyThemeVariant(s.app, values.Theme)
				popup.Hide()
				s.setStatus("Settings saved: " + values.Theme + " colours, new connections start with " +
					values.Protocol + ".")
			}),
			widget.NewButton("Cancel", func() { popup.Hide() }),
		),
	)
	popup = s.showModal(content, fyne.NewSize(480, 460))
}

// applyThemeVariant repaints the window with the chosen colours. Fyne takes the
// light/dark variant from the desktop and has no setting for it, so the theme pins
// the variant instead.
func applyThemeVariant(app fyne.App, variant string) {
	if app == nil {
		return
	}
	app.Settings().SetTheme(SoberThemeVariant(variant))
}

// SoberThemeForPreferences is the theme the stored colour choice asks for, so the
// first paint already matches the setting. cmd/mremoteng calls it before the window
// exists.
func SoberThemeForPreferences(prefs fyne.Preferences) fyne.Theme {
	return SoberThemeVariant(appPreferences{prefs: prefs}.themeVariant())
}
