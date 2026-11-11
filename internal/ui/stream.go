package ui

import (
	"net"
	"os/exec"
	"strconv"
	"time"

	"fyne.io/fyne/v2"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	telnetclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/telnet"
)

// streamSession is a connected byte stream (Telnet, RAW, rlogin).
type streamSession interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
}

// streamDialer opens a byte stream; tests substitute it.
type streamDialer func(address string, config telnetclient.Config) (streamSession, error)

func defaultStreamDialer(address string, config telnetclient.Config) (streamSession, error) {
	conn, err := telnetclient.Dial(address, config)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// streamMode maps a protocol to its wire mode.
func streamMode(protocol profiles.Protocol) (telnetclient.Mode, bool) {
	switch protocol {
	case profiles.TELNET:
		return telnetclient.ModeTelnet, true
	case profiles.RAW:
		return telnetclient.ModeRaw, true
	case profiles.RLOGIN:
		return telnetclient.ModeRlogin, true
	default:
		return "", false
	}
}

// connectStream opens a Telnet/RAW/rlogin session in the terminal view.
func (p *sessionPanel) connectStream() {
	mode, ok := streamMode(p.profile.Protocol)
	if !ok {
		p.setProblem("Unsupported stream protocol.")
		return
	}
	p.setConnecting("Connecting to " + p.profile.Host + " (" + string(mode) + ") ...")
	resolved := p.screen.resolve(p.profile)
	address := net.JoinHostPort(resolved.Host, strconv.Itoa(resolved.Port))
	config := telnetclient.Config{Mode: mode, Username: resolved.Username, Timeout: 10 * time.Second}
	go func() {
		session, err := p.screen.dialStream(address, config)
		fyne.Do(func() {
			if err != nil {
				p.setConnected(false)
				p.setProblem(string(mode) + " connection failed: " + err.Error())
				return
			}
			p.startStream(session, mode)
		})
	}()
}

// startStream installs the terminal view over the byte stream.
func (p *sessionPanel) startStream(session streamSession, mode telnetclient.Mode) {
	p.stream = session
	p.term = newTermWidget(80, 24, func(data []byte) { _, _ = session.Write(data) }, nil)
	p.installSlot(p.term)
	p.setConnected(true)
	p.setStatus(string(mode) + " session with " + p.profile.Host + " connected. The terminal has no remote pty, so some full-screen programs may misbehave.")
	go p.readStream(session)
}

func (p *sessionPanel) readStream(session streamSession) {
	pumpBytes(session, p.applyOutput, func() { p.streamClosed(session) })
}

func (p *sessionPanel) streamClosed(session streamSession) {
	if p.stream != session {
		return
	}
	p.stream = nil
	p.setConnected(false)
	p.setStatus("Session closed.")
}

// connectHTTP opens the address in the system browser: embedding a GTK webview
// inside the Fyne window needs window parenting and is not implemented yet.
func (p *sessionPanel) connectHTTP() {
	resolved := p.screen.resolve(p.profile)
	scheme := "http://"
	url := scheme + net.JoinHostPort(resolved.Host, strconv.Itoa(resolved.Port))
	if resolved.Port == 80 {
		url = scheme + resolved.Host
	}
	if err := p.screen.openBrowser(url); err != nil {
		p.setProblem("Could not open the browser: " + err.Error())
		return
	}
	p.setStatus("Opened " + url + " in the system browser; embedding a webview in the window is not implemented yet.")
}

// openExternalBrowser launches the default browser.
func openExternalBrowser(url string) error {
	return exec.Command("xdg-open", url).Start()
}
