<<<<<<< HEAD
// Package ssh provides the SSH connectivity used by connection profiles:
// host-key verification, authentication and a session probe that proves a
// channel can be opened. Rendering an interactive terminal is a later stage.
package ssh

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	cryptossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Target is the endpoint to reach.
type Target struct {
	Host     string
	Port     int
	Username string
}

// Options controls one connection attempt. Nothing here is persisted.
type Options struct {
	// Password is used for password authentication; empty means no password.
	Password string
	// KeyPaths overrides the default ~/.ssh key files.
	KeyPaths []string
	// KnownHostsFiles overrides ~/.ssh/known_hosts and known_hosts2.
	KnownHostsFiles []string
	// AcceptAnyHostKey skips verification. Only set it after the user has
	// explicitly approved the fingerprint of an unknown host.
	AcceptAnyHostKey bool
	// Timeout bounds the whole attempt; zero selects ten seconds.
	Timeout time.Duration
}

// Result describes a successful probe.
type Result struct {
	ServerVersion   string
	Fingerprint     string
	HostKeyVerified bool
	AuthMethods     []string
}

// UnknownHostKeyError reports a host that is not present in known_hosts.
type UnknownHostKeyError struct {
	HostPort    string
	Fingerprint string
	KnownHosts  string
}

func (e *UnknownHostKeyError) Error() string {
	return fmt.Sprintf("ssh: host key for %s is not trusted (fingerprint %s)", e.HostPort, e.Fingerprint)
}

// ErrNoAuthMethod means no agent, key file or password was available.
var ErrNoAuthMethod = errors.New("ssh: no authentication method available; no agent, key file or password")

const defaultTimeout = 10 * time.Second

// TermSize is a terminal size in character cells.
type TermSize struct {
	Cols int
	Rows int
}

// Shell is an interactive SSH session with a remote pseudo-terminal.
type Shell struct {
	session *cryptossh.Session
	input   io.WriteCloser
	output  io.Reader
	client  *cryptossh.Client
}

// OpenShell authenticates, requests a pseudo-terminal and starts the remote
// login shell. The caller owns the session and must Close it.
func OpenShell(target Target, options Options, size TermSize) (*Shell, Result, error) {
	if size.Cols < 1 {
		size.Cols = 80
	}
	if size.Rows < 1 {
		size.Rows = 24
	}
	client, result, err := connect(target, options)
	if err != nil {
		return nil, Result{}, err
	}
	session, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		return nil, Result{}, fmt.Errorf("ssh: open shell session: %w", err)
	}
	modes := cryptossh.TerminalModes{
		cryptossh.ECHO:          1,
		cryptossh.TTY_OP_ISPEED: 14400,
		cryptossh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", size.Rows, size.Cols, modes); err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, Result{}, fmt.Errorf("ssh: request pseudo-terminal: %w", err)
	}
	input, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, Result{}, fmt.Errorf("ssh: stdin pipe: %w", err)
	}
	output, err := session.StdoutPipe()
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, Result{}, fmt.Errorf("ssh: stdout pipe: %w", err)
	}
	if err := session.Shell(); err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, Result{}, fmt.Errorf("ssh: start shell: %w", err)
	}
	return &Shell{session: session, input: input, output: output, client: client}, result, nil
}

// Output streams remote terminal output.
func (s *Shell) Output() io.Reader { return s.output }

// Input sends keystrokes to the remote shell.
func (s *Shell) Input(p []byte) error {
	_, err := s.input.Write(p)
	return err
}

// Resize tells the remote pseudo-terminal that the view changed size.
func (s *Shell) Resize(size TermSize) error {
	if size.Cols < 1 || size.Rows < 1 {
		return nil
	}
	return s.session.WindowChange(size.Rows, size.Cols)
}

// Wait blocks until the remote shell exits.
func (s *Shell) Wait() error { return s.session.Wait() }

// Close ends the session and the underlying connection.
func (s *Shell) Close() error {
	sessionErr := s.session.Close()
	clientErr := s.client.Close()
	if sessionErr != nil {
		return sessionErr
=======
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
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
	}
	return clientErr
}

<<<<<<< HEAD
// Probe connects, authenticates and opens one session channel, then closes it.
// It performs no command execution and stores nothing.
func Probe(target Target, options Options) (Result, error) {
	client, result, err := connect(target, options)
	if err != nil {
		return Result{}, err
	}
	address := net.JoinHostPort(target.Host, fmt.Sprintf("%d", target.Port))
	defer func() {
		// Closing after a successful probe; a failure here cannot be acted on.
		_ = client.Close()
	}()

	session, err := client.NewSession()
	if err != nil {
		return Result{}, fmt.Errorf("ssh: open session on %s: %w", address, err)
	}
	// A session that opens and closes proves authentication and channel support
	// without running anything on the server.
	_ = session.Close()
	return result, nil
}

