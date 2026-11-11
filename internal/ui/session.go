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
	disconnect *widget.Button
	ctrlC      *widget.Button
	ctrlD      *widget.Button
	rdpTest    *widget.Button
	rdpLegacy  *widget.Check
	rdpCert    *widget.Check
	rdpOptions *widget.Button
	copyScreen *widget.Button
	// Redirection choices apply to the next Connect in this tab.
	rdpConfig rdpclient.Options

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
	// terminalTheme and terminalWrap are the single theme override this panel
	// keeps for zoom. The override takes effect during layout, so every zoom step
	// mutates the theme and then re-lays the slot.
	terminalTheme *zoomTheme
	terminalWrap  *container.ThemeOverride
	// rdpDetached is set when the user closes the session, so the watcher
	// goroutine never touches the UI (or a shutting-down app) afterwards.
	rdpDetached atomic.Bool

	hidConfig bool
}

func (s *screen) newSessionPanel(profile profiles.Profile) *sessionPanel {
	panel := &sessionPanel{screen: s, profile: profile}
	panel.password = widget.NewEntry()
	panel.password.Password = true
	panel.password.SetPlaceHolder("Password")

	panel.connect = widget.NewButtonWithIcon("Connect", theme.MediaPlayIcon(), panel.connectShell)
	panel.connect.Importance = widget.HighImportance
	panel.disconnect = widget.NewButtonWithIcon("Disconnect", theme.MediaStopIcon(), panel.disconnectShell)
	panel.disconnect.Importance = widget.LowImportance
	panel.disconnect.Disable()
	panel.ctrlC = widget.NewButton("Ctrl-C", func() { panel.sendControl(0x03) })
	panel.ctrlC.Importance = widget.LowImportance
	panel.ctrlD = widget.NewButton("Ctrl-D", func() { panel.sendControl(0x04) })
	panel.ctrlD.Importance = widget.LowImportance
	panel.rdpTest = widget.NewButtonWithIcon("Test", theme.SearchIcon(), panel.testRDP)
	panel.rdpLegacy = widget.NewCheck("Legacy TLS", nil)
	panel.rdpCert = widget.NewCheck("Ignore cert", nil)
	panel.rdpOptions = widget.NewButtonWithIcon("", theme.SettingsIcon(), panel.requestRDPOptions)
	panel.copyScreen = widget.NewButtonWithIcon("", theme.ContentCopyIcon(), panel.copyTerminalScreen)
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

// connectionBar is the only header above the terminal: which connection is
// open, the one-off password and the session controls.
func (p *sessionPanel) connectionBar() fyne.CanvasObject {
	name := widget.NewLabel(p.profile.Name)
	name.TextStyle = fyne.TextStyle{Bold: true}
	endpoint := widget.NewLabel(fmt.Sprintf("%s:%d", p.profile.Host, p.profile.Port))
	endpoint.Importance = widget.LowImportance
	// Name on the left, endpoint on the right, both readable.
	label := container.NewBorder(nil, nil, name, endpoint)

	buttons := []fyne.CanvasObject{p.connect, p.disconnect, p.vaultButton()}
	switch p.profile.Protocol {
	case profiles.RDP:
		buttons = append(buttons, p.rdpTest, p.rdpCert, p.rdpLegacy, p.rdpOptions)
	default:
		buttons = append(buttons, p.copyScreen, p.ctrlC, p.ctrlD)
	}
	// Name on its own row so it never collapses to an ellipsis; the password
	// shares the next row with the buttons. Fyne tab content scrolls
	// horizontally as soon as its minimum width exceeds the viewport.
	passwordRow := container.NewBorder(nil, nil, nil, container.NewHBox(buttons...), container.NewPadded(p.password))
	return container.NewVBox(container.NewPadded(label), passwordRow, widget.NewSeparator())
}

func (p *sessionPanel) rdpTarget() rdpclient.Target {
	resolved := p.screen.resolve(p.profile)
	return rdpclient.Target{Host: resolved.Host, Port: resolved.Port, Username: resolved.Username}
}

// rdpSecurity maps the explicit legacy checkbox to a FreeRDP security mode.
func (p *sessionPanel) rdpSecurity() string {
	if p.rdpLegacy != nil && p.rdpLegacy.Checked {
		return "rdp"
	}
	return ""
}

func (p *sessionPanel) connectRDP() {
	p.setConnecting("Starting FreeRDP for " + p.profile.Host + " ...")
	target := p.rdpTarget()
	options := p.rdpConfig
	options.Password = p.passwordFor()
	options.Security = p.rdpSecurity()
	options.IgnoreCertificate = p.rdpCert.Checked
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
			if time.Since(p.rdpStart) < 3*time.Second {
				p.setProblem("RDP session ended immediately. If FreeRDP reported a changed or self-signed certificate, enable Ignore certificate and try again.")
				return
			}
			p.setStatus("RDP session closed.")
		})
	}()
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
	options := p.rdpConfig
	options.Password = p.passwordFor()
	options.Security = p.rdpSecurity()
	options.IgnoreCertificate = p.rdpCert.Checked
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

