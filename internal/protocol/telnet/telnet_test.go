package telnet_test

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/telnet"
)

const (
	iac  = 255
	will = 251
	wont = 252
	do   = 253
	dont = 254
)

func newTestSession(t *testing.T, host string, port int) (protocol.Protocol, error) {
	t.Helper()
	info, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	info.Raw.Protocol = connection.ProtocolTelnet
	info.Raw.Hostname = host
	info.Raw.Port = port
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

func TestNew_MissingHostname_ReturnsError(t *testing.T) {
	if _, err := newTestSession(t, "", 0); err == nil {
		t.Fatal("expected an error for a missing hostname")
	}
}

// TestSession_Read_StripsNegotiationAndRefusesEveryOption starts a fake
// telnetd that interleaves IAC negotiation with real data, and asserts:
// the client's Read only ever surfaces the data bytes, and the client
// replies WONT/DONT to every DO/WILL it receives (falling back to plain
// NVT mode, which every real telnetd accepts).
func TestSession_Read_StripsNegotiationAndRefusesEveryOption(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	serverGotReplies := make(chan []byte, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// IAC WILL ECHO, "hel", IAC DO SUPPRESS-GO-AHEAD, "lo\n"
		conn.Write([]byte{iac, will, 1})
		conn.Write([]byte("hel"))
		conn.Write([]byte{iac, do, 3})
		conn.Write([]byte("lo\n"))

		reply := make([]byte, 6)
		total := 0
		for total < len(reply) {
			n, err := conn.Read(reply[total:])
			total += n
			if err != nil {
				break
			}
		}
		serverGotReplies <- reply[:total]
	}()

	host, port := splitAddr(t, ln.Addr().String())
	p, err := newTestSession(t, host, port)
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
	buf := make([]byte, 64)
	n, err := readUntil(term, buf, "hello\n")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := string(buf[:n]); got != "hello\n" {
		t.Errorf("data = %q, want %q (negotiation bytes leaked through)", got, "hello\n")
	}

	select {
	case got := <-serverGotReplies:
		want := []byte{iac, dont, 1, iac, wont, 3}
		if string(got) != string(want) {
			t.Errorf("negotiation replies = %v, want %v", got, want)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server never received negotiation replies")
	}
}

// readUntil calls Read repeatedly (Telnet filtering may split one logical
// message across several Read calls around negotiation bytes) until want
// has been fully accumulated.
func readUntil(term protocol.TerminalProtocol, buf []byte, want string) (int, error) {
	total := 0
	for total < len(want) {
		n, err := term.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func TestSession_Write_EscapesLiteralIACByte(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	serverGot := make(chan []byte, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 64)
		n, _ := conn.Read(buf)
		serverGot <- buf[:n]
	}()

	host, port := splitAddr(t, ln.Addr().String())
	p, err := newTestSession(t, host, port)
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
	payload := []byte{'a', iac, 'b'}
	if _, err := term.Write(payload); err != nil {
		t.Fatalf("Write: %v", err)
	}

	select {
	case got := <-serverGot:
		want := []byte{'a', iac, iac, 'b'}
		if string(got) != string(want) {
			t.Errorf("bytes on the wire = %v, want %v (0xFF must be escaped)", got, want)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server never received the write")
	}
}
