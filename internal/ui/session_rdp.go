package ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	rdpclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/rdp"
)

// This file holds the RDP side of a session tab: the external FreeRDP launch,
// the embedded window, the connection test and the gateway form.

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

func (p *sessionPanel) rdpTarget() rdpclient.Target {
	resolved := p.screen.resolve(p.profile)
	return rdpclient.Target{Host: resolved.Host, Port: resolved.Port, Username: resolved.Username}
}

func (p *sessionPanel) connectRDP() {
	options := rdpOptionsFromProfile(p.profile)
	p.setConnecting("Starting FreeRDP for " + p.profile.Host + gatewaySummary(options.Gateway) + " ...")
	target := p.rdpTarget()
	options.Password = p.passwordFor()
	var area embeddedArea
	var embedErr error
	if options.MultiMonitor {
		// A session spanning several monitors needs FreeRDP's own window: an embedded
		// one is confined to the tab, which is the opposite of what the flag asks for.
		embedErr = errors.New("a multi-monitor session opens in its own window")
	} else {
		area, embedErr = p.embedSession()
		if area != nil {
			options.ParentWindowID = area.ID()
		}
	}
	go func() {
		session, err := p.screen.launchRDP(target, options)
		runOnUI(func() { p.applyConnectRDPWithArea(session, area, embedErr, err) })
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
		runOnUI(func() {
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
			p.screen.setWarning("RDP session closed: " + cause)
		})
	}()
}

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
		runOnUI(func() {
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
