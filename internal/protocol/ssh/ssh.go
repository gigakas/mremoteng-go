// Package ssh implements protocol.TerminalProtocol for SSH-2, via
// golang.org/x/crypto/ssh. SSH-1 is registered too, but only to return a
// clear "not supported" error: it is a deprecated, cryptographically
// broken protocol with no maintained Go implementation, and the original
// mRemoteNG's SSH-1 support existed only for legacy device compatibility.
package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"

	"golang.org/x/crypto/ssh"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
)

func init() {
	protocol.Register(connection.ProtocolSSH2, New)
	protocol.Register(connection.ProtocolSSH1, newSSH1Unsupported)
}

func newSSH1Unsupported(*connection.ConnectionInfo, connection.ConnectionValues) (protocol.Protocol, error) {
	return nil, fmt.Errorf("ssh: SSH-1 is deprecated and insecure and is not implemented; use SSH-2")
}

// defaultRows/defaultCols size the PTY requested at Connect time, before
// the terminal emulator widget (Phase 3) has a chance to call Resize with
// the view's actual size.
const (
	defaultRows = 24
	defaultCols = 80
)

// Session is an SSH-2 client session backed by a single shell channel with
// a PTY attached.
//
// Known v1 limitations, both recorded as pending actions in the stage
// audit rather than silently accepted:
//   - Authentication is password-only (no public key, no keyboard-
//     interactive) — the connection model has no key-file field yet.
//   - The host key is accepted unconditionally (ssh.InsecureIgnoreHostKey).
//     There is no known_hosts store or a UI to ask the user to confirm a
//     new host key (that UI is Phase 3), so this is a real MITM exposure
//     until one exists.
type Session struct {
	protocol.Lifecycle
	address  string
	username string
	password string

	session *ssh.Session
	stream  *protocol.WatchedStream
}

// New builds a Session for info. It implements protocol.Constructor.
func New(_ *connection.ConnectionInfo, values connection.ConnectionValues) (protocol.Protocol, error) {
	if values.Hostname == "" {
		return nil, fmt.Errorf("ssh: hostname is required")
	}
	if values.Username == "" {
		return nil, fmt.Errorf("ssh: username is required")
	}
	port := values.Port
	if port <= 0 {
		port = connection.DefaultPort(connection.ProtocolSSH2)
	}
	return &Session{
		address:  net.JoinHostPort(values.Hostname, strconv.Itoa(port)),
		username: values.Username,
		password: values.Password,
	}, nil
}

// Connect implements protocol.Protocol: dials the host, performs the SSH
// handshake, opens a session channel with a PTY attached, and starts an
// interactive shell.
func (s *Session) Connect(ctx context.Context) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", s.address)
	if err != nil {
		return fmt.Errorf("ssh: dial %s: %w", s.address, err)
	}

	config := &ssh.ClientConfig{
		User:            s.username,
		Auth:            []ssh.AuthMethod{ssh.Password(s.password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // v1 gap, see Session doc comment
	}

	client, err := handshake(ctx, conn, s.address, config)
	if err != nil {
		conn.Close()
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return fmt.Errorf("ssh: open session: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", defaultRows, defaultCols, modes); err != nil {
		session.Close()
		client.Close()
		return fmt.Errorf("ssh: request pty: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return fmt.Errorf("ssh: stdin pipe: %w", err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		return fmt.Errorf("ssh: stdout pipe: %w", err)
	}

	if err := session.Shell(); err != nil {
		session.Close()
		client.Close()
		return fmt.Errorf("ssh: start shell: %w", err)
	}

	s.session = session
	rwc := &sessionStream{stdout: stdout, stdin: stdin, session: session, client: client}
	s.stream = protocol.NewWatchedStream(&s.Lifecycle, rwc)
	return nil
}

// handshake runs the SSH handshake in a goroutine so it can be abandoned
// when ctx is done — golang.org/x/crypto/ssh has no native context
// support, and the handshake can otherwise block indefinitely on a
// non-responsive or malicious server.
func handshake(ctx context.Context, conn net.Conn, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	type result struct {
		client *ssh.Client
		err    error
	}
	done := make(chan result, 1)
	go func() {
		sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
		if err != nil {
			done <- result{nil, err}
			return
		}
		done <- result{ssh.NewClient(sshConn, chans, reqs), nil}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			return nil, fmt.Errorf("ssh: handshake with %s: %w", addr, res.err)
		}
		return res.client, nil
	case <-ctx.Done():
		conn.Close()
		return nil, ctx.Err()
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

// Resize implements protocol.Protocol by sending an SSH window-change
// request. width/height are character cells (columns/rows), per
// protocol.TerminalProtocol.
func (s *Session) Resize(width, height int) {
	if s.session == nil {
		return
	}
	if err := s.session.WindowChange(height, width); err != nil {
		s.FireError(fmt.Errorf("ssh: window change: %w", err))
	}
}

// Read implements io.Reader over the shell's stdout.
func (s *Session) Read(p []byte) (int, error) { return s.stream.Read(p) }

// Write implements io.Writer over the shell's stdin.
func (s *Session) Write(p []byte) (int, error) { return s.stream.Write(p) }

// sessionStream adapts an ssh.Session's separate stdin/stdout pipes into a
// single io.ReadWriteCloser, closing both the session and the underlying
// client connection together.
type sessionStream struct {
	stdout  io.Reader
	stdin   io.WriteCloser
	session *ssh.Session
	client  *ssh.Client
}

func (s *sessionStream) Read(p []byte) (int, error)  { return s.stdout.Read(p) }
func (s *sessionStream) Write(p []byte) (int, error) { return s.stdin.Write(p) }

func (s *sessionStream) Close() error {
	sessErr := s.session.Close()
	clientErr := s.client.Close()
	if sessErr != nil && sessErr != io.EOF {
		return sessErr
	}
	return clientErr
}

var _ protocol.TerminalProtocol = (*Session)(nil)
