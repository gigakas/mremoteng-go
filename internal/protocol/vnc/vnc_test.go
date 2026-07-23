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
		if err != nil {
			return
		}
		defer conn.Close()
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
	}
}
