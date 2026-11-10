package ui

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

// requestQuickConnect opens an ad-hoc session without saving a profile.
func (s *screen) requestQuickConnect() {
	host := widget.NewEntry()
	host.SetPlaceHolder("host or address")
	user := widget.NewEntry()
	user.SetPlaceHolder("username (optional)")
	port := widget.NewEntry()
	port.SetPlaceHolder("port (optional)")
	protocol := widget.NewSelect([]string{string(profiles.SSH), string(profiles.RDP), string(profiles.VNC)}, func(choice string) {
		port.SetPlaceHolder(strconv.Itoa(profiles.DefaultPort(profiles.Protocol(choice))))
	})
	protocol.SetSelected(string(profiles.SSH))
	port.SetPlaceHolder(strconv.Itoa(profiles.DefaultPort(profiles.SSH)))

	var popup *widget.PopUp
	content := container.NewVBox(
		widget.NewLabelWithStyle("Quick connect", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		host, protocol, port, user,
		container.NewHBox(
			widget.NewButton("Connect", func() {
				popup.Hide()
				s.quickConnect(host.Text, protocol.Selected, port.Text, user.Text)
			}),
			widget.NewButton("Cancel", func() { popup.Hide() }),
		),
	)
	popup = widget.NewModalPopUp(container.NewGridWrap(fyne.NewSize(textDialogWidth, 320), container.NewPadded(content)), s.win.Canvas())
	popup.Show()
}

// quickConnect validates the ad-hoc endpoint and opens a temporary session tab.
func (s *screen) quickConnect(host, protocol, portText, username string) {
	name := strings.TrimSpace(host)
	if name == "" {
		s.setProblem("Quick connect needs a host.")
		return
	}
	chosen := profiles.Protocol(strings.TrimSpace(protocol))
	port := profiles.DefaultPort(chosen)
	if trimmed := strings.TrimSpace(portText); trimmed != "" {
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			s.setProblem("Quick connect port must be a number.")
			return
		}
		port = parsed
	}
	id, err := profiles.NewID()
	if err != nil {
		s.setProblem("Could not start the session: " + err.Error())
		return
	}
	profile := profiles.Profile{
		ID:       id,
		Name:     "Quick: " + name,
		Protocol: chosen,
		Host:     name,
		Port:     port,
		Username: strings.TrimSpace(username),
	}
	if err := profile.Validate(); err != nil {
		s.setProblem("Quick connect: " + err.Error())
		return
	}
	s.openEphemeral(profile)
	s.setStatus("Quick session for " + name + " opened. It is not saved; use New connection to keep it.")
}

// openEphemeral opens a tab for a profile that is not in the store or the tree.
func (s *screen) openEphemeral(profile profiles.Profile) {
	panel := s.newSessionPanel(profile)
	tab := container.NewTabItem(profile.Name, panel.content())
	s.openTabs[profile.ID] = tab
	s.panels[profile.ID] = panel
	s.tabs.Append(tab)
	s.tabs.Select(tab)
}
