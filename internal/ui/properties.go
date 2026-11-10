package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

// propertiesPanel is the bottom "Config" area of the mRemoteNG layout. It
// edits the selected connection; folders are structural and read-only here.
type propertiesPanel struct {
	box    *fyne.Container
	title  *widget.Label
	form   fyne.CanvasObject
	name   *widget.Entry
	folder *widget.Entry
	proto  *widget.Select
	host   *widget.Entry
	port   *widget.Entry
	user   *widget.Entry

	inheritUser *widget.Check
	inheritPort *widget.Check

	save   *widget.Button
	revert *widget.Button
	delete *widget.Button
}

func newPropertiesPanel(s *screen) *propertiesPanel {
	p := &propertiesPanel{}

	p.title = widget.NewLabelWithStyle("Connection details", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	p.name = widget.NewEntry()
	p.name.SetPlaceHolder("Production web server")
	p.folder = widget.NewEntry()
	p.folder.SetPlaceHolder("Production/Web")
	p.proto = widget.NewSelect([]string{
		string(profiles.SSH), string(profiles.RDP), string(profiles.VNC),
		string(profiles.TELNET), string(profiles.RAW), string(profiles.RLOGIN), string(profiles.HTTP),
	}, s.protocolChanged)
	p.host = widget.NewEntry()
	p.host.SetPlaceHolder("server.example.com")
	p.port = widget.NewEntry()
	p.port.SetPlaceHolder("22")
	p.user = widget.NewEntry()
	p.user.SetPlaceHolder("deploy")
	p.inheritUser = widget.NewCheck("From folder", nil)
	p.inheritPort = widget.NewCheck("From folder", nil)
	p.inheritUser.OnChanged = func(bool) { s.refreshInheritance() }
	p.inheritPort.OnChanged = func(bool) { s.refreshInheritance() }

	p.save = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), s.save)
	p.revert = widget.NewButtonWithIcon("Revert", theme.CancelIcon(), s.revert)
	p.delete = widget.NewButtonWithIcon("", theme.DeleteIcon(), s.requestDelete)

	// Delete lives next to the connection name so the toolbar stays free of
	// write actions, matching how mRemoteNG ties object actions to the object.
	nameRow := container.NewBorder(nil, nil, nil, p.delete, p.name)

	// One form, one field per row: a single form layout aligns every label and
	// field, while two side-by-side forms computed their columns independently
	// and looked ragged.
	p.form = widget.NewForm(
		widget.NewFormItem("Name", nameRow),
		widget.NewFormItem("Folder", p.folder),
		widget.NewFormItem("Protocol", p.proto),
		widget.NewFormItem("Host", p.host),
		widget.NewFormItem("Port", p.port),
		widget.NewFormItem("Username", container.NewBorder(nil, nil, nil, p.inheritUser, p.user)),
		widget.NewFormItem("Port inherited", p.inheritPort),
	)
	buttons := container.NewHBox(p.save, p.revert)
	p.box = container.NewBorder(
		container.NewVBox(p.title, widget.NewSeparator()),
		nil, nil, nil,
		container.NewBorder(nil, container.NewPadded(buttons), nil, nil, container.NewPadded(p.form)),
	)
	return p
}

func (p *propertiesPanel) populate(profile profiles.Profile) {
	p.inheritUser.SetChecked(profile.InheritUsername)
	p.inheritPort.SetChecked(profile.InheritPort)
	p.name.SetText(profile.Name)
	p.folder.SetText(profile.Folder)
	p.proto.SetSelected(string(profile.Protocol))
	p.host.SetText(profile.Host)
	p.port.SetText(strconv.Itoa(profile.Port))
	p.user.SetText(profile.Username)
}

func (p *propertiesPanel) rawForm(id string, protocol profiles.Protocol) profiles.Profile {
	port, err := strconv.Atoi(strings.TrimSpace(p.port.Text))
	if err != nil {
		port = 0
	}
	return profiles.Profile{
		ID:              id,
		Name:            strings.TrimSpace(p.name.Text),
		Folder:          strings.TrimSpace(p.folder.Text),
		Protocol:        protocol,
		Host:            strings.TrimSpace(p.host.Text),
		Port:            port,
		Username:        strings.TrimSpace(p.user.Text),
		InheritUsername: p.inheritUser.Checked,
		InheritPort:     p.inheritPort.Checked,
	}
}

func (p *propertiesPanel) formProfile(id string, protocol profiles.Protocol) (profiles.Profile, error) {
	if _, err := strconv.Atoi(strings.TrimSpace(p.port.Text)); err != nil {
		return profiles.Profile{}, fmt.Errorf("Port must be a whole number between 1 and 65535.")
	}
	profile := p.rawForm(id, protocol)
	if err := profile.Validate(); err != nil {
		return profiles.Profile{}, err
	}
	return profile, nil
}

func (p *propertiesPanel) setEnabled(enabled bool) {
	for _, button := range []*widget.Button{p.save, p.delete} {
		if enabled {
			button.Enable()
			continue
		}
		button.Disable()
	}
}

// welcomeContent is the start page shown when no session tab is open.
func welcomeContent() fyne.CanvasObject {
	heading := widget.NewLabelWithStyle("Welcome to mRemoteNG Go", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	intro := widget.NewLabel("A Go implementation of the mRemoteNG connection manager, built for Linux first.")
	intro.Alignment = fyne.TextAlignCenter
	steps := widget.NewLabel("1. Choose File > New connection (or the New button).\n" +
		"2. Fill in name, optional folder, protocol, host and port in the Config panel.\n" +
		"3. Choose Save: the connection appears in the tree and is stored locally.\n" +
		"4. Select it in the tree to open its session tab.")
	steps.Alignment = fyne.TextAlignLeading
	notice := widget.NewLabel("Session support is not implemented yet: this preview manages connection profiles only. " +
		"SSH and RDP sessions are the next delivery stage.")
	notice.Wrapping = fyne.TextWrapWord
	notice.Importance = widget.WarningImportance

	card := widget.NewCard("Getting started", "", container.NewVBox(steps, notice))
	return container.NewBorder(
		container.NewVBox(container.NewPadded(heading), intro, widget.NewSeparator()),
		nil, nil, nil,
		container.NewPadded(card),
	)
}
