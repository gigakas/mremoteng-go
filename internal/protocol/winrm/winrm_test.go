package winrm_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/dylanmei/winrmtest"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/winrm"
)

func newTestSession(t *testing.T, host string, port int) (protocol.Protocol, error) {
	t.Helper()
	info, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	info.Raw.Protocol = connection.ProtocolPowerShell
	info.Raw.Hostname = host
	info.Raw.Port = port
	info.Raw.Username = "alice"
	info.Raw.Password = "s3cret"
	return protocol.Create(info)
}

func TestNew_MissingHostname_ReturnsError(t *testing.T) {
	if _, err := newTestSession(t, "", 0); err == nil {
		t.Fatal("expected an error for a missing hostname")
	}
}

// TestSession_ConnectAndShell_EchoesData runs a real WinRM shell exchange
// against github.com/dylanmei/winrmtest's fake server (the standard test
// companion for masterzen/winrm — using it instead of hand-rolling
// WS-Management/SOAP XML avoids getting an untested protocol
// implementation subtly wrong on both ends of the same test). It proves
// the actual wire exchange (CreateShell, Execute, stdin/stdout streaming)
// works end-to-end, not just construction.
func TestSession_ConnectAndShell_EchoesData(t *testing.T) {
	remote := winrmtest.NewRemote()
	defer remote.Close()

	remote.CommandFunc(winrmtest.MatchText("cmd.exe"), func(out, errW io.Writer) int {
		io.WriteString(out, "hello from fake winrmd\r\n")
		return 0
	})

	p, err := newTestSession(t, remote.Host, remote.Port)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := p.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer p.Disconnect()

	term := p.(protocol.TerminalProtocol)
	buf := make([]byte, 256)
	n, err := readUntilEOFOrData(term, buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read: %v", err)
	}
	if got := string(buf[:n]); got != "hello from fake winrmd\r\n" {
		t.Errorf("output = %q, want %q", got, "hello from fake winrmd\r\n")
	}
}

// readUntilEOFOrData reads once and returns immediately once any data
// arrives, tolerating a leading empty read some transports/mocks produce.
func readUntilEOFOrData(r io.Reader, buf []byte) (int, error) {
	for {
		n, err := r.Read(buf)
		if n > 0 || err != nil {
			return n, err
		}
	}
}

// TestSession_Disconnect_IsIdempotent reads at least one byte before
// disconnecting, matching every real caller's actual usage pattern (a
// terminal widget starts reading as soon as a session opens). Disconnect
// *before* ever reading is a separate, known, timing-sensitive gap
// against the winrmtest fake server specifically — see Disconnect's doc
// comment and the stage audit — deliberately not exercised here to keep
// this test deterministic.
func TestSession_Disconnect_IsIdempotent(t *testing.T) {
	remote := winrmtest.NewRemote()
	defer remote.Close()

	remote.CommandFunc(winrmtest.MatchText("cmd.exe"), func(out, errW io.Writer) int {
		io.WriteString(out, "ok\r\n")
		return 0
	})

	p, err := newTestSession(t, remote.Host, remote.Port)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := p.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	term := p.(protocol.TerminalProtocol)
	buf := make([]byte, 256)
	if _, err := readUntilEOFOrData(term, buf); err != nil && err != io.EOF {
		t.Fatalf("Read: %v", err)
	}

	disconnected := make(chan error, 2)
	go func() { disconnected <- p.Disconnect() }()
	select {
	case err := <-disconnected:
		if err != nil {
			t.Fatalf("first Disconnect: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("first Disconnect did not return within 15s")
	}

	go func() { disconnected <- p.Disconnect() }()
	select {
	case err := <-disconnected:
		if err != nil {
			t.Fatalf("second Disconnect: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("second Disconnect did not return within 15s")
	}
}
