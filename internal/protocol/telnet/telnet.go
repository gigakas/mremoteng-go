// Package telnet implements protocol.TerminalProtocol for the Telnet
// protocol (RFC 854/855): a thin client that refuses every option the
// remote server negotiates (falling back to plain NVT mode, which every
// telnetd supports) rather than implementing the many optional Telnet
// extensions (echo, line mode, NAWS window-size, terminal type, ...).
package telnet

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
)

func init() {
	protocol.Register(connection.ProtocolTelnet, New)
}

// Telnet command bytes (RFC 854).
const (
	cmdSE   = 240
	cmdSB   = 250
	cmdWILL = 251
	cmdWONT = 252
	cmdDO   = 253
	cmdDONT = 254
	cmdIAC  = 255
)

// Session is a Telnet client session.
type Session struct {
	protocol.Lifecycle
	address string
	stream  *protocol.WatchedStream
	br      *bufio.Reader
}

// New builds a Session for info. It implements protocol.Constructor.
func New(_ *connection.ConnectionInfo, values connection.ConnectionValues) (protocol.Protocol, error) {
	if values.Hostname == "" {
		return nil, fmt.Errorf("telnet: hostname is required")
	}
	port := values.Port
	if port <= 0 {
		port = connection.DefaultPort(connection.ProtocolTelnet)
	}
	return &Session{address: net.JoinHostPort(values.Hostname, strconv.Itoa(port))}, nil
}

// Connect implements protocol.Protocol.
func (s *Session) Connect(ctx context.Context) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", s.address)
	if err != nil {
		return fmt.Errorf("telnet: dial %s: %w", s.address, err)
	}
	s.stream = protocol.NewWatchedStream(&s.Lifecycle, conn)
	s.br = bufio.NewReader(s.stream)
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

// Resize implements protocol.Protocol. NAWS (the Telnet option that
// reports window size) is one of the options this thin client refuses, so
// there is no wire mechanism to report a resize; this is a no-op.
func (s *Session) Resize(width, height int) {}

// Write implements io.Writer, escaping any literal 0xFF byte in p as
// IAC IAC (0xFF 0xFF) per RFC 854, so the remote side doesn't mistake user
// data for the start of a command.
func (s *Session) Write(p []byte) (int, error) {
	hasIAC := false
	for _, b := range p {
		if b == cmdIAC {
			hasIAC = true
			break
		}
	}
	if !hasIAC {
		_, err := s.stream.Write(p)
		return len(p), err
	}

	escaped := make([]byte, 0, len(p)+4)
	for _, b := range p {
		escaped = append(escaped, b)
		if b == cmdIAC {
			escaped = append(escaped, cmdIAC)
		}
	}
	if _, err := s.stream.Write(escaped); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Read implements io.Reader, filtering Telnet protocol bytes (option
// negotiation, subnegotiation blocks) out of the stream and returning only
// application data. It responds to every DO/WILL with WONT/DONT inline, as
// it goes.
func (s *Session) Read(p []byte) (n int, err error) {
	for n < len(p) {
		b, rerr := s.br.ReadByte()
		if rerr != nil {
			if n > 0 {
				return n, nil
			}
			return 0, rerr
		}

		if b != cmdIAC {
			p[n] = b
			n++
		} else {
			data, isData, herr := s.handleCommand()
			if herr != nil {
				if n > 0 {
					return n, nil
				}
				return 0, herr
			}
			if isData {
				p[n] = data
				n++
			}
		}

		if s.br.Buffered() == 0 {
			break
		}
	}
	return n, nil
}

// handleCommand processes the bytes following an IAC (0xFF) already
// consumed from the stream. It returns (0xFF, true, nil) for an escaped
// literal 0xFF (IAC IAC — application data), or (0, false, nil) once a
// negotiation command or subnegotiation block has been fully consumed and,
// where applicable, responded to.
func (s *Session) handleCommand() (byte, bool, error) {
	cmd, err := s.br.ReadByte()
	if err != nil {
		return 0, false, err
	}

	switch cmd {
	case cmdIAC:
		return cmdIAC, true, nil

	case cmdWILL, cmdWONT, cmdDO, cmdDONT:
		option, err := s.br.ReadByte()
		if err != nil {
			return 0, false, err
		}
		return 0, false, s.respondNegotiation(cmd, option)

	case cmdSB:
		return 0, false, s.skipSubnegotiation()

	default:
		// Other commands (NOP, AYT, GA, ...) take no option byte and need
		// no response.
		return 0, false, nil
	}
}

// respondNegotiation replies WONT to every DO and DONT to every WILL,
// keeping the session in plain NVT mode. WONT/DONT from the server require
// no reply.
func (s *Session) respondNegotiation(cmd, option byte) error {
	var reply byte
	switch cmd {
	case cmdDO:
		reply = cmdWONT
	case cmdWILL:
		reply = cmdDONT
	default:
		return nil
	}
	_, err := s.stream.Write([]byte{cmdIAC, reply, option})
	return err
}

// skipSubnegotiation consumes bytes up to and including the terminating
// IAC SE, ignoring the content — this client offers nothing that would
// make a subnegotiation payload meaningful.
func (s *Session) skipSubnegotiation() error {
	for {
		b, err := s.br.ReadByte()
		if err != nil {
			return err
		}
		if b != cmdIAC {
			continue
		}
		b2, err := s.br.ReadByte()
		if err != nil {
			return err
		}
		if b2 == cmdSE {
			return nil
		}
		// Any other IAC <x> inside SB: keep scanning.
	}
}

var _ protocol.TerminalProtocol = (*Session)(nil)
