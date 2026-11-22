package ui

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	sshclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/ssh"
)

// shellSession is the interactive session surface the panel needs; tests
// substitute it.
type shellSession interface {
	Output() io.Reader
	Input([]byte) error
	Resize(sshclient.TermSize) error
	Close() error
}

// shellOpener opens an interactive session; tests substitute it.
type shellOpener func(sshclient.Target, sshclient.Options, sshclient.TermSize) (shellSession, sshclient.Result, error)

func defaultShellOpener(target sshclient.Target, options sshclient.Options, size sshclient.TermSize) (shellSession, sshclient.Result, error) {
	shell, result, err := sshclient.OpenShell(target, options, size)
	if err != nil {
		return nil, result, err
	}
	return shell, result, nil
}

// sessionPanel is the content of one session tab. SSH tabs host an interactive
// terminal; other protocols state what is still missing.
type sessionPanel struct {
	screen  *screen
	profile profiles.Profile

	password   *widget.Entry
	connect    *widget.Button
	sftp       *hintButton
	disconnect *hintButton
	ctrlC      *widget.Button
	ctrlD      *widget.Button
	rdpTest    *widget.Button
	rdpLegacy  *widget.Check
	rdpCert    *widget.Check
	rdpOptions *hintButton
	copyScreen *hintButton

	slot  *fyne.Container
	term  *termWidget
	shell shellSession

	rdp     rdpSession
	stream  streamSession
	vnc     vncSession
	vncView *vncView
	// web is the embedded WebKit view of an HTTP session.
	web      webSession
	rdpStart time.Time
	embedded embeddedArea
	note     *widget.Label
	// rdpDetached is set when the user closes the session, so the watcher
	// goroutine never touches the UI (or a shutting-down app) afterwards.
	rdpDetached atomic.Bool

	hidConfig bool

	// pane is this panel's cell in the tab's split layout.
	pane *paneNode
}

func (s *screen) newSessionPanel(profile profiles.Profile) *sessionPanel {
	panel := &sessionPanel{screen: s, profile: profile}
	panel.password = widget.NewEntry()
	panel.password.Password = true
	panel.password.SetPlaceHolder("Password")

	panel.connect = widget.NewButtonWithIcon("Connect", theme.MediaPlayIcon(), panel.connectShell)
	panel.connect.Importance = widget.HighImportance
	// The X closes the connection; the words "Disconnect" spent bar width the
	// terminal needs, and the tooltip says what it does.
	panel.disconnect = newHintButton(theme.CancelIcon(), "End the session and a split pane; asks for the last one", panel.requestDisconnect, s.hint)
	panel.sftp = newHintButton(SFTPIcon(), "Browse and transfer files over SFTP", panel.openSFTP, s.hint)
	panel.disconnect.Importance = widget.LowImportance
	panel.sftp.Importance = widget.LowImportance
	panel.disconnect.Disable()
	panel.ctrlC = widget.NewButton("Ctrl-C", func() { panel.sendControl(0x03) })
	panel.ctrlC.Importance = widget.LowImportance
	panel.ctrlD = widget.NewButton("Ctrl-D", func() { panel.sendControl(0x04) })
	panel.ctrlD.Importance = widget.LowImportance
	panel.rdpTest = widget.NewButtonWithIcon("Test", theme.SearchIcon(), panel.testRDP)
	panel.rdpLegacy = widget.NewCheck("Legacy TLS", nil)
	panel.rdpCert = widget.NewCheck("Ignore cert", nil)
	// The stored options are the single source of truth, so the boxes show the
	// connection's settings and writing one back is what makes it stick. The fields
	// are set directly: SetChecked runs the handler, which would write the store
	// again every time a tab opens.
	panel.rdpLegacy.Checked = profile.RDP.LegacyTLS
	panel.rdpCert.Checked = profile.RDP.IgnoreCertificate
	panel.rdpLegacy.OnChanged = func(checked bool) {
		panel.persistRDPToggle("Legacy TLS", panel.rdpLegacy, checked, func(options *profiles.RDPOptions) {
			options.LegacyTLS = checked
		})
	}
	panel.rdpCert.OnChanged = func(checked bool) {
		panel.persistRDPToggle("Ignore certificate", panel.rdpCert, checked, func(options *profiles.RDPOptions) {
			options.IgnoreCertificate = checked
		})
	}
	panel.rdpOptions = newHintButton(theme.SettingsIcon(), "RDP options: redirection, resolution, audio and credentials", panel.requestRDPOptions, panel.screen.hint)
	panel.copyScreen = newHintButton(theme.ContentCopyIcon(), "Copy the terminal contents to the clipboard", panel.copyTerminalScreen, panel.screen.hint)
	panel.copyScreen.Importance = widget.LowImportance
	panel.rdpOptions.Importance = widget.LowImportance
	panel.slot = container.NewStack(widget.NewLabel("The interactive terminal appears here after Connect."))
	return panel
}

