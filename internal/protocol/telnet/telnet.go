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
