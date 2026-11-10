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
	}
	return clientErr
}

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