func endpointText(profile profiles.Profile) string {
	text := fmt.Sprintf("%s  %s:%d", profile.Protocol, profile.Host, profile.Port)
	if profile.Folder != "" {
		text += "  in " + profile.Folder
	}
	if profile.Username != "" {
		text += "  as " + profile.Username
	}
	return text
}

func (p *sessionPanel) content() fyne.CanvasObject {
	if p.profile.Protocol == profiles.VNC {
		return container.NewBorder(p.connectionBar(), nil, nil, nil, container.NewPadded(p.slot))
	}
	if p.profile.Protocol == profiles.HTTP {
		note := widget.NewLabel(webViewNote())
		note.Wrapping = fyne.TextWrapWord
		note.Importance = widget.LowImportance
		p.installSlot(container.NewVBox(note))
		return container.NewBorder(p.connectionBar(), nil, nil, nil, container.NewPadded(p.slot))
	}
	if p.profile.Protocol == profiles.RDP {
		p.note = widget.NewLabel("Connect embeds the FreeRDP session in this tab and rescales it with the window. " +
			"Test needs server support for auth-only; enable Ignore cert for changed or self-signed certificates.")
		p.note.Wrapping = fyne.TextWrapWord
		p.note.Importance = widget.WarningImportance
		p.installSlot(container.NewVBox(p.note))
		return container.NewBorder(p.connectionBar(), nil, nil, nil, container.NewPadded(p.slot))
	}
	return container.NewBorder(p.connectionBar(), nil, nil, nil, container.NewPadded(p.slot))
}

// paneButtons are the tmux-style split controls of a terminal pane. Labels would
// push the session bar into horizontal scrolling, so they explain themselves on
// hover instead.
func (p *sessionPanel) paneButtons() []fyne.CanvasObject {
	right := newHintButton(theme.NavigateNextIcon(), "Split the terminal to the right", func() { p.screen.splitPane(p, true) }, p.screen.hint)
	down := newHintButton(theme.MoveDownIcon(), "Split the terminal downwards", func() { p.screen.splitPane(p, false) }, p.screen.hint)
	for _, button := range []*hintButton{right, down} {
		button.Importance = widget.LowImportance
	}
	// Closing a pane moved to View > Close pane: the X on this bar disconnects.
	return []fyne.CanvasObject{right, down}
}

// connectionBar is the only header above the terminal: which connection is
// open, the one-off password and the session controls.
func (p *sessionPanel) connectionBar() fyne.CanvasObject {
	name := widget.NewLabel(p.profile.Name)
	name.TextStyle = fyne.TextStyle{Bold: true}
	endpoint := widget.NewLabel(fmt.Sprintf("%s:%d", p.profile.Host, p.profile.Port))
	endpoint.Importance = widget.LowImportance
	// Name on the left, endpoint on the right, both readable.
	label := container.NewBorder(nil, nil, name, endpoint)

	buttons := []fyne.CanvasObject{p.connect, p.vaultButton()}
	if p.profile.Protocol == profiles.SSH {
		buttons = append(buttons, p.sftp)
	}
	if p.profile.Protocol == profiles.SERIAL {
		buttons = append(buttons, p.serialOptionsButton())
	}
	switch p.profile.Protocol {
	case profiles.RDP:
		buttons = append(buttons, p.rdpTest, p.rdpCert, p.rdpLegacy, p.rdpOptions, p.specialKeysButton())
	default:
		buttons = append(buttons, p.copyScreen, p.ctrlC, p.ctrlD)
		if splittable(p.profile.Protocol) {
			buttons = append(buttons, p.paneButtons()...)
		}
	}
	// The X closes the connection and sits after the split controls, where the
	// hand already is.
	buttons = append(buttons, p.disconnect)
	// Name on its own row so it never collapses to an ellipsis; the password
	// shares the next row with the buttons. Fyne tab content scrolls
	// horizontally as soon as its minimum width exceeds the viewport.
	passwordRow := container.NewBorder(nil, nil, nil, container.NewHBox(buttons...), container.NewPadded(p.password))
	return container.NewVBox(container.NewPadded(label), passwordRow, widget.NewSeparator())
}

