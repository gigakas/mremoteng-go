<<<<<<< HEAD
// Package telnet implements the byte-stream protocols used by the terminal
// widget: Telnet with minimal option negotiation, RAW TCP and rlogin.
package telnet

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

// Mode selects the wire protocol.
type Mode string

const (
	// ModeTelnet performs IAC option negotiation and strips it from the stream.
	ModeTelnet Mode = "TELNET"
	// ModeRaw is a plain TCP byte stream.
	ModeRaw Mode = "RAW"
	// ModeRlogin performs the RFC 1282 handshake.
	ModeRlogin Mode = "RLOGIN"
)

// Telnet commands.
const (
	iac  = 255
	dont = 254
	do   = 253
	wont = 252
	will = 251
)

// Config controls one connection.
type Config struct {
	Mode     Mode
	Username string
	Terminal string
	Timeout  time.Duration
}

// Conn is a connected byte stream. Telnet negotiation is handled internally so
// callers only see application data.
type Conn struct {
	conn net.Conn
	mode Mode
}

// Dial opens the connection and completes any handshake.
func Dial(address string, config Config) (*Conn, error) {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil, fmt.Errorf("telnet: connect: %w", err)
	}
	result := &Conn{conn: conn, mode: config.Mode}
	if result.mode == ModeRlogin {
		if err := result.rloginHandshake(config); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	return result, nil
}

// rloginHandshake sends the RFC 1282 preamble.
func (c *Conn) rloginHandshake(config Config) error {
	terminal := config.Terminal
	if terminal == "" {
		terminal = "xterm"
	}
	payload := make([]byte, 0, 64)
	payload = append(payload, 0, 0, 0, 0)
	payload = append(payload, []byte(config.Username+"\x00")...)
	payload = append(payload, []byte(terminal+"/"+"38400\x00")...)
	_, err := c.conn.Write(payload)
	return err
}

// Read returns application data, consuming Telnet negotiation in RAW-agnostic
// mode. Non-Telnet modes read straight from the socket.
func (c *Conn) Read(buffer []byte) (int, error) {
	if c.mode != ModeTelnet {
		return c.conn.Read(buffer)
	}
	written := 0
	for written == 0 {
		chunk := make([]byte, len(buffer))
		n, err := c.conn.Read(chunk)
		if n == 0 {
			if err != nil {
				return 0, err
			}
			continue
		}
		for i := 0; i < n; i++ {
			if chunk[i] != iac {
				if written < len(buffer) {
					buffer[written] = chunk[i]
					written++
				}
				continue
			}
			// Negotiation or escaped 0xFF.
			if i+1 >= n {
				break
			}
			command := chunk[i+1]
			switch command {
			case iac:
				if written < len(buffer) {
					buffer[written] = iac
					written++
				}
				i++
			case will, wont, do, dont:
				if i+2 < n {
					c.refuse(command, chunk[i+2])
					i += 2
				}
			default:
				i++
			}
		}
		if written > 0 || err != nil {
			return written, err
		}
	}
	return written, nil
}

// refuse declines every option, which is enough for line-oriented servers.
func (c *Conn) refuse(command, option byte) {
	reply := byte(wont)
	if command == do || command == dont {
		reply = dont
	}
	_, _ = c.conn.Write([]byte{iac, reply, option})
}

// Write sends application bytes.
func (c *Conn) Write(payload []byte) (int, error) {
	escaped := make([]byte, 0, len(payload))
	for _, value := range payload {
		escaped = append(escaped, value)
		if c.mode == ModeTelnet && value == iac {
			escaped = append(escaped, iac)
		}
	}
	return c.conn.Write(escaped)
}

// Close ends the connection.
func (c *Conn) Close() error { return c.conn.Close() }

// ErrUnsupportedMode is returned for an unknown protocol mode.
var ErrUnsupportedMode = errors.New("telnet: unsupported mode")

// ParseMode validates a protocol name.
func ParseMode(raw string) (Mode, error) {
	switch Mode(raw) {
	case ModeTelnet, ModeRaw, ModeRlogin:
		return Mode(raw), nil
	}
	return "", ErrUnsupportedMode
}

var _ io.ReadWriteCloser = (*Conn)(nil)
=======
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
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
