// Package rlogin implements protocol.TerminalProtocol for the BSD rlogin
// protocol (RFC 1282): a thin handshake over a plain TCP socket, after
// which the stream is transparent terminal data. It is a legacy,
// unencrypted protocol kept for compatibility with the original mRemoteNG;
// prefer SSH wherever the remote host supports it.
package rlogin

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
)

func init() {
	protocol.Register(connection.ProtocolRlogin, New)
}

// defaultTermSpeed is sent as the terminal type/speed field of the
// handshake when nothing more specific is known. Real terminal type
// negotiation is the terminal emulator widget's job (Phase 3); this only
// needs to be a value the remote rlogind accepts.
const defaultTermSpeed = "xterm/38400"

// Session is an rlogin client session.
type Session struct {
	protocol.Lifecycle
	address    string
	clientUser string
	serverUser string
	stream     *protocol.WatchedStream
}

// New builds a Session for info. It implements protocol.Constructor.
// mRemoteNG's connection model has a single Username field, used here as
// both the local and remote rlogin user — the original protocol allows
// them to differ (e.g. for .rhosts trust mapping), but the data model has
// no second field for it.
func New(_ *connection.ConnectionInfo, values connection.ConnectionValues) (protocol.Protocol, error) {
	if values.Hostname == "" {
		return nil, fmt.Errorf("rlogin: hostname is required")
	}
	if values.Username == "" {
		return nil, fmt.Errorf("rlogin: username is required")
	}
	port := values.Port
	if port <= 0 {
		port = connection.DefaultPort(connection.ProtocolRlogin)
	}
	return &Session{
		address:    net.JoinHostPort(values.Hostname, strconv.Itoa(port)),
		clientUser: values.Username,
		serverUser: values.Username,
	}, nil
}

// Connect implements protocol.Protocol: dials the host and performs the
// RFC 1282 handshake (four NUL-terminated fields, then a single NUL byte
// acknowledgement from the server) before the stream becomes transparent.
func (s *Session) Connect(ctx context.Context) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", s.address)
	if err != nil {
		return fmt.Errorf("rlogin: dial %s: %w", s.address, err)
	}

	handshake := fmt.Sprintf("\x00%s\x00%s\x00%s\x00", s.clientUser, s.serverUser, defaultTermSpeed)
	if _, err := conn.Write([]byte(handshake)); err != nil {
		conn.Close()
		return fmt.Errorf("rlogin: send handshake: %w", err)
	}

	ack, err := readByteCtx(ctx, conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("rlogin: read handshake acknowledgement: %w", err)
	}
	if ack != 0x00 {
		conn.Close()
		return fmt.Errorf("rlogin: unexpected handshake acknowledgement byte %#x, want 0x00", ack)
	}

	s.stream = protocol.NewWatchedStream(&s.Lifecycle, conn)
	return nil
}

// readByteCtx reads a single byte, or returns ctx.Err() if ctx is done
// first — net.Dialer.DialContext already gave us a connected socket, but a
// plain net.Conn.Read has no context awareness of its own.
func readByteCtx(ctx context.Context, conn net.Conn) (byte, error) {
	type result struct {
		b   byte
		err error
	}
	done := make(chan result, 1)
	go func() {
		var buf [1]byte
		_, err := conn.Read(buf[:])
		done <- result{buf[0], err}
	}()
	select {
	case res := <-done:
		return res.b, res.err
	case <-ctx.Done():
		conn.Close()
		return 0, ctx.Err()
	}
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

// Resize implements protocol.Protocol. Classic rlogin signals a window
// size change out-of-band (TCP urgent data plus a control message), which
// is not implemented here — a thin client, per the blueprint. Remote
// programs relying on SIGWINCH after an interactive resize won't see it;
// re-attaching (e.g. via tmux/screen) is the usual workaround.
func (s *Session) Resize(width, height int) {}

// Read implements io.Reader by delegating to the underlying connection.
func (s *Session) Read(p []byte) (int, error) { return s.stream.Read(p) }

// Write implements io.Writer by delegating to the underlying connection.
func (s *Session) Write(p []byte) (int, error) { return s.stream.Write(p) }

var _ protocol.TerminalProtocol = (*Session)(nil)
