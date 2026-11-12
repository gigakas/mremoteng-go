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
	p.delete = widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), s.requestDelete)

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

// welcomeHelpRow is one line of the button reference on the start page. The icon
// is the one the button actually shows, so the page and the bar match.
type welcomeHelpRow struct {
	// icon is resolved when the page is built, not at package init: the theme
	// accessors need a running Fyne app, and a package-level call logs an error
	// for every accessor before the app exists.
	icon func() fyne.Resource
	name string
	what string
}

// welcomeHelp documents the buttons of the three bars. Hovering a button repeats
// its line in the status bar; the start page is where the whole set can be read
// at once, with the icon beside it, which is the only way to learn the icons you
// have not hovered yet.
var welcomeHelp = []welcomeHelpRow{
	{theme.MenuIcon, "Connection tree", "Show or hide the tree."},
	{theme.SettingsIcon, "Config panel", "Show or hide the connection form."},
	{theme.InfoIcon, "Activity log", "Show or hide the log."},
	{theme.ContentAddIcon, "Quick connect", "Connect once without saving."},
	{theme.MediaPlayIcon, "Connect", "Open the selected session."},
	{theme.MediaStopIcon, "Disconnect", "End the session, keep the profile."},
	{theme.SearchIcon, "Test", "Check RDP credentials only."},
	{theme.CheckButtonCheckedIcon, "Ignore cert", "Accept a self-signed certificate."},
	{theme.CheckButtonCheckedIcon, "Legacy TLS", "Use legacy RDP security."},
	{theme.SettingsIcon, "RDP options", "Redirection, resolution, audio."},
	{theme.ContentCopyIcon, "Copy terminal", "Copy the terminal to the clipboard."},
	{nil, "Ctrl-C / Ctrl-D", "Send control bytes as buttons."},
	{theme.NavigateNextIcon, "Split right", "Second pane beside this one."},
	{theme.MoveDownIcon, "Split down", "Second pane below this one."},
	{theme.CancelIcon, "Close pane", "Close this pane."},
	{theme.DocumentSaveIcon, "Save password", "Store it in the encrypted vault."},
	{theme.DeleteIcon, "Delete", "Delete the selected connection."},
}

// welcomeIconSize is the box the reference icons are drawn in. Fyne's default
// icon size is too heavy for a list, where the icon is a reminder and not a
// control.
const welcomeIconSize = 16

// welcomeHelpRowContent renders one reference line: the icon in a fixed box, then
// the name in bold followed by its explanation on the same line. Stacking name
// and explanation doubled the height and pushed the page into scrolling.
func welcomeHelpRowContent(row welcomeHelpRow) fyne.CanvasObject {
	box := fyne.NewSize(welcomeIconSize, welcomeIconSize)
	var leading fyne.CanvasObject = container.NewGridWrap(box)
	if row.icon != nil {
		leading = container.NewGridWrap(box, widget.NewIcon(row.icon()))
	}
	text := widget.NewRichText(
		&widget.TextSegment{Text: row.name + "  ", Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}}},
		&widget.TextSegment{Text: row.what},
	)
	text.Wrapping = fyne.TextWrapWord
	return container.NewBorder(nil, nil, container.NewCenter(leading), nil, text)
}

// welcomeSessionNotice states what works today. It used to claim that sessions
// were unimplemented, which stopped being true several stages ago.
var welcomeSessionNotice = "SSH, RDP, VNC, Telnet, RAW, rlogin and HTTP work today; RDP needs xfreerdp3 on the PATH."

// welcomeContent is the start page shown when no session tab is open.
func welcomeContent() fyne.CanvasObject {
	heading := widget.NewLabelWithStyle("Welcome to mRemoteNG Go", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	intro := widget.NewLabel("A mRemoteNG-compatible connection manager for Linux.")
	intro.Alignment = fyne.TextAlignCenter
	intro.Wrapping = fyne.TextWrapWord
	steps := widget.NewLabel("1. File > New connection (or the New button).\n" +
		"2. Fill name, optional folder, protocol, host and port.\n" +
		"3. Save, then select it in the tree to open a session.")
	steps.Alignment = fyne.TextAlignLeading
	steps.Wrapping = fyne.TextWrapWord
	notice := widget.NewLabel(welcomeSessionNotice)
	notice.Wrapping = fyne.TextWrapWord
	notice.Importance = widget.LowImportance

	rows := make([]fyne.CanvasObject, 0, len(welcomeHelp))
	for _, row := range welcomeHelp {
		rows = append(rows, welcomeHelpRowContent(row))
	}
	// Two columns halve the height, which is what kept the page scrolling.
	buttons := widget.NewCard("Buttons", "Hover any icon to see the same line in the status bar.",
		container.NewGridWithColumns(2, rows...))

	cards := welcomeCards(steps, notice, buttons)
	return container.NewBorder(
		container.NewVBox(container.NewPadded(heading), intro, widget.NewSeparator()),
		nil, nil, nil,
		container.NewVScroll(cards),
	)
}

// welcomeCards is the scrollable body of the start page, split out so a test can
// assert it fits the default window without a scrollbar.
func welcomeCards(steps, notice fyne.CanvasObject, buttons fyne.CanvasObject) fyne.CanvasObject {
	return container.NewVBox(
		container.NewPadded(widget.NewCard("Getting started", "", container.NewVBox(steps, notice))),
		container.NewPadded(buttons),
	)
}