// requestRDPOptions edits the redirection flags used by the next Connect.
func (p *sessionPanel) requestRDPOptions() {
	clipboard := widget.NewCheck("Clipboard", nil)
	printers := widget.NewCheck("Printers", nil)
	audio := widget.NewCheck("Audio to this machine", nil)
	dynamic := widget.NewCheck("Dynamic resolution (may break xrdp)", nil)
	multimon := widget.NewCheck("All monitors (separate window)", nil)
	drives := widget.NewEntry()
	clipboard.SetChecked(p.rdpConfig.Clipboard)
	printers.SetChecked(p.rdpConfig.Printers)
	audio.SetChecked(p.rdpConfig.Audio)
	dynamic.SetChecked(p.rdpConfig.DynamicResolution)
	multimon.SetChecked(p.rdpConfig.MultiMonitor)
	drives.SetText(strings.Join(p.rdpConfig.Drives, ","))
	drives.SetPlaceHolder("Shared drives: name:path,name:path")

	var popup *widget.PopUp
	content := container.NewVBox(
		widget.NewLabelWithStyle("RDP options", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		clipboard, printers, audio, dynamic, multimon, drives,
		container.NewHBox(
			widget.NewButton("Apply", func() {
				popup.Hide()
				options := p.rdpConfig
				options.Clipboard = clipboard.Checked
				options.Printers = printers.Checked
				options.Audio = audio.Checked
				options.DynamicResolution = dynamic.Checked
				options.MultiMonitor = multimon.Checked
				options.Drives = nil
				for _, drive := range strings.Split(drives.Text, ",") {
					if trimmed := strings.TrimSpace(drive); trimmed != "" {
						options.Drives = append(options.Drives, trimmed)
					}
				}
				p.rdpConfig = options
				p.setStatus("RDP options saved for the next Connect.")
			}),
			widget.NewButton("Cancel", func() { popup.Hide() }),
		),
	)
	popup = p.screen.showModal(content, fyne.NewSize(460, 300))
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
	p.forgetTerminalOverride()
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
	if p.terminalWrap != nil {
		p.terminalTheme.size = p.term.textSize()
		// The override applies its theme only while laying out its content, so a
		// theme change needs a layout pass, not just a refresh.
		p.slot.Refresh()
		p.term.fit(p.term.Size())
		p.term.render()
		return
	}
	p.terminalTheme = &zoomTheme{base: fyne.CurrentApp().Settings().Theme(), size: p.term.textSize()}
	p.terminalWrap = container.NewThemeOverride(p.term, p.terminalTheme)
	p.installSlot(p.terminalWrap)
}

// forgetTerminalOverride drops the zoom override of the session that ended. A
// new session installs a new terminal widget, and the previous wrapper still
// wraps the old one.
func (p *sessionPanel) forgetTerminalOverride() {
	p.terminalWrap = nil
	p.terminalTheme = nil
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
	term.onZoom = p.installTerminal
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
