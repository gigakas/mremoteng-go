package ui

import (
	"net"
	"strconv"

	"fyne.io/fyne/v2"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	webviewclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/webview"
)

// webSession is an embedded WebKit view running in its helper process; tests
// substitute it.
type webSession interface {
	Wait() error
	Close() error
	Output() string
}

// webLauncher starts the helper for a window; tests substitute it.
type webLauncher func(windowID uint64, url string) (webSession, error)

func defaultWebLauncher(windowID uint64, url string) (webSession, error) {
	return webviewclient.Launch(windowID, url, webviewclient.Options{})
}

// httpURL is the address an HTTP connection opens. Ports 80 and 443 are the scheme's
// own, so they stay out of the URL.
func (p *sessionPanel) httpURL() string {
	resolved := p.screen.resolve(p.profile)
	scheme := "http://"
	port := resolved.Port
	if port == 0 {
		port = profiles.DefaultPort(profiles.HTTP)
	}
	if port == 443 {
		scheme = "https://"
	}
	if port == 80 || port == 443 {
		return scheme + resolved.Host
	}
	return scheme + net.JoinHostPort(resolved.Host, strconv.Itoa(port))
}

// connectHTTP embeds the page in the tab.
//
// The view is a native window — the helper is a separate process with its own GTK loop —
// placed in the same area FreeRDP uses, so switching tabs, hiding the session behind a
// dialog and resizing the window all work through the code that already exists. When
// embedding is impossible the address opens in the system browser and the status line
// says which of the two happened.
func (p *sessionPanel) connectHTTP() {
	url := p.httpURL()
	p.setConnecting("Opening " + url + " ...")
	area, embedErr := p.embedSession()
	if embedErr != nil {
		p.openInBrowser(url, "there is no window to embed into ("+embedErr.Error()+")")
		return
	}
	go func() {
		session, err := p.screen.launchWeb(area.ID(), url)
		fyne.Do(func() {
			if err != nil {
				area.Destroy()
				p.setConnected(false)
				p.openInBrowser(url, "the embedded view could not start ("+err.Error()+")")
				return
			}
			p.web = session
			p.embedded = area
			p.setConnected(true)
			p.setStatus("HTTP session for " + p.profile.Host + " embedded in the tab.")
			go p.watchWebView(session)
		})
	}()
}

// watchWebView reports the view closing, unless the user closed it first.
func (p *sessionPanel) watchWebView(session webSession) {
	_ = session.Wait()
	fyne.Do(func() {
		if p.web != session {
			// Closed by disconnectShell, which has already cleaned up.
			return
		}
		p.web = nil
		p.releaseEmbedded()
		p.setConnected(false)
		if cause := webviewclient.Summarize(session.Output()); cause != "" {
			p.screen.setWarning("The embedded view closed: " + cause)
			return
		}
		p.setStatus("The embedded view closed.")
	})
}

// openInBrowser is the fallback when the page cannot be embedded.
func (p *sessionPanel) openInBrowser(url, why string) {
	if err := p.screen.openBrowser(url); err != nil {
		p.setProblem("Could not open " + url + ": " + err.Error())
		return
	}
	p.setStatus("Opened " + url + " in the system browser because " + why + ".")
}

// webViewNote is the placeholder the HTTP tab shows before Connect, and it must match
// what Connect will actually do.
func webViewNote() string {
	if webHelperAvailable() {
		return "Connect embeds the page in this tab with a WebKit view. It runs in a helper " +
			"process, so its own certificates and logins behave as in a browser."
	}
	return "Connect opens the address in the system browser: the embedded WebKit helper " +
		"(mremoteng-webview) was not found next to the application or on the PATH."
}

// webHelperAvailable reports whether the embedded view can be offered at all.
func webHelperAvailable() bool {
	_, err := webviewclient.FindExecutable()
	return err == nil
}

// closeWebView ends the helper, if one is running.
func (p *sessionPanel) closeWebView() {
	if p.web == nil {
		return
	}
	session := p.web
	p.web = nil
	_ = session.Close()
}