// openSFTP shows the transfer window for this panel's connection.
func (p *sessionPanel) openSFTP() {
	if p.profile.Protocol != profiles.SSH {
		p.setProblem("File transfer is available on SSH sessions.")
		return
	}
	options := sshclient.Options{Password: p.passwordFor(), Timeout: 15 * time.Second}
	p.screen.openRemoteFiles(p.target(), options, p.profile.Host)
}

// rdpQuickExit is how soon an exit counts as a failure rather than a session the
// user ended. The detached check already covers an explicit disconnect, so a client
// that dies inside this window never got a connection at all.
const rdpQuickExit = 3 * time.Second

// setEmbeddedVisible shows or hides the embedded RDP window with its tab.
func (p *sessionPanel) setEmbeddedVisible(visible bool) {
	if p.embedded != nil {
		p.embedded.SetVisible(visible)
	}
}

// setEmbeddedSessionsVisible maps or unmaps every embedded session window.
//
// An embedded session is a native X11 child window, so X draws it above everything
// this process paints into the same window: a Fyne popup would open behind the
// remote desktop and stay unreadable until the session was closed. A dialog hides
// the sessions instead, and restoring means asking which one belongs to the selected
// tab rather than mapping them all.
func (s *screen) setEmbeddedSessionsVisible(visible bool) {
	if !visible {
		for _, panel := range s.allPanels() {
			panel.setEmbeddedVisible(false)
		}
		return
	}
	s.refreshEmbeddedVisibility()
}

// refreshEmbeddedVisibility shows the embedded window of the selected tab only.
func (s *screen) refreshEmbeddedVisibility() {
	if s.tabs == nil {
		return
	}
	s.applyEmbeddedVisibility(s.tabs.Selected())
}

// applyEmbeddedVisibility is the one rule both the tab switch and the dialog close
// follow: an embedded session is visible exactly when its tab is the selected one.
func (s *screen) applyEmbeddedVisibility(selected *container.TabItem) {
	for id, panel := range s.panels {
		panel.setEmbeddedVisible(s.openTabs[id] == selected)
	}
}

// restoreConfig brings the Config panel back if this session hid it.
func (p *sessionPanel) restoreConfig() {
	if !p.hidConfig {
		return
	}
	p.hidConfig = false
	if !p.screen.propertiesVisible {
		p.screen.toggleProperties()
	}
}

func (p *sessionPanel) releaseEmbedded() {
	if p.embedded != nil {
		p.embedded.Destroy()
		p.embedded = nil
	}
}

// gatewayMethodLabels are the usage-method choices, in the order the client
// documents them.
var gatewayMethodLabels = []string{"client default", "direct", "detect"}

func (p *sessionPanel) target() sshclient.Target {
	resolved := p.screen.resolve(p.profile)
	return sshclient.Target{Host: resolved.Host, Port: resolved.Port, Username: resolved.Username}
}

func (p *sessionPanel) connectShell() {
	if p.profile.Protocol == profiles.RDP {
		p.connectRDP()
		return
	}
	if p.profile.Protocol == profiles.VNC {
		p.connectVNC()
		return
	}
	if p.profile.Protocol == profiles.HTTP {
		p.connectHTTP()
		return
	}
	if p.profile.Protocol == profiles.SERIAL {
		p.connectSerial()
		return
	}
	if _, ok := streamMode(p.profile.Protocol); ok {
		p.connectStream()
		return
	}
	if strings.TrimSpace(p.profile.Username) == "" {
		p.setProblem("Add a username in the Config panel and save the connection before connecting.")
		return
	}
	p.setConnecting("Connecting to " + p.profile.Host + " ...")
	target := p.target()
	options := sshclient.Options{Password: p.passwordFor(), Timeout: 15 * time.Second}
	go func() {
		shell, result, err := p.screen.openShell(target, options, sshclient.TermSize{Cols: 80, Rows: 24})
		runOnUI(func() { p.applyConnect(target, options, shell, result, err) })
	}()
}

