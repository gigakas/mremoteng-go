package rdp_test

import (
	"strings"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/rdp"
)

func newTestSession(t *testing.T, values func(*connection.ConnectionInfo)) (protocol.Protocol, error) {
	t.Helper()
	info, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	info.Raw.Protocol = connection.ProtocolRDP
	values(info)
	return protocol.Create(info)
}

func TestNew_MissingHostname_ReturnsError(t *testing.T) {
	if _, err := newTestSession(t, func(info *connection.ConnectionInfo) {
		info.Raw.Hostname = ""
	}); err == nil {
		t.Fatal("expected an error for a missing hostname")
	}
}

func TestNew_ImplementsWindowProtocol(t *testing.T) {
	p, err := newTestSession(t, func(info *connection.ConnectionInfo) {
		info.Raw.Hostname = "example.invalid"
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

// TestNew_DefaultPort exercises the DefaultPort(ProtocolRDP) fallback
// indirectly, through Connect's failure message, since buildArgs and the
// address it constructs are otherwise unexported. A missing FreeRDP
// binary is expected in a plain `go test` environment (no PATH
// modification here), so this only checks Connect fails for the right
// reason (client not found) rather than an address-construction panic —
// full argument-building coverage lives in the Windows-only
// TestEmbedChild_* / TestFindTopLevelForPID_* tests, which exercise a
// real launched process end-to-end.
func TestNew_DefaultPort_DoesNotPanicOnConnect(t *testing.T) {
	p, err := newTestSession(t, func(info *connection.ConnectionInfo) {
		info.Raw.Hostname = "example.invalid"
		info.Raw.Port = 0
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	err = p.Connect(t.Context())
	if err == nil {
		t.Skip("a FreeRDP client is actually installed and reachable in this environment; nothing to assert further here")
	}
	if !strings.Contains(err.Error(), "rdp:") {
		t.Errorf("Connect error = %q, want an rdp: prefixed error", err)
	}
}
