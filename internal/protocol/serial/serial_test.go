package serial_test

import (
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/serial"
)

// These tests cover construction/validation only. Exercising Connect
// against a real serial line needs actual hardware (or a virtual COM port
// pair such as com0com/socat), neither of which is available in this
// environment — see the stage audit's pending actions.

func newTestSession(t *testing.T, portName string, baud int) (protocol.Protocol, error) {
	t.Helper()
	info, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	info.Raw.Protocol = connection.ProtocolSerial
	info.Raw.Hostname = portName
	info.Raw.Port = baud
	return protocol.Create(info)
}

func TestNew_MissingPortName_ReturnsError(t *testing.T) {
	if _, err := newTestSession(t, "", 9600); err == nil {
		t.Fatal("expected an error for a missing port name")
	}
}

func TestNew_ValidPortName_Succeeds(t *testing.T) {
	p, err := newTestSession(t, "COM3", 115200)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p == nil {
		t.Fatal("Create returned a nil Protocol with no error")
	}
}

func TestNew_ZeroBaud_DefaultsTo9600(t *testing.T) {
	// baud 0 should fall back to connection.DefaultPort(ProtocolSerial),
	// not fail or open at 0 baud. Connect itself needs real hardware to
	// verify the fallback took effect, so this only checks construction
	// succeeds without a baud rate set.
	if _, err := newTestSession(t, "COM3", 0); err != nil {
		t.Fatalf("Create: %v", err)
	}
}
