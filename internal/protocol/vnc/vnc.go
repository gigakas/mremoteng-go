// Package vnc implements protocol.FramebufferProtocol for VNC (RFB,
// RFC 6143), built on github.com/mitchellh/go-vnc rather than a native
// reimplementation — the blueprint's explicit preference for this stage.
// The library is unmaintained (last commit 2015) but implements the core
// RFB 3.8 handshake, VNC password authentication, and raw-encoded
// framebuffer updates correctly; it has seen years of real-world use via
// HashiCorp Packer's VNC boot-command feature. Only RawEncoding ships with
// it — CopyRect/Hextile/Tight would need custom Encoding implementations
// added on top if performance over slow links becomes a problem (v2
// backlog, per the blueprint's "fill gaps... if needed").
package vnc

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"net"
	"strconv"
	"sync"
	"time"

	govnc "github.com/mitchellh/go-vnc"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
)

func init() {
	protocol.Register(connection.ProtocolVNC, New)
}

// keepAliveInterval bounds how long it can take to notice a dead
// connection that isn't actively pushing framebuffer updates: go-vnc's
// mainLoop has no disconnect signal beyond silently no longer sending on
// ServerMessageCh (confirmed by reading its source — it never closes the
// channel), so periodically re-requesting an incremental update is what
// surfaces a write error and lets Session detect the session has died.
const keepAliveInterval = 10 * time.Second

// vncPixelFormat is fixed to 32bpp truecolor with Max=255 on every
// channel, set explicitly right after connecting instead of using
// whatever the server defaults to. This makes decoding
// straightforward — every Color component is already 0-255, no per-server
// scaling by RedMax/GreenMax/BlueMax needed.
var vncPixelFormat = govnc.PixelFormat{
	BPP: 32, Depth: 24, TrueColor: true,
	RedMax: 255, GreenMax: 255, BlueMax: 255,
	RedShift: 16, GreenShift: 8, BlueShift: 0,
}

// Session is a VNC client session.
type Session struct {
	protocol.Lifecycle
	address  string
	password string

	client      *govnc.ClientConn
	serverMsgCh chan govnc.ServerMessage
	frames      chan image.Image
	done        chan struct{}
	closeOnce   sync.Once

	fbMu sync.Mutex
	fb   *image.RGBA
}

// New builds a Session for info. It implements protocol.Constructor.
func New(_ *connection.ConnectionInfo, values connection.ConnectionValues) (protocol.Protocol, error) {
	if values.Hostname == "" {
		return nil, fmt.Errorf("vnc: hostname is required")
	}
	port := values.Port
	if port <= 0 {
		port = connection.DefaultPort(connection.ProtocolVNC)
	}
	return &Session{
		address:  net.JoinHostPort(values.Hostname, strconv.Itoa(port)),
		password: values.Password,
		done:     make(chan struct{}),
	}, nil
}

// Connect implements protocol.Protocol: dials the host, performs the RFB
// handshake (preferring VNC password authentication when a password is
// set, falling back to no-auth), fixes the pixel format, and starts the
// background loop that turns FramebufferUpdate messages into images on
// Frames().
func (s *Session) Connect(ctx context.Context) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", s.address)
	if err != nil {
		return fmt.Errorf("vnc: dial %s: %w", s.address, err)
	}

	var auth []govnc.ClientAuth
	if s.password != "" {
		auth = append(auth, &govnc.PasswordAuth{Password: s.password})
	}
	auth = append(auth, new(govnc.ClientAuthNone))

	msgCh := make(chan govnc.ServerMessage, 16)
	client, err := connectVNC(ctx, conn, &govnc.ClientConfig{Auth: auth, ServerMessageCh: msgCh})
	if err != nil {
		conn.Close()
		return err
	}

	if err := client.SetPixelFormat(&vncPixelFormat); err != nil {
		client.Close()
		return fmt.Errorf("vnc: set pixel format: %w", err)
	}
	if err := client.SetEncodings([]govnc.Encoding{new(govnc.RawEncoding)}); err != nil {
		client.Close()
		return fmt.Errorf("vnc: set encodings: %w", err)
	}

	s.client = client
	s.serverMsgCh = msgCh
	s.frames = make(chan image.Image, 4)
	s.fb = image.NewRGBA(image.Rect(0, 0, int(client.FrameBufferWidth), int(client.FrameBufferHeight)))

	go s.readLoop()

	if err := client.FramebufferUpdateRequest(false, 0, 0, client.FrameBufferWidth, client.FrameBufferHeight); err != nil {
		s.Disconnect()
		return fmt.Errorf("vnc: initial framebuffer request: %w", err)
	}
	return nil
}

