package ui

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	rdpclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/rdp"
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

// rdpSession is a running FreeRDP process; tests substitute it.
type rdpSession interface {
	Wait() error
	Close() error
	// Output is the client's own log, which says why a session that died did so.
	Output() string
}

// rdpLauncher starts an RDP session; tests substitute it.
type rdpLauncher func(rdpclient.Target, rdpclient.Options) (rdpSession, error)

func defaultRDPLauncher(target rdpclient.Target, options rdpclient.Options) (rdpSession, error) {
	session, err := rdpclient.Launch(target, options)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// rdpProber verifies RDP credentials without opening a desktop.
type rdpProber func(rdpclient.Target, rdpclient.Options) (string, error)

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

	rdp      rdpSession
	stream   streamSession
	vnc      vncSession
	vncView  *vncView
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
	panel.sftp = newHintButton(theme.FolderIcon(), "Browse and transfer files over SFTP", panel.openSFTP, s.hint)
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
		note := widget.NewLabel("Connect opens the address in your default browser. An embedded webview is not implemented yet.")
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
	switch p.profile.Protocol {
	case profiles.RDP:
		buttons = append(buttons, p.rdpTest, p.rdpCert, p.rdpLegacy, p.rdpOptions)
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

func (p *sessionPanel) rdpTarget() rdpclient.Target {
	resolved := p.screen.resolve(p.profile)
	return rdpclient.Target{Host: resolved.Host, Port: resolved.Port, Username: resolved.Username}
}

func (p *sessionPanel) connectRDP() {
	options := rdpOptionsFromProfile(p.profile)
	p.setConnecting("Starting FreeRDP for " + p.profile.Host + gatewaySummary(options.Gateway) + " ...")
	target := p.rdpTarget()
	options.Password = p.passwordFor()
	area, embedErr := p.embedSession()
	if area != nil {
		options.ParentWindowID = area.ID()
	}
	go func() {
		session, err := p.screen.launchRDP(target, options)
		fyne.Do(func() { p.applyConnectRDPWithArea(session, area, embedErr, err) })
	}()
}

// embedSession creates the embedded area when the platform supports it.
func (p *sessionPanel) embedSession() (embeddedArea, error) {
	if p.screen.embedder == nil {
		return nil, ErrNoX11
	}
	return p.screen.embedder(p.screen.win, p.slot, p.screen.tabs)
}

func (p *sessionPanel) applyConnectRDP(session rdpSession, err error) {
	p.applyConnectRDPWithArea(session, nil, ErrNoX11, err)
}

func (p *sessionPanel) applyConnectRDPWithArea(session rdpSession, area embeddedArea, embedErr, err error) {
	if err != nil {
		if area != nil {
			area.Destroy()
		}
		p.setConnected(false)
		p.setProblem("RDP launch failed: " + err.Error())
		if embedErr != nil {
			p.setProblem("RDP launch failed: " + err.Error() + " (embedding unavailable: " + embedErr.Error() + ")")
		}
		return
	}
	p.rdp = session
	p.rdpStart = time.Now()
	p.rdpDetached.Store(false)
	p.embedded = area
	p.setConnected(true)
	if area != nil {
		p.setStatus("RDP session for " + p.profile.Host + " embedded in the tab; it rescales with the window.")
	} else {
		p.setStatus("RDP session for " + p.profile.Host + " opened in a separate window. Closing that window ends the session.")
	}
	go func() {
		_ = session.Wait()
		// The user may have disconnected already; in that case the panel state
		// is clean and touching the UI (or the app, which may be closing) is not
		// needed.
		if p.rdpDetached.Load() {
			return
		}
		fyne.Do(func() {
			if p.rdpDetached.Load() {
				return
			}
			p.rdp = nil
			p.releaseEmbedded()
			p.setConnected(false)
			cause := rdpclient.Summarize(session.Output())
			if time.Since(p.rdpStart) < rdpQuickExit {
				p.reportRDPExit(cause)
				return
			}
			p.setStatus("RDP session closed: " + cause)
		})
	}()
}

// rdpQuickExit is how soon an exit counts as a failure rather than a session the
// user ended. The detached check already covers an explicit disconnect, so a client
// that dies inside this window never got a connection at all.
const rdpQuickExit = 3 * time.Second

// reportRDPExit explains why FreeRDP ended without opening a session. The cause is
// the client's own line, reduced to one actionable sentence: "ended immediately"
// on its own told the user nothing and the session bar could not be read behind the
// window that had already closed.
func (p *sessionPanel) reportRDPExit(cause string) {
	message := "FreeRDP ended before opening a session: " + cause
	p.setProblem(message)
	if p.screen.notice == nil {
		return
	}
	p.screen.notice("RDP session failed",
		p.profile.Name+" ("+p.profile.Host+":"+strconv.Itoa(p.profile.Port)+")\n\n"+message+"\n\n"+rdpFailureHint(cause))
}

// rdpFailureHint turns the client's cause into the next thing to try.
func rdpFailureHint(cause string) string {
	lower := strings.ToLower(cause)
	switch {
	case strings.Contains(lower, "gateway"):
		return "The RD Gateway refused the hop: check its host, its port and the usage method."
	case strings.Contains(lower, "cert") || strings.Contains(lower, "tofu"):
		return "The server certificate was refused: enable Ignore cert on the session bar if this host is trusted."
	case strings.Contains(lower, "auth") || strings.Contains(lower, "password") || strings.Contains(lower, "logon"):
		return "The credentials were refused: check the username, the domain and the password."
	// Security before transport: a negotiation failure also mentions "connect".
	case strings.Contains(lower, "legacy") || strings.Contains(lower, "protocol") || strings.Contains(lower, "negotiat"):
		return "The server rejected the security settings: try Legacy TLS on the session bar for old servers."
	case strings.Contains(lower, "refused") || strings.Contains(lower, "connect") || strings.Contains(lower, "timeout") || strings.Contains(lower, "resolv"):
		return "The address did not answer: check the host, the port and the network."
	}
	return "Check the address, the credentials and the certificate policy (Ignore cert on the session bar)."
}

// setEmbeddedVisible shows or hides the embedded RDP window with its tab.
func (p *sessionPanel) setEmbeddedVisible(visible bool) {
	if p.embedded != nil {
		p.embedded.SetVisible(visible)
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

func (p *sessionPanel) testRDP() {
	if strings.TrimSpace(p.profile.Username) == "" {
		p.setProblem("Add a username in the Config panel and save the connection before testing.")
		return
	}
	p.rdpTest.Disable()
	p.setStatus("Testing RDP credentials against " + p.profile.Host + " ...")
	target := p.rdpTarget()
	options := rdpOptionsFromProfile(p.profile)
	options.Password = p.passwordFor()
	options.AuthOnly = true
	go func() {
		output, err := p.screen.probeRDP(target, options)
		fyne.Do(func() {
			p.rdpTest.Enable()
			if err != nil {
				p.setProblem("RDP auth-only check failed: " + err.Error() +
					". Some servers (for example xrdp) do not support auth-only; use Connect to open a real session.")
				return
			}
			p.setStatus(fmt.Sprintf("RDP authentication succeeded for %s (FreeRDP reported: %s)", p.profile.Username, strings.TrimSpace(output)))
		})
	}()
}

// requestRDPOptions edits the redirection flags and the gateway used by the next
// Connect.
func (p *sessionPanel) requestRDPOptions() {
	clipboard := widget.NewCheck("Clipboard", nil)
	printers := widget.NewCheck("Printers", nil)
	audio := widget.NewCheck("Audio to this machine", nil)
	dynamic := widget.NewCheck("Dynamic resolution (may break xrdp)", nil)
	multimon := widget.NewCheck("All monitors (separate window)", nil)
	drives := widget.NewEntry()
	current := rdpOptionsFromProfile(p.profile)
	clipboard.SetChecked(current.Clipboard)
	printers.SetChecked(current.Printers)
	audio.SetChecked(current.Audio)
	dynamic.SetChecked(current.DynamicResolution)
	multimon.SetChecked(current.MultiMonitor)
	drives.SetText(strings.Join(current.Drives, ","))
	drives.SetPlaceHolder("Shared drives: name:path,name:path")
	gatewayForm, gatewayBox := newRDPGatewayForm(current.Gateway)
	problem := widget.NewLabel("")
	problem.Importance = widget.DangerImportance
	problem.Wrapping = fyne.TextWrapWord

	var popup *modal
	content := container.NewVBox(
		widget.NewLabelWithStyle("RDP options", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		clipboard, printers, audio, dynamic, multimon, drives,
		widget.NewSeparator(), gatewayBox, problem,
		container.NewHBox(
			widget.NewButton("Apply", func() {
				gateway, err := gatewayForm.gateway()
				if err != nil {
					// Keep the dialog open: a rejected gateway must not look saved.
					problem.SetText(err.Error())
					return
				}
				options := rdpclient.Options{
					Clipboard:         clipboard.Checked,
					Printers:          printers.Checked,
					Audio:             audio.Checked,
					DynamicResolution: dynamic.Checked,
					MultiMonitor:      multimon.Checked,
					Security:          legacySecurity(p.rdpLegacy.Checked),
					IgnoreCertificate: p.rdpCert.Checked,
					Gateway:           gateway,
					Drives:            strings.Split(drives.Text, ","),
				}
				if err := p.saveRDPOptions(rdpOptionsToProfile(options)); err != nil {
					// Keep the dialog open: nothing was stored.
					problem.SetText(err.Error())
					return
				}
				popup.Hide()
				p.setStatus("RDP options saved for " + p.profile.Name + gatewaySummary(gateway) + ".")
			}),
			widget.NewButton("Cancel", func() { popup.Hide() }),
		),
	)
	popup = p.screen.showModal(content, fyne.NewSize(480, 440))
}

// rdpGatewayForm is the RD Gateway section of the options dialog. There is no
// password field on purpose: argv is readable by every process of the machine, and
// the client rejects gateway credentials that arrive without their password, so
// FreeRDP asks for them itself.
type rdpGatewayForm struct {
	host   *widget.Entry
	port   *widget.Entry
	method *widget.Select
}

// gatewayMethodLabels are the usage-method choices, in the order the client
// documents them.
var gatewayMethodLabels = []string{"client default", "direct", "detect"}

func newRDPGatewayForm(current rdpclient.Gateway) (*rdpGatewayForm, fyne.CanvasObject) {
	form := &rdpGatewayForm{
		host:   widget.NewEntry(),
		port:   widget.NewEntry(),
		method: widget.NewSelect(gatewayMethodLabels, nil),
	}
	form.host.SetPlaceHolder("gateway.example.com")
	form.host.SetText(current.Host)
	form.port.SetPlaceHolder("443")
	if current.Port != 0 {
		form.port.SetText(strconv.Itoa(current.Port))
	}
	form.method.SetSelected(gatewayMethodLabel(current.UsageMethod))
	note := widget.NewLabel("FreeRDP asks for the gateway password; it is never stored. " +
		"Direct always goes through the gateway, detect lets the server decide.")
	note.Importance = widget.LowImportance
	note.Wrapping = fyne.TextWrapWord
	box := container.NewVBox(
		widget.NewLabelWithStyle("RD Gateway", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form.host, form.port, form.method, note,
	)
	return form, box
}

// gateway reads the fields, refusing what the client would reject before it ever
// reaches an argument list.
func (f *rdpGatewayForm) gateway() (rdpclient.Gateway, error) {
	gateway := rdpclient.Gateway{
		Host:        strings.TrimSpace(f.host.Text),
		UsageMethod: gatewayMethodValue(f.method.Selected),
	}
	if text := strings.TrimSpace(f.port.Text); text != "" {
		port, err := strconv.Atoi(text)
		if err != nil {
			return gateway, fmt.Errorf("the gateway port %q is not a number", text)
		}
		gateway.Port = port
	}
	if err := gateway.Problem(); err != nil {
		return gateway, err
	}
	return gateway, nil
}

// gatewayMethodLabel turns a usage method into its select entry.
func gatewayMethodLabel(method string) string {
	if method == "direct" || method == "detect" {
		return method
	}
	return gatewayMethodLabels[0]
}

// gatewayMethodValue turns a select entry back into the stored value.
func gatewayMethodValue(label string) string {
	if label == "direct" || label == "detect" {
		return label
	}
	return ""
}

// gatewaySummary names the route for status lines, or nothing when there is no
// gateway.
func gatewaySummary(gateway rdpclient.Gateway) string {
	if strings.TrimSpace(gateway.Host) == "" {
		return ""
	}
	port := gateway.Port
	if port == 0 {
		port = 443
	}
	return fmt.Sprintf(" through gateway %s:%d", gateway.Host, port)
}

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
		fyne.Do(func() { p.applyConnect(target, options, shell, result, err) })
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
