package rlogin_test

import (
	"bufio"
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/rlogin"
)

// fakeRlogind runs a minimal RFC 1282 server: reads the four NUL-delimited
// handshake fields, replies with a single NUL ack, then echoes data until
// the connection closes.
func fakeRlogind(t *testing.T) string {
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

		r := bufio.NewReader(conn)
		for i := 0; i < 4; i++ {
			if _, err := r.ReadBytes(0x00); err != nil {
				return
			}
		}
		if _, err := conn.Write([]byte{0x00}); err != nil {
			return
		}

		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				conn.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()
	return ln.Addr().String()
}

func newTestSession(t *testing.T, host string, port int, username string) (protocol.Protocol, error) {
	t.Helper()
	info, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	info.Raw.Protocol = connection.ProtocolRlogin
	info.Raw.Hostname = host
	info.Raw.Port = port
	info.Raw.Username = username
	return protocol.Create(info)
}

func TestNew_MissingUsername_ReturnsError(t *testing.T) {
	if _, err := newTestSession(t, "example.invalid", 513, ""); err == nil {
		t.Fatal("expected an error for a missing username")
	}
}

func TestSession_Connect_PerformsHandshakeThenEchoesData(t *testing.T) {
	addr := fakeRlogind(t)
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	p, err := newTestSession(t, host, port, "alice")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer p.Disconnect()

	term := p.(protocol.TerminalProtocol)
	if _, err := term.Write([]byte("ls\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	buf := make([]byte, 64)
	n, err := term.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := string(buf[:n]); got != "ls\n" {
		t.Errorf("echoed data = %q, want %q", got, "ls\n")
	}
}

func TestSession_Connect_BadAckByte_ReturnsError(t *testing.T) {
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
		r := bufio.NewReader(conn)
		for i := 0; i < 4; i++ {
			if _, err := r.ReadBytes(0x00); err != nil {
				return
			}
		}
		conn.Write([]byte{0x01}) // wrong ack byte
	}()

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	p, err := newTestSession(t, host, port, "alice")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.Connect(ctx); err == nil {
		t.Fatal("expected Connect to fail on a bad acknowledgement byte")
	}
}
