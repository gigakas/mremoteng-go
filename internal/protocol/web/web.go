// Package web implements protocol.WindowProtocol for HTTP/HTTPS via an
// OS-native webview (WebView2 on Windows, WebKitGTK on Linux), through
// github.com/webview/webview_go — the only place in this module where cgo
// is acceptable, and only through this wrapper library, per the blueprint.
package web

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"runtime"
	"strconv"
	"sync"

	"github.com/webview/webview_go"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
)

func init() {
	protocol.Register(connection.ProtocolHTTP, New)
	protocol.Register(connection.ProtocolHTTPS, New)
}

const (
	defaultWidth  = 1024
	defaultHeight = 768
)

// Session is a webview-backed HTTP/HTTPS session. The webview library
// creates its own native top-level window and runs a blocking OS message
// loop on the thread that created it (webview_go's Run(), documented as
// such) — Session.Connect starts that on a dedicated locked OS thread and
// returns once the window exists and navigation has started, rather than
// waiting for Run to return (which only happens at session end). The
// window is exposed via NativeWindowHandle for the eventual Phase 3 UI to
// reparent into a tab, the same pattern Phase 0 validated for RDP.
type Session struct {
	protocol.Lifecycle
	targetURL string

	mu        sync.Mutex
	wv        webview.WebView
	hwnd      uintptr
	closeOnce sync.Once
}

// New builds a Session for info. It implements protocol.Constructor and
// is registered for both ProtocolHTTP and ProtocolHTTPS — the only
// difference between them is the URL scheme.
func New(_ *connection.ConnectionInfo, values connection.ConnectionValues) (protocol.Protocol, error) {
	if values.Hostname == "" {
		return nil, fmt.Errorf("web: hostname is required")
	}

	scheme := "http"
	defaultPort := connection.DefaultPort(connection.ProtocolHTTP)
	if values.Protocol == connection.ProtocolHTTPS {
		scheme = "https"
		defaultPort = connection.DefaultPort(connection.ProtocolHTTPS)
	}

	host := values.Hostname
	if values.Port > 0 && values.Port != defaultPort {
		host = net.JoinHostPort(values.Hostname, strconv.Itoa(values.Port))
	}

	u := url.URL{Scheme: scheme, Host: host}
	return &Session{targetURL: u.String()}, nil
}

// Connect implements protocol.Protocol.
func (s *Session) Connect(ctx context.Context) error {
	ready := make(chan error, 1)
	go s.run(ready)

	select {
	case err := <-ready:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// run creates the native webview window and its message loop on a locked
// OS thread (required by both Win32 and Cocoa: a window's message loop
// must run on the thread that created it) and blocks in w.Run() until
// Disconnect calls w.Terminate(). It signals readiness on ready as soon as
// the window exists and navigation has been requested — matching
// Protocol.Connect's contract of returning once the session has started,
// not once the page has finished loading.
func (s *Session) run(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	w := webview.New(false)
	defer func() {
		w.Destroy()
		s.FireClose()
	}()

	w.SetTitle(s.targetURL)
	w.SetSize(defaultWidth, defaultHeight, webview.HintNone)
	w.Navigate(s.targetURL)

	s.mu.Lock()
	s.wv = w
	s.hwnd = uintptr(w.Window())
	s.mu.Unlock()

	ready <- nil

	w.Run()
}

// Disconnect implements protocol.Protocol.
//
// Terminate is called through Dispatch rather than directly, despite
// webview_go's doc comment claiming Terminate is itself safe to call from
// another thread: on the Windows backend, terminate_impl is a bare
// PostQuitMessage(0), which is thread-local (it posts WM_QUIT to the
// *calling* thread's queue, not the message loop's) — calling it directly
// from Disconnect's goroutine silently does nothing, confirmed by a failing
// test before this fix. Dispatch correctly marshals across threads via
// PostMessageW to the webview's message-only window, so routing Terminate
// through it fixes Windows and is a harmless no-op extra hop on the GTK
// backend (whose terminate_impl already dispatches internally).
func (s *Session) Disconnect() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		w := s.wv
		s.mu.Unlock()
		if w != nil {
			w.Dispatch(func() { w.Terminate() })
		}
	})
	return nil
}

// Focus implements protocol.Protocol as a no-op: once the Phase 3 UI
// reparents this window into a tab, giving it input focus is a native
// window-management operation the UI performs on the handle from
// NativeWindowHandle, not something this backend can meaningfully do
// itself beforehand.
func (s *Session) Focus() {}

// Resize implements protocol.Protocol in pixels (this is a window-embedded
// backend, not a TerminalProtocol). Dispatched onto the webview's own
// thread, as webview_go's docs require for anything touching the native
// window outside its own message loop.
func (s *Session) Resize(width, height int) {
	s.mu.Lock()
	w := s.wv
	s.mu.Unlock()
	if w == nil {
		return
	}
	w.Dispatch(func() {
		w.SetSize(width, height, webview.HintNone)
	})
}

// NativeWindowHandle implements protocol.WindowProtocol.
func (s *Session) NativeWindowHandle() uintptr {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hwnd
}

var _ protocol.WindowProtocol = (*Session)(nil)
