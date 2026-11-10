// Package serial implements protocol.TerminalProtocol for a local serial
// (COM/tty) port, via go.bug.st/serial — the standard library has no
// cross-platform serial port support, and this is the most widely used,
// actively maintained pure-Go implementation (no cgo), justifying the new
// dependency per AGENTS.md.
package serial

import (
	"context"
	"fmt"

	sp "go.bug.st/serial"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
)

func init() {
	protocol.Register(connection.ProtocolSerial, New)
}

// Session is a serial-port connection. Like the original mRemoteNG (which
// drives PuTTY's -serial mode for this protocol), the connection model has
// no dedicated serial fields: Hostname holds the port name (e.g. "COM3" on
// Windows, "/dev/ttyUSB0" on Linux) and Port holds the baud rate.
type Session struct {
	protocol.Lifecycle
	portName string
	baudRate int
	stream   *protocol.WatchedStream
}

// New builds a Session for info. It implements protocol.Constructor.
func New(_ *connection.ConnectionInfo, values connection.ConnectionValues) (protocol.Protocol, error) {
	if values.Hostname == "" {
		return nil, fmt.Errorf("serial: port name is required (set in the connection's Hostname field)")
	}
	baud := values.Port
	if baud <= 0 {
		baud = connection.DefaultPort(connection.ProtocolSerial)
	}
	return &Session{portName: values.Hostname, baudRate: baud}, nil
}

// Connect implements protocol.Protocol by opening the serial port at 8N1
// (8 data bits, no parity, 1 stop bit — the near-universal default).
// Opening a local serial port is effectively instantaneous, so ctx is only
// checked up front rather than threaded through the call.
func (s *Session) Connect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	mode := &sp.Mode{
		BaudRate: s.baudRate,
		DataBits: 8,
		Parity:   sp.NoParity,
		StopBits: sp.OneStopBit,
	}
	port, err := sp.Open(s.portName, mode)
	if err != nil {
		return fmt.Errorf("serial: open %s: %w", s.portName, err)
	}
	s.stream = protocol.NewWatchedStream(&s.Lifecycle, port)
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

// Focus implements protocol.Protocol; the terminal widget owns input
// focus, so this is a no-op.
func (s *Session) Focus() {}

// Resize implements protocol.Protocol. A serial line has no terminal-size
// concept of its own beyond whatever the connected device assumes, so
// this is a no-op.
func (s *Session) Resize(width, height int) {}

// Read implements io.Reader over the serial port.
func (s *Session) Read(p []byte) (int, error) { return s.stream.Read(p) }

// Write implements io.Writer over the serial port.
func (s *Session) Write(p []byte) (int, error) { return s.stream.Write(p) }

var _ protocol.TerminalProtocol = (*Session)(nil)