// connectVNC runs the RFB handshake (govnc.Client, which is synchronous
// and has no context support of its own) in a goroutine so it can be
// abandoned when ctx is done, mirroring the same pattern used by the ssh
// backend for the same reason.
func connectVNC(ctx context.Context, conn net.Conn, cfg *govnc.ClientConfig) (*govnc.ClientConn, error) {
	type result struct {
		client *govnc.ClientConn
		err    error
	}
	done := make(chan result, 1)
	go func() {
		client, err := govnc.Client(conn, cfg)
		done <- result{client, err}
	}()
	select {
	case res := <-done:
		if res.err != nil {
			return nil, fmt.Errorf("vnc: handshake with %s: %w", conn.RemoteAddr(), res.err)
		}
		return res.client, nil
	case <-ctx.Done():
		conn.Close()
		return nil, ctx.Err()
	}
}

// readLoop applies incoming FramebufferUpdate messages to the persistent
// framebuffer and publishes a copy on Frames, re-requesting the next
// incremental update after each one. It also fires a keepalive request on
// a timer so a dead connection is noticed even when the server has gone
// silent rather than sent an update (see keepAliveInterval).
func (s *Session) readLoop() {
	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()
	defer close(s.frames)

	requestNext := func() error {
		return s.client.FramebufferUpdateRequest(true, 0, 0, s.client.FrameBufferWidth, s.client.FrameBufferHeight)
	}

	for {
		select {
		case msg := <-s.serverMsgCh:
			update, ok := msg.(*govnc.FramebufferUpdateMessage)
			if !ok {
				continue // Bell / ServerCutText / SetColorMapEntries: no consumer yet
			}
			img := s.applyUpdate(update)
			select {
			case s.frames <- img:
			default:
				// Consumer isn't keeping up; drop the frame rather than
				// block the VNC session (a later update supersedes it).
			}
			if err := requestNext(); err != nil {
				s.dead(err)
				return
			}

		case <-ticker.C:
			if err := requestNext(); err != nil {
				s.dead(err)
				return
			}

		case <-s.done:
			return
		}
	}
}

// applyUpdate composites update's rectangles onto the persistent
// framebuffer and returns an independent copy of the result.
func (s *Session) applyUpdate(update *govnc.FramebufferUpdateMessage) image.Image {
	s.fbMu.Lock()
	defer s.fbMu.Unlock()

	for _, rect := range update.Rectangles {
		raw, ok := rect.Enc.(*govnc.RawEncoding)
		if !ok {
			continue // only RawEncoding was ever advertised via SetEncodings
		}
		for y := 0; y < int(rect.Height); y++ {
			for x := 0; x < int(rect.Width); x++ {
				c := raw.Colors[y*int(rect.Width)+x]
				s.fb.SetRGBA(int(rect.X)+x, int(rect.Y)+y, color.RGBA{
					R: uint8(c.R), G: uint8(c.G), B: uint8(c.B), A: 0xFF,
				})
			}
		}
	}

	cp := image.NewRGBA(s.fb.Rect)
	copy(cp.Pix, s.fb.Pix)
	return cp
}

func (s *Session) dead(err error) {
	if err != nil {
		s.FireError(fmt.Errorf("vnc: connection lost: %w", err))
	}
	s.FireClose()
}

// Disconnect implements protocol.Protocol.
func (s *Session) Disconnect() error {
	s.closeOnce.Do(func() {
		close(s.done)
		if s.client != nil {
			s.client.Close()
		}
	})
	s.FireClose()
	return nil
}

// Focus implements protocol.Protocol; the widget that renders Frames owns
// input focus, so this is a no-op.
func (s *Session) Focus() {}

// Resize implements protocol.Protocol as a no-op: VNC's framebuffer size
// is set by the server at connect time (RFB has a "DesktopSize"
// pseudo-encoding for server-driven resize, not requested here since it
// isn't in the encodings advertised via SetEncodings).
func (s *Session) Resize(width, height int) {}

// Frames implements protocol.FramebufferProtocol.
func (s *Session) Frames() <-chan image.Image { return s.frames }

// SendKey implements protocol.FramebufferProtocol.
func (s *Session) SendKey(keysym uint32, down bool) {
	if s.client == nil {
		return
	}
	if err := s.client.KeyEvent(keysym, down); err != nil {
		s.FireError(fmt.Errorf("vnc: send key event: %w", err))
	}
}

// SendPointer implements protocol.FramebufferProtocol.
func (s *Session) SendPointer(buttons protocol.PointerButtons, x, y int) {
	if s.client == nil {
		return
	}
	if err := s.client.PointerEvent(toVNCButtonMask(buttons), uint16(x), uint16(y)); err != nil {
		s.FireError(fmt.Errorf("vnc: send pointer event: %w", err))
	}
}

func toVNCButtonMask(b protocol.PointerButtons) govnc.ButtonMask {
	var m govnc.ButtonMask
	if b&protocol.PointerButtonLeft != 0 {
		m |= govnc.ButtonLeft
	}
	if b&protocol.PointerButtonMiddle != 0 {
		m |= govnc.ButtonMiddle
	}
	if b&protocol.PointerButtonRight != 0 {
		m |= govnc.ButtonRight
	}
	if b&protocol.PointerButtonWheelUp != 0 {
		m |= govnc.Button4
	}
	if b&protocol.PointerButtonWheelDown != 0 {
		m |= govnc.Button5
	}
	return m
}

var _ protocol.FramebufferProtocol = (*Session)(nil)
