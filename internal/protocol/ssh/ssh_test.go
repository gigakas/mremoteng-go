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
		},
	}
	config.AddHostKey(signer)

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
	}
}