// connect authenticates and returns a client plus the metadata both Probe and
// OpenShell report. The caller closes the client.
func connect(target Target, options Options) (*cryptossh.Client, Result, error) {
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	methods, authNames, cleanup, err := authMethods(options)
	if err != nil {
		return nil, Result{}, err
	}
	defer cleanup()
	if len(methods) == 0 {
		return nil, Result{}, ErrNoAuthMethod
	}
	verify, hostKeyVerified, err := hostKeyVerification(options)
	if err != nil {
		return nil, Result{}, err
	}

	address := net.JoinHostPort(target.Host, fmt.Sprintf("%d", target.Port))
	var fingerprint string
	config := &cryptossh.ClientConfig{
		User: target.Username,
		Auth: methods,
		HostKeyCallback: func(hostname string, remote net.Addr, key cryptossh.PublicKey) error {
			// Capture first so unknown-host errors can report the fingerprint.
			fingerprint = cryptossh.FingerprintSHA256(key)
			if verify == nil {
				return nil
			}
			return verify(hostname, remote, key)
		},
		Timeout: timeout,
	}

	client, err := dial(address, config)
	if err != nil {
		var unknown *UnknownHostKeyError
		if errors.As(err, &unknown) {
			return nil, Result{}, err
		}
		return nil, Result{}, fmt.Errorf("ssh: connect to %s: %w", address, err)
	}
	return client, Result{
		ServerVersion:   string(client.ServerVersion()),
		Fingerprint:     fingerprint,
		HostKeyVerified: hostKeyVerified,
		AuthMethods:     authNames,
	}, nil
}

func dial(address string, config *cryptossh.ClientConfig) (*cryptossh.Client, error) {
	conn, err := net.DialTimeout("tcp", address, config.Timeout)
	if err != nil {
		return nil, err
	}
	clientConn, channels, requests, err := cryptossh.NewClientConn(conn, address, config)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return cryptossh.NewClient(clientConn, channels, requests), nil
}

// hostKeyVerification returns the host-key callback and whether a successful
// connection is verified against known_hosts. A nil callback means the caller
// explicitly accepted any key; without that flag a missing known_hosts file
// makes every host unknown, so the callback reports an UnknownHostKeyError
// instead of silently trusting it.
func hostKeyVerification(options Options) (cryptossh.HostKeyCallback, bool, error) {
	if options.AcceptAnyHostKey {
		return nil, false, nil
	}
	files := options.KnownHostsFiles
	if len(files) == 0 {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, false, fmt.Errorf("ssh: locate home directory: %w", err)
		}
		files = []string{filepath.Join(home, ".ssh", "known_hosts")}
	}
	existing := make([]string, 0, len(files))
	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			existing = append(existing, file)
		}
	}
	if len(existing) == 0 {
		missing := strings.Join(files, ", ")
		return func(hostname string, _ net.Addr, key cryptossh.PublicKey) error {
			return &UnknownHostKeyError{
				HostPort:    hostname,
				Fingerprint: cryptossh.FingerprintSHA256(key),
				KnownHosts:  missing,
			}
		}, false, nil
	}
	verify, err := knownhosts.New(existing...)
	if err != nil {
		return nil, false, fmt.Errorf("ssh: load known_hosts: %w", err)
	}
	path := strings.Join(existing, ", ")
	return func(hostname string, remote net.Addr, key cryptossh.PublicKey) error {
		err := verify(hostname, remote, key)
		if err == nil {
			return nil
		}
		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
			return &UnknownHostKeyError{HostPort: hostname, Fingerprint: cryptossh.FingerprintSHA256(key), KnownHosts: path}
		}
		return fmt.Errorf("ssh: host key verification for %s failed: %w", hostname, err)
	}, true, nil
}

// authMethods collects agent, key-file and password authentication in the order
// users expect: agent and keys first, password last as a fallback.
func authMethods(options Options) ([]cryptossh.AuthMethod, []string, func(), error) {
	var methods []cryptossh.AuthMethod
	var names []string
	cleanup := func() {}

	if socket := os.Getenv("SSH_AUTH_SOCK"); socket != "" {
		conn, err := net.Dial("unix", socket)
		if err == nil {
			agentClient := agent.NewClient(conn)
			methods = append(methods, cryptossh.PublicKeysCallback(agentClient.Signers))
			names = append(names, "agent")
			cleanup = func() { _ = conn.Close() }
		}
	}

	for _, path := range keyPaths(options) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		signer, err := cryptossh.ParsePrivateKey(data)
		if err != nil {
			continue
		}
		methods = append(methods, cryptossh.PublicKeys(signer))
		names = append(names, filepath.Base(path))
	}

	if options.Password != "" {
		methods = append(methods, cryptossh.Password(options.Password))
		names = append(names, "password")
	}
	return methods, names, cleanup, nil
}

func keyPaths(options Options) []string {
	if len(options.KeyPaths) > 0 {
		return options.KeyPaths
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
		filepath.Join(home, ".ssh", "id_rsa"),
	}
}
=======
var _ protocol.TerminalProtocol = (*Session)(nil)
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
