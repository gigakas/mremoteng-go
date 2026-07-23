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
		},
	}
	config.AddHostKey(signer)

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
	}
}
