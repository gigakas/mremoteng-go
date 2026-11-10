package anydesk_test

import (
	"strings"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/anydesk"
)

func newTestSession(t *testing.T, values func(*connection.ConnectionInfo)) (protocol.Protocol, error) {
	t.Helper()
	info, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	info.Raw.Protocol = connection.ProtocolAnyDesk
	values(info)
	return protocol.Create(info)
}

func TestNew_MissingAddress_ReturnsError(t *testing.T) {
	if _, err := newTestSession(t, func(info *connection.ConnectionInfo) {
		info.Raw.Hostname = ""
	}); err == nil {
		t.Fatal("expected an error for a missing address")
	}
}

func TestNew_ImplementsWindowProtocol(t *testing.T) {
	p, err := newTestSession(t, func(info *connection.ConnectionInfo) {
		info.Raw.Hostname = "123456789"
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	win, ok := p.(protocol.WindowProtocol)
	if !ok {
		t.Fatalf("Create returned %T, want a protocol.WindowProtocol", p)
	}
	if h := win.NativeWindowHandle(); h != 0 {
		t.Errorf("NativeWindowHandle() = %x before Connect, want 0", h)
	}
}

// TestConnect_ClientNotOnPath_ReturnsClearError is the only behavior
// exercisable in this environment: AnyDesk is not installed here (and was
// deliberately not fetched — see the package doc comment), so Connect's
// LookPath failure path is what's actually testable. Real client-launch
// and window-discovery behavior is unverified; see the stage audit.
func TestConnect_ClientNotOnPath_ReturnsClearError(t *testing.T) {
	p, err := newTestSession(t, func(info *connection.ConnectionInfo) {
		info.Raw.Hostname = "123456789"
		info.Raw.Password = "s3cret"
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	err = p.Connect(t.Context())
	if err == nil {
		t.Skip("an AnyDesk client is actually installed and reachable in this environment; nothing to assert further here")
	}
	if !strings.Contains(err.Error(), "anydesk:") {
		t.Errorf("Connect error = %q, want an anydesk: prefixed error", err)
	}
}
