package ui

import (
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strconv"
	"time"

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
		runOnUI(func() {
			if err != nil {
				p.setConnected(false)
				p.setProblem(string(mode) + " connection failed: " + err.Error())
				return
			}
			p.startStream(session, string(mode))
		})
	}()
}

// startStream installs the terminal view over the byte stream.
func (p *sessionPanel) startStream(session streamSession, label string) {
	p.stream = session
	p.term = newTermWidget(80, 24, func(data []byte) {
		// A write error means the stream closed; the read pump reports that.
		_, _ = session.Write(data)
	}, nil)
	// Same setup as an SSH terminal: clipboard, interrupt and zoom wiring, and
	// the theme override that makes zoom affect only the console.
	p.wireTerminal(p.term)
	p.installTerminal()
	p.setConnected(true)
	p.setStatus(label + " session with " + p.profile.Host + " connected. The terminal has no remote pty, so some full-screen programs may misbehave.")
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

// openExternalBrowser launches the default browser. Only http and https URLs
// are handed to xdg-open: a profile host must never become a file:// path or an
// arbitrary URL scheme handler invocation.
func openExternalBrowser(target string) error {
	if err := checkBrowserURL(target); err != nil {
		return err
	}
	return exec.Command("xdg-open", target).Start()
}

func checkBrowserURL(target string) error {
	parsed, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("open browser: refusing %q: only http and https URLs are opened", target)
	}
	return nil
}
