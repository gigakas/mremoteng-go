package telnet

import (
	"io"
	"net"
	"testing"
	"time"
)

func startServer(t *testing.T, handler func(net.Conn)) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		handler(conn)
	}()
	return listener.Addr().String()
}

func TestRead_StripsTelnetNegotiation(t *testing.T) {
	address := startServer(t, func(conn net.Conn) {
		_, _ = conn.Write([]byte{iac, will, 1, 'o', 'k', '\n'})
		time.Sleep(50 * time.Millisecond)
	})
	client, err := Dial(address, Config{Mode: ModeTelnet, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer func() { _ = client.Close() }()
	buffer := make([]byte, 16)
	n, err := client.Read(buffer)
	if err != nil && err != io.EOF {
		t.Fatalf("Read: %v", err)
	}
	if string(buffer[:n]) != "ok\n" {
		t.Fatalf("read = %q, want ok\\n", buffer[:n])
	}
}

func TestRead_KeepsEscapedIAC(t *testing.T) {
	address := startServer(t, func(conn net.Conn) {
		_, _ = conn.Write([]byte{iac, iac, 'x'})
		time.Sleep(50 * time.Millisecond)
	})
	client, err := Dial(address, Config{Mode: ModeTelnet, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer func() { _ = client.Close() }()
	buffer := make([]byte, 8)
	n, _ := client.Read(buffer)
	if n != 2 || buffer[0] != iac || buffer[1] != 'x' {
		t.Fatalf("read = %v", buffer[:n])
	}
}

func TestRlogin_HandshakeCarriesUserAndTerminal(t *testing.T) {
	received := make(chan string, 1)
	address := startServer(t, func(conn net.Conn) {
		buffer := make([]byte, 64)
		n, _ := conn.Read(buffer)
		received <- string(buffer[:n])
	})
	client, err := Dial(address, Config{Mode: ModeRlogin, Username: "demo", Terminal: "xterm", Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer func() { _ = client.Close() }()
	select {
	case payload := <-received:
		if len(payload) < 4 || payload[0] != 0 || payload[1] != 0 || payload[2] != 0 || payload[3] != 0 {
			t.Fatalf("handshake = %q, want four zero bytes first", payload)
		}
		if !contains(payload, "demo\x00") || !contains(payload, "xterm/") {
			t.Fatalf("handshake = %q, want user and terminal", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server received nothing")
	}
}

func contains(haystack string, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func TestWrite_RawPassesBytesUnchanged(t *testing.T) {
	received := make(chan []byte, 1)
	address := startServer(t, func(conn net.Conn) {
		buffer := make([]byte, 8)
		n, _ := conn.Read(buffer)
		received <- buffer[:n]
	})
	client, err := Dial(address, Config{Mode: ModeRaw, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer func() { _ = client.Close() }()
	if _, err := client.Write([]byte{iac, 'a'}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	select {
	case payload := <-received:
		if len(payload) != 2 || payload[0] != iac {
			t.Fatalf("raw write = %v, want the byte untouched", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server received nothing")
	}
}

func TestParseMode_RejectsUnknown(t *testing.T) {
	if _, err := ParseMode("SSH"); err == nil {
		t.Fatal("ParseMode accepted SSH")
	}
}
