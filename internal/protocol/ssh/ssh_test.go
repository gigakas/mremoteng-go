<<<<<<< HEAD
package ssh

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	cryptossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// testServer is an in-process SSH server used to exercise the real client.
type testServer struct {
	address  string
	signer   cryptossh.Signer
	knownKey cryptossh.PublicKey
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	signer, err := cryptossh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	config := &cryptossh.ServerConfig{
		PasswordCallback: func(metadata cryptossh.ConnMetadata, password []byte) (*cryptossh.Permissions, error) {
			if string(password) == "correct-horse" {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid password")
=======
package ssh_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	xssh "golang.org/x/crypto/ssh"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/ssh"
)

const (
	testUsername = "alice"
	testPassword = "s3cret"
)

// fakeSSHServer runs a minimal in-process SSH-2 server: authenticates a
// single username/password pair, accepts one session channel, replies to
// pty-req/shell/window-change requests, and echoes whatever it receives on
// the channel back to the client (standing in for an interactive shell).
func fakeSSHServer(t *testing.T) string {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	signer, err := xssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("signer from host key: %v", err)
	}

	config := &xssh.ServerConfig{
		PasswordCallback: func(c xssh.ConnMetadata, password []byte) (*xssh.Permissions, error) {
			if c.User() == testUsername && string(password) == testPassword {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid credentials")
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
		},
	}
	config.AddHostKey(signer)

<<<<<<< HEAD
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go serveTestConn(conn, config)
		}
	}()
	return &testServer{address: listener.Addr().String(), signer: signer, knownKey: signer.PublicKey()}
}

func serveTestConn(conn net.Conn, config *cryptossh.ServerConfig) {
	defer func() { _ = conn.Close() }()
	serverConn, channels, requests, err := cryptossh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	defer func() { _ = serverConn.Close() }()
	go cryptossh.DiscardRequests(requests)
	for newChannel := range channels {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(cryptossh.UnknownChannelType, "unsupported")
			continue
		}
		channel, reqs, err := newChannel.Accept()
		if err != nil {
			continue
		}
		go serveSession(channel, reqs)
	}
}

// serveSession answers pty/shell requests and echoes input, which is enough to
// exercise the interactive client path.
func serveSession(channel cryptossh.Channel, requests <-chan *cryptossh.Request) {
	for request := range requests {
		switch request.Type {
		case "pty-req":
			_ = request.Reply(true, nil)
		case "shell":
			_ = request.Reply(true, nil)
			go func() {
				_, _ = io.Copy(channel, channel)
			}()
		case "window-change":
			_ = request.Reply(true, nil)
		default:
			_ = request.Reply(false, nil)
		}
	}
}

