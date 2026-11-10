package raw_test

import (
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/raw"
)

// echoServer starts a local TCP listener that echoes back whatever it
// receives, closing the connection once it sees "bye\n". It returns the
// listener's address and a cleanup func.
func echoServer(t *testing.T) string {
	t.Helper()
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
		defer conn.Close()
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				conn.Write(buf[:n])
				if strings.Contains(string(buf[:n]), "bye\n") {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	return ln.Addr().String()
}

func hostPort(t *testing.T, addr string) (string, int) {
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

func TestNew_MissingHostname_ReturnsError(t *testing.T) {
	if _, err := raw_New(t, "", 0); err == nil {
		t.Fatal("expected an error for a missing hostname")
	}
}

func raw_New(t *testing.T, host string, port int) (protocol.Protocol, error) {
	t.Helper()
	info, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	info.Raw.Protocol = connection.ProtocolRAW
	info.Raw.Hostname = host
	info.Raw.Port = port
	return protocol.Create(info)
}

func TestSession_ConnectReadWriteDisconnect_EchoesData(t *testing.T) {
	addr := echoServer(t)
	host, port := hostPort(t, addr)

	p, err := raw_New(t, host, port)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	term := p.(protocol.TerminalProtocol)

	closed := make(chan struct{})
	term.OnClose(func() { close(closed) })

	if _, err := term.Write([]byte("hello\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	buf := make([]byte, 64)
	n, err := term.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := string(buf[:n]); got != "hello\n" {
		t.Errorf("echoed data = %q, want %q", got, "hello\n")
	}

	if _, err := term.Write([]byte("bye\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// Drain the echo of "bye\n" before disconnecting.
	if _, err := term.Read(buf); err != nil && err != io.EOF {
		t.Fatalf("Read (drain): %v", err)
	}

	if err := p.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}

	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("OnClose was not fired by Disconnect")
	}

	// Idempotent.
	if err := p.Disconnect(); err != nil {
		t.Fatalf("second Disconnect: %v", err)
	}
}

func TestSession_Connect_RefusedConnection_ReturnsError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close() // free the port immediately so the connect is refused

	host, port := hostPort(t, addr)
	p, err := raw_New(t, host, port)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.Connect(ctx); err == nil {
		t.Fatal("expected Connect to fail against a closed port")
	}
}