func (p *sessionPanel) applyConnect(target sshclient.Target, options sshclient.Options, shell shellSession, result sshclient.Result, err error) {
	if err != nil {
		var unknown *sshclient.UnknownHostKeyError
		if errors.As(err, &unknown) {
			p.setProblem(fmt.Sprintf("Host key for %s is not in known_hosts (fingerprint %s).", target.Host, unknown.Fingerprint))
			p.screen.confirm("Unverified SSH host key",
				fmt.Sprintf("The host key for %s is not in known_hosts.\n\nFingerprint: %s\n\nConnect and trust it for this session?",
					target.Host, unknown.Fingerprint),
				func() { p.retryTrusted(options) }, func() { p.setConnected(false) })
			return
		}
		p.setConnected(false)
		p.setProblem("SSH connection failed: " + err.Error())
		return
	}
	p.startSession(shell, result)
}

func (p *sessionPanel) retryTrusted(options sshclient.Options) {
	target := p.target()
	options.AcceptAnyHostKey = true
	shell, result, err := p.screen.openShell(target, options, sshclient.TermSize{Cols: 80, Rows: 24})
	p.applyConnect(target, options, shell, result, err)
}

func (p *sessionPanel) startSession(shell shellSession, result sshclient.Result) {
	p.shell = shell
	p.term = newTermWidget(80, 24, func(data []byte) { _ = shell.Input(data) }, func(cols, rows int) {
		_ = shell.Resize(sshclient.TermSize{Cols: cols, Rows: rows})
	})
	p.wireTerminal(p.term)
	p.installTerminal()
	// Take the keyboard on the first layout pass; the tab selection also
	// focuses the terminal once the tab is painted.
	p.term.autoFocus = true
	p.setConnected(true)

	verification := "unverified host key"
	if result.HostKeyVerified {
		verification = "host key verified"
	}
	p.setStatus(fmt.Sprintf("Connected: %s (%s). Click the terminal and type to use it.",
		strings.TrimSpace(result.ServerVersion), verification))
	go p.readOutput(shell)
}

// maximiseTerminal hides the Config panel while a session is active so the
// terminal gets the window height. Disconnecting restores it.
func (p *sessionPanel) maximiseTerminal() {
	if p.screen.propertiesVisible {
		p.hidConfig = true
		p.screen.toggleProperties()
	}
}

func (p *sessionPanel) readOutput(shell shellSession) {
	pumpBytes(shell.Output(), p.applyOutput, p.sessionClosed)
}

func (p *sessionPanel) applyOutput(data []byte) {
	if p.term != nil {
		p.term.write(data)
	}
}

// clearTerminal replaces the terminal with a placeholder so a finished session
// leaves no scrollback on screen.
func (p *sessionPanel) clearTerminal() {
	p.term = nil
	placeholder := widget.NewLabel("Session closed. Choose Connect to open a new terminal.")
	placeholder.Importance = widget.LowImportance
	p.installSlot(placeholder)
}

func (p *sessionPanel) sessionClosed() {
	p.shell = nil
	if p.term != nil {
		p.clearTerminal()
	}
	// A finished session must leave the panel ready to connect again, including
	// the password field that startSession disabled.
	p.setConnected(false)
	p.setStatus("Session closed.")
}

// shutdown ends any live session without touching the UI, used on application
// exit so no child process or embedded window survives.
func (p *sessionPanel) shutdown() {
	p.closeWebView()
	if p.shell != nil {
		_ = p.shell.Close()
		p.shell = nil
	}
	if p.rdp != nil {
		p.rdpDetached.Store(true)
		_ = p.rdp.Close()
		p.rdp = nil
	}
	if p.vnc != nil {
		_ = p.vnc.Close()
		p.vnc = nil
	}
	if p.stream != nil {
		_ = p.stream.Close()
		p.stream = nil
	}
	p.releaseEmbedded()
}

// hasSession reports whether the panel still owns an open transport. It is the
// state the disconnect control follows, so it needs no flag of its own.
func (p *sessionPanel) hasSession() bool {
	return p.shell != nil || p.stream != nil || p.vnc != nil || p.rdp != nil
}

