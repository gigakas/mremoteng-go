package web_test

import (
	"context"
	"testing"
	"time"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/web"
)

func newTestSession(t *testing.T, proto connection.ProtocolType, host string, port int) (protocol.Protocol, error) {
	t.Helper()
	info, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	info.Raw.Protocol = proto
	info.Raw.Hostname = host
	info.Raw.Port = port
	return protocol.Create(info)
}

func TestNew_MissingHostname_ReturnsError(t *testing.T) {
	if _, err := newTestSession(t, connection.ProtocolHTTP, "", 0); err == nil {
		t.Fatal("expected an error for a missing hostname")
	}
}

func TestNew_RegistersBothHTTPAndHTTPS(t *testing.T) {
	for _, proto := range []connection.ProtocolType{connection.ProtocolHTTP, connection.ProtocolHTTPS} {
		if _, err := newTestSession(t, proto, "example.invalid", 0); err != nil {
			t.Errorf("Create(%s): %v", proto, err)
		}
	}
}

// TestSession_ConnectAndDisconnect_CreatesAndClosesANativeWindow is a real
// integration test: it creates an actual OS webview window (WebView2 on
// Windows) and tears it down, not just a construction stub. It's bounded
// by a context timeout precisely because this needs a real, interactive
// window station — if a future CI/headless environment lacks one, this
// fails cleanly within the timeout instead of hanging the whole test
// binary (Connect's underlying goroutine has no way to abort a stuck
// native window-creation call, so a genuine hang there would still leak
// that one goroutine, but the test itself won't block on it).
func TestSession_ConnectAndDisconnect_CreatesAndClosesANativeWindow(t *testing.T) {
	p, err := newTestSession(t, connection.ProtocolHTTP, "example.invalid", 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := p.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v (needs a real, interactive window station)", err)
	}

	win, ok := p.(protocol.WindowProtocol)
	if !ok {
		t.Fatalf("Create returned %T, want a protocol.WindowProtocol", p)
	}
	if h := win.NativeWindowHandle(); h == 0 {
		t.Error("NativeWindowHandle() = 0 after a successful Connect")
	}

	closed := make(chan struct{})
	p.OnClose(func() { close(closed) })

	// Resize must not panic or block against a live window.
	p.Resize(640, 480)

	if err := p.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}

	select {
	case <-closed:
	case <-time.After(10 * time.Second):
		t.Fatal("OnClose was not fired after Disconnect")
	}

	// Idempotent.
	if err := p.Disconnect(); err != nil {
		t.Fatalf("second Disconnect: %v", err)
	}
}
