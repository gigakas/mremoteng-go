package vnc

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

// fakeServer implements just enough RFB to exercise the client.
type fakeServer struct {
	address string
	pointer chan [2]int
	keys    chan uint32
}

func newFakeServer(t *testing.T, requireAuth bool, password string) *fakeServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	server := &fakeServer{address: listener.Addr().String(), pointer: make(chan [2]int, 4), keys: make(chan uint32, 4)}
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		server.serve(conn, requireAuth, password)
	}()
	return server
}

func (s *fakeServer) serve(conn net.Conn, requireAuth bool, password string) {
	reader := bufio.NewReader(conn)
	_, _ = conn.Write([]byte("RFB 003.008\n"))
	version := make([]byte, 12)
	if _, err := io.ReadFull(reader, version); err != nil {
		return
	}
	security := byte(1)
	types := []byte{securityNone}
	if requireAuth {
		security = securityVNCAuth
		types = []byte{securityVNCAuth}
	}
	_, _ = conn.Write([]byte{byte(len(types))})
	_, _ = conn.Write(types)
	chosen, err := reader.ReadByte()
	if err != nil || chosen != security {
		return
	}
	if requireAuth {
		challenge := []byte("0123456789abcdef")
		_, _ = conn.Write(challenge)
		response := make([]byte, 16)
		if _, err := io.ReadFull(reader, response); err != nil {
			return
		}
		if string(response) == string(make([]byte, 16)) {
			_, _ = conn.Write([]byte{0, 0, 0, 1})
			return
		}
	}
	_, _ = conn.Write([]byte{0, 0, 0, 0})        // security result OK
	if _, err := reader.ReadByte(); err != nil { // client init
		return
	}

	init := make([]byte, 24)
	binary.BigEndian.PutUint16(init[0:2], 2) // width
	binary.BigEndian.PutUint16(init[2:4], 2) // height
	copy(init[4:20], []byte{32, 24, 0, 1, 0, 255, 0, 255, 0, 255, 16, 8, 0, 0, 0, 0})
	binary.BigEndian.PutUint32(init[20:24], uint32(len("fake")))
	_, _ = conn.Write(init)
	_, _ = conn.Write([]byte("fake"))

	pixelFormat := make([]byte, 20)
	if _, err := io.ReadFull(reader, pixelFormat); err != nil {
		return
	}
	encodings := make([]byte, 5)
	if _, err := io.ReadFull(reader, encodings); err != nil {
		return
	}
	request := make([]byte, 10)
	if _, err := io.ReadFull(reader, request); err != nil {
		return
	}

	// One Raw update: two pixels, red and green.
	update := []byte{0, 0, 0, 1}
	rect := make([]byte, 12)
	binary.BigEndian.PutUint16(rect[4:6], 2)
	binary.BigEndian.PutUint16(rect[6:8], 2)
	binary.BigEndian.PutUint32(rect[8:12], encodingRaw)
	update = append(update, rect...)
	pixels := make([]byte, 2*2*4)
	// Pixel 0: red as BGRX little-endian layout used by the client.
	pixels[0], pixels[1], pixels[2] = 0x00, 0x00, 0xff
	pixels[3] = 0
	pixels[4], pixels[5], pixels[6] = 0x00, 0xff, 0x00
	pixels[7] = 0
	update = append(update, pixels...)
	_, _ = conn.Write(update)

	for {
		messageType, err := reader.ReadByte()
		if err != nil {
			return
		}
		switch messageType {
		case 3: // FramebufferUpdateRequest
			if _, err := reader.Discard(9); err != nil {
				return
			}
		case 5: // PointerEvent
			payload := make([]byte, 5)
			if _, err := io.ReadFull(reader, payload); err != nil {
				return
			}
			s.pointer <- [2]int{int(binary.BigEndian.Uint16(payload[1:3])), int(binary.BigEndian.Uint16(payload[3:5]))}
		case 4: // KeyEvent
			payload := make([]byte, 7)
			if _, err := io.ReadFull(reader, payload); err != nil {
				return
			}
			s.keys <- binary.BigEndian.Uint32(payload[3:7])
		default:
			return
		}
	}
}

func TestDial_NoneSecurity_ReadsFramebuffer(t *testing.T) {
	server := newFakeServer(t, false, "")
	client, err := Dial(server.address, Config{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer func() { _ = client.Close() }()
	if width, height := client.Size(); width != 2 || height != 2 {
		t.Fatalf("size = %dx%d, want 2x2", width, height)
	}
	if client.Name() != "fake" {
		t.Errorf("name = %q", client.Name())
	}
	if err := client.ReadMessage(); err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	image := client.Image()
	if got := image.RGBAAt(0, 0); got.R != 0xff || got.G != 0 {
		t.Errorf("pixel 0 = %+v, want red", got)
	}
	if got := image.RGBAAt(1, 0); got.G != 0xff || got.R != 0 {
		t.Errorf("pixel 1 = %+v, want green", got)
	}
}

func TestDial_VNCAuth_AcceptsPassword(t *testing.T) {
	server := newFakeServer(t, true, "secret")
	client, err := Dial(server.address, Config{Password: "secret", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Dial with VNC auth: %v", err)
	}
	_ = client.Close()
}

func TestPointerAndKey_AreForwarded(t *testing.T) {
	server := newFakeServer(t, false, "")
	client, err := Dial(server.address, Config{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer func() { _ = client.Close() }()
	if err := client.Pointer(1, 1, 1); err != nil {
		t.Fatalf("Pointer: %v", err)
	}
	if err := client.Key(0xff0d, true); err != nil {
		t.Fatalf("Key: %v", err)
	}
	select {
	case position := <-server.pointer:
		if position != [2]int{1, 1} {
			t.Errorf("pointer = %v", position)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pointer event was not received")
	}
	select {
	case keysym := <-server.keys:
		if keysym != 0xff0d {
			t.Errorf("keysym = %x", keysym)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("key event was not received")
	}
}

func TestDial_UnreachableServer_ReturnsError(t *testing.T) {
	if _, err := Dial("127.0.0.1:1", Config{Timeout: 300 * time.Millisecond}); err == nil {
		t.Fatal("Dial succeeded against a closed port")
	}
}

func TestReverseBits_MirrorsByte(t *testing.T) {
	if got := reverseBits(0b10000000); got != 0b00000001 {
		t.Errorf("reverseBits = %080b", got)
	}
}

func TestDial_UnsupportedSecurity_ReturnsError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = conn.Write([]byte("RFB 003.008\n"))
		_, _ = conn.Write([]byte{1, 19}) // one type: TLS (unsupported)
	}()
	_, err = Dial(listener.Addr().String(), Config{Timeout: 2 * time.Second})
	if !errors.Is(err, ErrUnsupportedSecurity) {
		t.Fatalf("Dial error = %v, want ErrUnsupportedSecurity", err)
	}
}
