// Package raw implements protocol.TerminalProtocol as a plain TCP socket
// with no protocol-level negotiation at all — mRemoteNG's "RAW" connection
// type, typically used to talk directly to a device's management port.
package raw

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
)

func init() {
	protocol.Register(connection.ProtocolRAW, New)
}

// Session is a raw TCP passthrough.
type Session struct {
	protocol.Lifecycle
	address string
	stream  *protocol.WatchedStream
}

// New builds a Session for info. It implements protocol.Constructor.
func New(_ *connection.ConnectionInfo, values connection.ConnectionValues) (protocol.Protocol, error) {
	if values.Hostname == "" {
		return nil, fmt.Errorf("raw: hostname is required")
	}
	port := values.Port
	if port <= 0 {
		port = connection.DefaultPort(connection.ProtocolRAW)
	}
	s := &Session{address: net.JoinHostPort(values.Hostname, strconv.Itoa(port))}
	return s, nil
}

// Connect implements protocol.Protocol.
func (s *Session) Connect(ctx context.Context) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", s.address)
	if err != nil {
		return fmt.Errorf("raw: dial %s: %w", s.address, err)
	}
	s.stream = protocol.NewWatchedStream(&s.Lifecycle, conn)
	return nil
}

// Disconnect implements protocol.Protocol.
func (s *Session) Disconnect() error {
	var err error
	if s.stream != nil {
		err = s.stream.Close()
	}
	s.FireClose()
	return err
}

// Focus implements protocol.Protocol. A raw socket has no input focus
// concept beyond the terminal widget that renders it, so this is a no-op.
func (s *Session) Focus() {}

// Resize implements protocol.Protocol. A raw socket carries no
// out-of-band signaling channel to notify the remote end of a size
// change, so this is a no-op.
func (s *Session) Resize(width, height int) {}

// Read implements io.Reader by delegating to the underlying connection.
func (s *Session) Read(p []byte) (int, error) { return s.stream.Read(p) }

// Write implements io.Writer by delegating to the underlying connection.
func (s *Session) Write(p []byte) (int, error) { return s.stream.Write(p) }

var _ protocol.TerminalProtocol = (*Session)(nil)
