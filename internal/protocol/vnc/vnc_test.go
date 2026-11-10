<<<<<<< HEAD
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
=======
package vnc_test

import (
	"context"
	"encoding/binary"
	"image/color"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/vnc"
)

const (
	fbWidth  = 4
	fbHeight = 4
)

// fakeRFBServer runs a minimal in-process RFB 3.8 server: no
// authentication, a fbWidth x fbHeight desktop, and — once it sees the
// client's first FramebufferUpdateRequest — sends a single 2x2 raw-encoded
// red rectangle at (0,0), then discards any further client messages
// (including the client's own periodic re-requests) until the connection
// closes.
func fakeRFBServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		conn, err := ln.Accept()
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
		if err != nil {
			return
		}
		defer conn.Close()
<<<<<<< HEAD
		_, _ = conn.Write([]byte("RFB 003.008\n"))
		_, _ = conn.Write([]byte{1, 19}) // one type: TLS (unsupported)
	}()
	_, err = Dial(listener.Addr().String(), Config{Timeout: 2 * time.Second})
	if !errors.Is(err, ErrUnsupportedSecurity) {
		t.Fatalf("Dial error = %v, want ErrUnsupportedSecurity", err)
=======
		serveRFB(conn)
	}()
	return ln.Addr().String()
}

func serveRFB(conn net.Conn) {
	// 7.1.1 ProtocolVersion.
	if _, err := conn.Write([]byte("RFB 003.008\n")); err != nil {
		return
	}
	var clientVersion [12]byte
	if _, err := io.ReadFull(conn, clientVersion[:]); err != nil {
		return
	}

	// 7.1.2 Security handshake: offer only "None" (type 1).
	if _, err := conn.Write([]byte{1, 1}); err != nil {
		return
	}
	var chosen [1]byte
	if _, err := io.ReadFull(conn, chosen[:]); err != nil {
		return
	}

	// 7.1.3 SecurityResult: OK.
	if err := binary.Write(conn, binary.BigEndian, uint32(0)); err != nil {
		return
	}

	// 7.3.1 ClientInit.
	var clientInit [1]byte
	if _, err := io.ReadFull(conn, clientInit[:]); err != nil {
		return
	}

	// 7.3.2 ServerInit: width, height, 16-byte pixel format, name.
	if err := binary.Write(conn, binary.BigEndian, uint16(fbWidth)); err != nil {
		return
	}
	if err := binary.Write(conn, binary.BigEndian, uint16(fbHeight)); err != nil {
		return
	}
	pixelFormat := [16]byte{32, 24, 0, 1, 0, 255, 0, 255, 0, 255, 16, 8, 0, 0, 0, 0}
	if _, err := conn.Write(pixelFormat[:]); err != nil {
		return
	}
	name := "test"
	if err := binary.Write(conn, binary.BigEndian, uint32(len(name))); err != nil {
		return
	}
	if _, err := conn.Write([]byte(name)); err != nil {
		return
	}

	sentUpdate := false
	for {
		var msgType [1]byte
		if _, err := io.ReadFull(conn, msgType[:]); err != nil {
			return
		}
		switch msgType[0] {
		case 0: // SetPixelFormat: 3 bytes padding + 16 bytes format
			var rest [19]byte
			if _, err := io.ReadFull(conn, rest[:]); err != nil {
				return
			}
		case 2: // SetEncodings: 1 byte padding + uint16 count + count*int32
			var header [3]byte
			if _, err := io.ReadFull(conn, header[:]); err != nil {
				return
			}
			count := binary.BigEndian.Uint16(header[1:3])
			buf := make([]byte, int(count)*4)
			if _, err := io.ReadFull(conn, buf); err != nil {
				return
			}
		case 3: // FramebufferUpdateRequest: 9 more bytes
			var rest [9]byte
			if _, err := io.ReadFull(conn, rest[:]); err != nil {
				return
			}
			if !sentUpdate {
				sentUpdate = true
				if err := sendRedRectangle(conn); err != nil {
					return
				}
			}
		default:
			// Not exercised by this test (KeyEvent, PointerEvent, ...).
			return
		}
	}
}