// requestDisconnect is what the X on the session bar runs. It ends the session of
// this pane and, when the tab is split, takes the pane with it: a sub-panel left
// behind shows a closed session and has nothing left to do. The last pane of a
// tab keeps its place, because removing it is closing the tab.
//
// A tab is one group of sessions, and splitting a terminal adds a session to that
// group, so asking on every disconnect would be noise while the group is still on
// screen. The question is kept for the last session of the group.
func (p *sessionPanel) requestDisconnect() {
	end := p.closeSelf
	if p.screen.liveSessionsInTab(p) > 0 {
		end()
		return
	}
	p.screen.confirm("Disconnect session?",
		fmt.Sprintf("%s is the last session in this tab. Disconnecting ends it.", p.profile.Name),
		end, nil)
}

// closeSelf ends this pane's session and gives a split pane's space back to its
// sibling. The last pane of a tab stays: it is where Connect opens the next
// session, and closing the tab is what removes it.
func (p *sessionPanel) closeSelf() {
	if p.pane != nil && p.pane.parent != nil {
		p.screen.closePane(p)
		return
	}
	p.disconnectShell()
}

func (p *sessionPanel) disconnectShell() {
	if p.web != nil {
		p.closeWebView()
		p.releaseEmbedded()
		p.setConnected(false)
		p.setStatus("Web view closed.")
		return
	}
	if p.stream != nil {
		session := p.stream
		p.stream = nil
		_ = session.Close()
		p.clearTerminal()
		p.setConnected(false)
		p.setStatus("Session closed.")
		return
	}
	if p.vnc != nil {
		session := p.vnc
		p.vnc = nil
		p.vncView = nil
		_ = session.Close()
		p.setConnected(false)
		p.setStatus("VNC session closed.")
		return
	}
	if p.rdp != nil {
		p.rdpDetached.Store(true)
		_ = p.rdp.Close()
		p.rdp = nil
		p.releaseEmbedded()
		p.setConnected(false)
		p.setStatus("RDP session closed.")
		return
	}
	if p.shell != nil {
		_ = p.shell.Close()
	}
	p.clearTerminal()
	p.sessionClosed()
}

// installTerminal keeps the terminal under one theme override so zoom affects
// only the console, never the rest of the window.
//
// The override is created once and its theme is mutated afterwards: Fyne keeps
// theme overrides in a per-object cache, and ThemeOverride.Refresh re-registers
// its own theme. Creating a second wrapper left the first one alive, so any
// refresh of that orphan re-registered the previous text size and the zoom
// jumped back on the next output or scroll.
func (p *sessionPanel) installTerminal() {
	if p.term == nil {
		return
	}
	p.installSlot(p.term)
}

// wireTerminal connects clipboard and interrupt actions to a terminal view.
func (p *sessionPanel) wireTerminal(term *termWidget) {
	term.paused = !p.screen.panelShown(p)
	term.onCopy = func(text string) {
		clipboard := p.screen.clipboard()
		if clipboard == nil || text == "" {
			return
		}
		clipboard.SetContent(text)
		p.setStatus("Selection copied to the clipboard.")
	}
	term.onInterrupt = func() { term.sendBytes([]byte{0x03}) }
}

// copyTerminalScreen copies the terminal contents to the clipboard.
func (p *sessionPanel) copyTerminalScreen() {
	if p.term == nil {
		p.setStatus("No terminal is open in this tab.")
		return
	}
	text := p.term.screenText()
	clipboard := p.screen.clipboard()
	if clipboard == nil {
		p.setProblem("This system has no clipboard available.")
		return
	}
	clipboard.SetContent(text)
	if _, _, hasSelection := p.term.selectionRange(); hasSelection {
		p.setStatus("Selected text copied to the clipboard.")
		return
	}
	p.setStatus("Whole buffer copied (scrollback plus screen). Select text to copy only part of it.")
}

func (p *sessionPanel) sendControl(b byte) {
	if p.term != nil {
		p.term.sendBytes([]byte{b})
	}
}

// setStatus reports in the application status bar; the session tab keeps a
// single connection bar so the terminal owns the space.
func (p *sessionPanel) setStatus(text string) {
	p.screen.setStatus(text)
}

func (p *sessionPanel) setProblem(text string) {
	p.screen.setProblem(text)
}
