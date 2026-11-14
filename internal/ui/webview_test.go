package ui

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

// fakeWebSession is a helper process that stays alive until the test ends it.
type fakeWebSession struct {
	once   sync.Once
	done   chan struct{}
	closed bool
	output string
}

func newFakeWebSession() *fakeWebSession {
	return &fakeWebSession{done: make(chan struct{})}
}

func (f *fakeWebSession) Wait() error { <-f.done; return nil }

func (f *fakeWebSession) Close() error {
	f.once.Do(func() {
		f.closed = true
		close(f.done)
	})
	return nil
}

func (f *fakeWebSession) Output() string { return f.output }

func httpProfile(t *testing.T, port int) profiles.Profile {
	t.Helper()
	return sampleProfile(t, "Intranet", "", string(profiles.HTTP), "intranet.example.com", port)
}

// httpScreen builds a screen whose embedded views are fake and whose browser calls are
// recorded instead of launched.
func httpScreen(t *testing.T) (*screen, *[]string) {
	t.Helper()
	s := newTestScreen(t, &fakeStore{})
	opened := make([]string, 0, 1)
	s.openBrowser = func(url string) error {
		opened = append(opened, url)
		return nil
	}
	return s, &opened
}

func TestHTTPURL_KeepsTheSchemePortsOutOfTheAddress(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	cases := map[int]string{
		80:   "http://intranet.example.com",
		443:  "https://intranet.example.com",
		8080: "http://intranet.example.com:8080",
	}
	for port, want := range cases {
		panel := s.newSessionPanel(httpProfile(t, port))
		if got := panel.httpURL(); got != want {
			t.Errorf("port %d: URL = %q, want %q", port, got, want)
		}
	}
}

// TestConnectHTTP_EmbedsTheViewInTheTab is the feature: the page lives in the tab, in a
// window the application owns, instead of a browser window.
func TestConnectHTTP_EmbedsTheViewInTheTab(t *testing.T) {
	s, opened := httpScreen(t)
	area := &fakeArea{id: 51}
	s.embedder = func(fyne.Window, fyne.CanvasObject, fyne.CanvasObject) (embeddedArea, error) {
		return area, nil
	}
	session := newFakeWebSession()
	type call struct {
		window uint64
		url    string
	}
	calls := make(chan call, 1)
	s.launchWeb = func(window uint64, url string) (webSession, error) {
		calls <- call{window: window, url: url}
		return session, nil
	}
	profile := httpProfile(t, 80)
	panel := s.newSessionPanel(profile)

	panel.connectHTTP()

	select {
	case got := <-calls:
		if got.window != 51 {
			t.Errorf("window = %d, want the embedded area", got.window)
		}
		if got.url != "http://intranet.example.com" {
			t.Errorf("url = %q", got.url)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the helper was not started")
	}
	if !waitFor(t, func() bool { return strings.Contains(s.status.Text, "embedded in the tab") }) {
		t.Fatalf("status = %q, want the embedded session reported", s.status.Text)
	}
	if panel.web != session {
		t.Error("the panel does not keep the web session")
	}
	if panel.embedded != area {
		t.Error("the panel does not keep the window the view lives in")
	}
	if len(*opened) != 0 {
		t.Errorf("the browser was opened as well: %v", *opened)
	}

	panel.disconnectShell()
	if !session.closed {
		t.Error("disconnect did not end the helper")
	}
	if panel.web != nil || panel.embedded != nil {
		t.Error("the panel still holds the closed view")
	}
}

func TestConnectHTTP_WithoutAWindow_OpensTheBrowser(t *testing.T) {
	s, opened := httpScreen(t)
	s.embedder = func(fyne.Window, fyne.CanvasObject, fyne.CanvasObject) (embeddedArea, error) {
		return nil, ErrNoX11
	}
	panel := s.newSessionPanel(httpProfile(t, 80))

	panel.connectHTTP()

	if len(*opened) != 1 || (*opened)[0] != "http://intranet.example.com" {
		t.Fatalf("opened = %v, want the browser with the address", *opened)
	}
	if !strings.Contains(s.status.Text, "system browser") {
		t.Errorf("status = %q, want the fallback named", s.status.Text)
	}
	if !strings.Contains(s.status.Text, "no window to embed into") {
		t.Errorf("status = %q, want the reason", s.status.Text)
	}
}

func TestConnectHTTP_HelperFailure_FallsBackToTheBrowser(t *testing.T) {
	s, opened := httpScreen(t)
	area := &fakeArea{id: 52}
	s.embedder = func(fyne.Window, fyne.CanvasObject, fyne.CanvasObject) (embeddedArea, error) {
		return area, nil
	}
	s.launchWeb = func(uint64, string) (webSession, error) {
		return nil, errors.New("mremoteng-webview: no such file")
	}
	panel := s.newSessionPanel(httpProfile(t, 80))

	panel.connectHTTP()

	if !waitFor(t, func() bool { return len(*opened) == 1 }) {
		t.Fatalf("opened = %v, want the browser as a fallback", *opened)
	}
	if !area.destroyed {
		t.Error("the unused window was left behind")
	}
	if !strings.Contains(s.status.Text, "could not start") {
		t.Errorf("status = %q, want the reason", s.status.Text)
	}
	if panel.web != nil {
		t.Error("a failed launch left a web session behind")
	}
}

func TestWatchWebView_ReportsTheHelperCause(t *testing.T) {
	s, _ := httpScreen(t)
	profile := httpProfile(t, 80)
	s.openSessionTab(profile)
	panel := s.panels[profile.ID]
	session := newFakeWebSession()
	session.output = "\n(WebKitWebProcess:1234): Gtk-WARNING **: 22:00:00.000: cannot open display\n"
	panel.web = session
	panel.embedded = &fakeArea{id: 53}
	panel.setConnected(true)
	go panel.watchWebView(session)

	_ = session.Close() // the helper exits on its own

	if !waitFor(t, func() bool { return s.notifications.count() > 0 }) {
		t.Fatalf("status = %q, want the closing reported", s.status.Text)
	}
	entry := s.notifications.entries[0]
	if entry.Level != noticeWarning {
		t.Errorf("level = %v, want a warning", entry.Level)
	}
	if !strings.Contains(entry.Text, "cannot open display") {
		t.Errorf("entry = %q, want the helper's own line", entry.Text)
	}
	if panel.web != nil {
		t.Error("the panel kept a view that exited")
	}
}

// TestDisconnectHTTP_WithoutAView_DoesNothing guards the cleanup path: HTTP panels that
// never connected must not touch a nil session.
func TestDisconnectHTTP_WithoutAView_DoesNothing(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	panel := s.newSessionPanel(httpProfile(t, 80))

	panel.disconnectShell()

	if panel.web != nil {
		t.Error("a disconnected panel invented a web session")
	}
}

func TestWebViewNote_DependsOnTheHelperBeingAvailable(t *testing.T) {
	note := webViewNote()
	if webHelperAvailable() {
		if !strings.Contains(note, "embeds the page") {
			t.Errorf("note = %q, want it to promise the embedded view", note)
		}
		return
	}
	if !strings.Contains(note, "system browser") || !strings.Contains(note, "mremoteng-webview") {
		t.Errorf("note = %q, want it to explain the fallback and the missing helper", note)
	}
}