// sendRedRectangle sends one FramebufferUpdateMessage with a single 2x2
// raw-encoded fully-red rectangle at (0,0), matching the fixed 32bpp
// truecolor pixel format the vnc package always requests
// (RedShift=16, GreenShift=8, BlueShift=0, little-endian).
func sendRedRectangle(conn net.Conn) error {
	if _, err := conn.Write([]byte{0, 0}); err != nil { // type=0, padding
		return err
	}
	if err := binary.Write(conn, binary.BigEndian, uint16(1)); err != nil { // 1 rectangle
		return err
	}
	rect := struct{ X, Y, W, H uint16 }{0, 0, 2, 2}
	for _, v := range []uint16{rect.X, rect.Y, rect.W, rect.H} {
		if err := binary.Write(conn, binary.BigEndian, v); err != nil {
			return err
		}
	}
	if err := binary.Write(conn, binary.BigEndian, int32(0)); err != nil { // Raw encoding
		return err
	}
	// 4 pixels, little-endian uint32 each: B, G, R, pad — red is
	// 0x00FF0000 given RedShift=16.
	pixel := []byte{0x00, 0x00, 0xFF, 0x00}
	for i := 0; i < 4; i++ {
		if _, err := conn.Write(pixel); err != nil {
			return err
		}
	}
	return nil
}

func newTestSession(t *testing.T, host string, port int) (protocol.Protocol, error) {
	t.Helper()
	info, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	info.Raw.Protocol = connection.ProtocolVNC
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

func TestSession_Connect_ReceivesDecodedFramebuffer(t *testing.T) {
	addr := fakeRFBServer(t)
	host, port := splitAddr(t, addr)

	p, err := newTestSession(t, host, port)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer p.Disconnect()

	fbp := p.(protocol.FramebufferProtocol)

	select {
	case img := <-fbp.Frames():
		bounds := img.Bounds()
		if bounds.Dx() != fbWidth || bounds.Dy() != fbHeight {
			t.Fatalf("frame size = %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), fbWidth, fbHeight)
		}
		r, g, b, a := img.At(0, 0).RGBA()
		got := color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
		want := color.RGBA{R: 255, G: 0, B: 0, A: 255}
		if got != want {
			t.Errorf("pixel (0,0) = %+v, want %+v", got, want)
		}
		// A pixel outside the updated 2x2 rectangle should still be at
		// the image's zero value (fully transparent black), proving the
		// backend didn't paint the whole framebuffer, only the rectangle
		// the server actually sent.
		if r2, g2, b2, a2 := img.At(3, 3).RGBA(); r2 != 0 || g2 != 0 || b2 != 0 || a2 != 0 {
			t.Errorf("pixel (3,3) = (%d,%d,%d,%d), want all zero (untouched)", r2, g2, b2, a2)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no frame received")
	}

	// SendKey/SendPointer must not fail against a live session.
	fbp.SendKey(0x61, true)
	fbp.SendKey(0x61, false)
	fbp.SendPointer(protocol.PointerButtonLeft, 1, 1)
}

func TestSession_Disconnect_ClosesFramesChannel(t *testing.T) {
	addr := fakeRFBServer(t)
	host, port := splitAddr(t, addr)

	p, err := newTestSession(t, host, port)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	fbp := p.(protocol.FramebufferProtocol)
	closed := make(chan struct{})
	p.OnClose(func() { close(closed) })

	if err := p.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}

	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("OnClose was not fired by Disconnect")
	}

	select {
	case _, ok := <-fbp.Frames():
		if ok {
			t.Error("Frames channel produced a value after Disconnect instead of closing")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Frames channel was not closed after Disconnect")
	}

	// Idempotent.
	if err := p.Disconnect(); err != nil {
		t.Fatalf("second Disconnect: %v", err)
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
	}
}