func (s *testServer) target(t *testing.T) Target {
	t.Helper()
	host, portText, err := net.SplitHostPort(s.address)
	if err != nil {
		t.Fatalf("split address: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return Target{Host: host, Port: port, Username: "tester"}
}

func TestProbe_TrustedHostAndPassword_Succeeds(t *testing.T) {
	server := newTestServer(t)
	target := server.target(t)
	knownHosts := filepath.Join(t.TempDir(), "known_hosts")
	line := knownhosts.Line([]string{knownhosts.Normalize(server.address)}, server.knownKey)
	if err := os.WriteFile(knownHosts, []byte(line+"\n"), 0o600); err != nil {
		t.Fatalf("write known_hosts: %v", err)
	}

	result, err := Probe(target, Options{
		Password:        "correct-horse",
		KnownHostsFiles: []string{knownHosts},
		Timeout:         5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !result.HostKeyVerified {
		t.Error("HostKeyVerified = false, want true for a known host")
	}
	if want := cryptossh.FingerprintSHA256(server.knownKey); result.Fingerprint != want {
		t.Errorf("Fingerprint = %q, want %q", result.Fingerprint, want)
	}
	if result.ServerVersion == "" {
		t.Error("ServerVersion is empty")
	}
	if len(result.AuthMethods) == 0 || result.AuthMethods[len(result.AuthMethods)-1] != "password" {
		t.Errorf("AuthMethods = %v, want password last", result.AuthMethods)
	}
}

func TestProbe_UnknownHost_ReportsFingerprint(t *testing.T) {
	server := newTestServer(t)
	_, err := Probe(server.target(t), Options{
		Password:        "correct-horse",
		KnownHostsFiles: []string{filepath.Join(t.TempDir(), "absent_known_hosts")},
		Timeout:         5 * time.Second,
	})
	var unknown *UnknownHostKeyError
	if !errors.As(err, &unknown) {
		t.Fatalf("Probe error = %v, want UnknownHostKeyError", err)
	}
	if want := cryptossh.FingerprintSHA256(server.knownKey); unknown.Fingerprint != want {
		t.Errorf("Fingerprint = %q, want %q", unknown.Fingerprint, want)
	}
}

func TestProbe_AcceptAnyHostKey_SucceedsButUnverified(t *testing.T) {
	server := newTestServer(t)
	result, err := Probe(server.target(t), Options{
		Password:         "correct-horse",
		AcceptAnyHostKey: true,
		Timeout:          5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if result.HostKeyVerified {
		t.Error("HostKeyVerified = true after accepting any key")
	}
}

func TestProbe_WrongPassword_ReturnsAuthenticationError(t *testing.T) {
	server := newTestServer(t)
	_, err := Probe(server.target(t), Options{
		Password:         "wrong",
		AcceptAnyHostKey: true,
		Timeout:          5 * time.Second,
	})
	if err == nil {
		t.Fatal("Probe succeeded with a wrong password")
	}
	var unknown *UnknownHostKeyError
	if errors.As(err, &unknown) {
		t.Fatalf("error = %v, want an authentication failure", err)
	}
}

func TestProbe_NoCredentials_ReturnsErrNoAuthMethod(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	_, err := Probe(Target{Host: "127.0.0.1", Port: 1, Username: "nobody"}, Options{Timeout: time.Second})
	if !errors.Is(err, ErrNoAuthMethod) {
		t.Fatalf("Probe error = %v, want ErrNoAuthMethod", err)
	}
}

func TestOpenShell_PTYAndEcho_TransfersInputAndOutput(t *testing.T) {
	server := newTestServer(t)
	shell, result, err := OpenShell(server.target(t), Options{
		Password:         "correct-horse",
		AcceptAnyHostKey: true,
		Timeout:          5 * time.Second,
	}, TermSize{Cols: 100, Rows: 30})
	if err != nil {
		t.Fatalf("OpenShell: %v", err)
	}
	defer func() {
		if err := shell.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()
	if result.Fingerprint == "" {
		t.Error("result fingerprint is empty")
	}
	if err := shell.Input([]byte("hello\r")); err != nil {
		t.Fatalf("Input: %v", err)
	}
	reader := bufio.NewReader(shell.Output())
	line, err := reader.ReadString('\r')
	if err != nil {
		t.Fatalf("read echoed output: %v", err)
	}
	if line != "hello\r" {
		t.Fatalf("echo = %q, want %q", line, "hello\r")
	}
	if err := shell.Resize(TermSize{Cols: 120, Rows: 40}); err != nil {
		t.Fatalf("Resize: %v", err)
	}
}

// TestProbeAndShell_LocalContainer exercises a real SSH server, for example the
// container started with:
//
//	docker run -d --name ssh-test -p 12222:2222 -e USER_NAME=demo \
//	  -e USER_PASSWORD=demo -e PASSWORD_ACCESS=true \
//	  lscr.io/linuxserver/openssh-server:latest
//	SSH_LOCAL=127.0.0.1:12222 SSH_LOCAL_USER=demo SSH_LOCAL_PASSWORD=demo \
//	  go test ./internal/protocol/ssh/ -run LocalContainer -v
func TestProbeAndShell_LocalContainer(t *testing.T) {
	endpoint := os.Getenv("SSH_LOCAL")
	if endpoint == "" {
		t.Skip("SSH_LOCAL not set; no local SSH container to probe")
	}
	host, portText, err := net.SplitHostPort(endpoint)
	if err != nil {
		t.Fatalf("SSH_LOCAL: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("SSH_LOCAL port: %v", err)
	}
	target := Target{Host: host, Port: port, Username: os.Getenv("SSH_LOCAL_USER")}
	options := Options{
		Password:         os.Getenv("SSH_LOCAL_PASSWORD"),
		AcceptAnyHostKey: true,
		Timeout:          15 * time.Second,
	}

	result, err := Probe(target, options)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if result.ServerVersion == "" || result.Fingerprint == "" {
		t.Errorf("incomplete result: %+v", result)
	}

	shell, _, err := OpenShell(target, options, TermSize{Cols: 100, Rows: 30})
	if err != nil {
		t.Fatalf("OpenShell: %v", err)
	}
	defer func() { _ = shell.Close() }()
	if err := shell.Input([]byte("echo container-ok\n")); err != nil {
		t.Fatalf("Input: %v", err)
	}
	reader := bufio.NewReader(shell.Output())
	deadline := time.Now().Add(20 * time.Second)
	collected := ""
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		collected += line
		if strings.Contains(collected, "container-ok") {
			t.Logf("shell responded; server %s", result.ServerVersion)
			return
		}
		if err != nil {
			break
		}
	}
	t.Fatalf("shell did not echo container-ok; output: %q", collected)
}

func TestOpenShell_UnreachableHost_ReturnsError(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	_, _, err := OpenShell(Target{Host: "127.0.0.1", Port: 1, Username: "tester"}, Options{
		Password:         "secret",
		AcceptAnyHostKey: true,
		Timeout:          500 * time.Millisecond,
	}, TermSize{})
	if err == nil {
		t.Fatal("OpenShell succeeded against a closed port")
	}
}

func TestProbe_UnreachableHost_ReturnsConnectError(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	_, err := Probe(Target{Host: "127.0.0.1", Port: 1, Username: "tester"}, Options{
		Password:         "secret",
		AcceptAnyHostKey: true,
		Timeout:          500 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("Probe succeeded against a closed port")
=======
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		serverConn, chans, reqs, err := xssh.NewServerConn(conn, config)
		if err != nil {
			return
		}
		defer serverConn.Close()
		go xssh.DiscardRequests(reqs)

		for newChannel := range chans {
			if newChannel.ChannelType() != "session" {
				newChannel.Reject(xssh.UnknownChannelType, "unsupported channel type")
				continue
			}
			channel, requests, err := newChannel.Accept()
			if err != nil {
				return
			}
			go func() {
				for req := range requests {
					switch req.Type {
					case "pty-req", "shell", "window-change":
						req.Reply(true, nil)
					default:
						req.Reply(false, nil)
					}
				}
			}()
			go func() {
				defer channel.Close()
				buf := make([]byte, 4096)
				for {
					n, err := channel.Read(buf)
					if n > 0 {
						channel.Write(buf[:n])
					}
					if err != nil {
						return
					}
				}
			}()
		}
	}()

	return ln.Addr().String()
}

func newTestSession(t *testing.T, host string, port int, username, password string) (protocol.Protocol, error) {
	t.Helper()
	info, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	info.Raw.Protocol = connection.ProtocolSSH2
	info.Raw.Hostname = host
	info.Raw.Port = port
	info.Raw.Username = username
	info.Raw.Password = password
	return protocol.Create(info)
}

func splitAddr(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return host, port
}

func TestNew_MissingUsername_ReturnsError(t *testing.T) {
	if _, err := newTestSession(t, "example.invalid", 22, "", "x"); err == nil {
		t.Fatal("expected an error for a missing username")
	}
}

func TestSSH1_IsRegisteredButUnsupported(t *testing.T) {
	info, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	info.Raw.Protocol = connection.ProtocolSSH1
	info.Raw.Hostname = "example.invalid"
	if _, err := protocol.Create(info); err == nil {
		t.Fatal("expected SSH-1 to be rejected as unsupported")
	}
}

func TestSession_ConnectAndShell_EchoesData(t *testing.T) {
	addr := fakeSSHServer(t)
	host, port := splitAddr(t, addr)

	p, err := newTestSession(t, host, port, testUsername, testPassword)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer p.Disconnect()

	term := p.(protocol.TerminalProtocol)
	if _, err := term.Write([]byte("echo hi\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	buf := make([]byte, 64)
	n, err := term.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := string(buf[:n]); got != "echo hi\n" {
		t.Errorf("echoed data = %q, want %q", got, "echo hi\n")
	}

	// Resize must not error against a server that accepts window-change.
	term.Resize(100, 30)
}

func TestSession_Connect_WrongPassword_ReturnsError(t *testing.T) {
	addr := fakeSSHServer(t)
	host, port := splitAddr(t, addr)

	p, err := newTestSession(t, host, port, testUsername, "wrong-password")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.Connect(ctx); err == nil {
		t.Fatal("expected Connect to fail with the wrong password")
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
	}
}
