<<<<<<< HEAD
// Package vnc implements the RFB (VNC) client used by connection profiles:
// handshake, None and VNC authentication, a 32-bit true-colour framebuffer and
// Raw rectangle updates. Encodings such as Tight/ZRLE and clipboard transfer are
// not implemented yet.
package vnc

import (
	"bufio"
	"crypto/des"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"net"
	"strings"
	"time"
)

// Security types of the RFB protocol.
const (
	securityNone    = 1
	securityVNCAuth = 2
)

// encodingRaw is the only encoding requested by this client.
const encodingRaw = 0

// ErrAuthFailed means the server rejected the credentials.
var ErrAuthFailed = errors.New("vnc: authentication failed")

// ErrUnsupportedSecurity means the server offers no usable security type.
var ErrUnsupportedSecurity = errors.New("vnc: server requires an unsupported security type")

// Config controls one connection attempt. The password is used for VNC auth only
// and is never logged or stored.
type Config struct {
	Password string
	Timeout  time.Duration
}

// Client is a connected RFB session.
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	width  int
	height int
	name   string
	img    *image.RGBA

	closed bool
}

// Dial connects and completes the handshake up to the first framebuffer update
// request. Call ReadMessage to consume updates.
func Dial(address string, config Config) (*Client, error) {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil, fmt.Errorf("vnc: connect: %w", err)
	}
	client := &Client{conn: conn, reader: bufio.NewReader(conn)}
	if err := client.handshake(config.Password); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := client.requestUpdate(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

// Size reports the remote framebuffer size.
func (c *Client) Size() (width, height int) { return c.width, c.height }

// Name is the desktop name announced by the server.
func (c *Client) Name() string { return c.name }

// Image returns the current framebuffer. It is updated in place by ReadMessage.
func (c *Client) Image() *image.RGBA { return c.img }

func (c *Client) handshake(password string) error {
	version := make([]byte, 12)
	if _, err := io.ReadFull(c.reader, version); err != nil {
		return fmt.Errorf("vnc: read server version: %w", err)
	}
	if !strings.HasPrefix(string(version), "RFB ") {
		return errors.New("vnc: not an RFB server")
	}
	if _, err := c.conn.Write([]byte("RFB 003.008\n")); err != nil {
		return fmt.Errorf("vnc: write client version: %w", err)
	}

	count, err := c.reader.ReadByte()
	if err != nil {
		return fmt.Errorf("vnc: read security types: %w", err)
	}
	if count == 0 {
		return c.readSecurityFailure()
	}
	offered := make([]byte, count)
	if _, err := io.ReadFull(c.reader, offered); err != nil {
		return fmt.Errorf("vnc: read security types: %w", err)
	}
	chosen := byte(0)
	for _, candidate := range offered {
		if candidate == securityNone {
			chosen = securityNone
			break
		}
		if candidate == securityVNCAuth {
			chosen = securityVNCAuth
		}
	}
	if chosen == 0 {
		return ErrUnsupportedSecurity
	}
	if _, err := c.conn.Write([]byte{chosen}); err != nil {
		return fmt.Errorf("vnc: choose security: %w", err)
	}
	if chosen == securityVNCAuth {
		if err := c.authenticate(password); err != nil {
			return err
		}
	}
	result := make([]byte, 4)
	if _, err := io.ReadFull(c.reader, result); err != nil {
		return fmt.Errorf("vnc: read security result: %w", err)
	}
	if binary.BigEndian.Uint32(result) != 0 {
		return ErrAuthFailed
	}
	if _, err := c.conn.Write([]byte{1}); err != nil { // shared session
		return fmt.Errorf("vnc: client init: %w", err)
	}

	header := make([]byte, 24)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return fmt.Errorf("vnc: read server init: %w", err)
	}
	c.width = int(binary.BigEndian.Uint16(header[0:2]))
	c.height = int(binary.BigEndian.Uint16(header[2:4]))
	nameLength := int(binary.BigEndian.Uint32(header[20:24]))
	name := make([]byte, nameLength)
	if _, err := io.ReadFull(c.reader, name); err != nil {
		return fmt.Errorf("vnc: read desktop name: %w", err)
	}
	c.name = string(name)
	c.img = image.NewRGBA(image.Rect(0, 0, c.width, c.height))

	// 32bpp true colour with red in the high byte, little-endian pixels.
	pixelFormat := []byte{
		32, 24, 0, 1, // bits-per-pixel, depth, big-endian, true-colour
		0, 255, 0, 255, 0, 255, // max red, green, blue
		16, 8, 0, // shifts
		0, 0, 0, // padding
	}
	if err := c.writeMessage(append([]byte{0, 0, 0, 0}, pixelFormat...)); err != nil {
		return err
	}
	encodings := []byte{2, 0, 0, 0, 0} // one encoding: Raw
	return c.writeMessage(encodings)
}

func (c *Client) readSecurityFailure() error {
	length := make([]byte, 4)
	if _, err := io.ReadFull(c.reader, length); err != nil {
		return ErrAuthFailed
	}
	reason := make([]byte, binary.BigEndian.Uint32(length))
	_, _ = io.ReadFull(c.reader, reason)
	return fmt.Errorf("%w: %s", ErrAuthFailed, strings.TrimSpace(string(reason)))
}

// authenticate answers the VNC DES challenge.
func (c *Client) authenticate(password string) error {
	challenge := make([]byte, 16)
	if _, err := io.ReadFull(c.reader, challenge); err != nil {
		return fmt.Errorf("vnc: read challenge: %w", err)
	}
	key := make([]byte, 8)
	copy(key, []byte(password))
	for i := range key {
		key[i] = reverseBits(key[i])
	}
	block, err := des.NewCipher(key)
	if err != nil {
		return fmt.Errorf("vnc: des: %w", err)
	}
	response := make([]byte, 16)
	block.Encrypt(response[0:8], challenge[0:8])
	block.Encrypt(response[8:16], challenge[8:16])
	if err := c.writeMessage(response); err != nil {
		return err
=======
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
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
	}
	return nil
}

<<<<<<< HEAD
// reverseBits mirrors a byte, as required by the RFB VNC-auth key schedule.
func reverseBits(value byte) byte {
	var result byte
	for i := 0; i < 8; i++ {
		result <<= 1
		result |= value & 1
		value >>= 1
	}
	return result
}

func (c *Client) requestUpdate() error {
	return c.writeMessage([]byte{3, 1, 0, 0, 0, 0, byte(c.width >> 8), byte(c.width), byte(c.height >> 8), byte(c.height)})
}

// ReadMessage reads server messages until a framebuffer update has been applied.
func (c *Client) ReadMessage() error {
	messageType, err := c.reader.ReadByte()
	if err != nil {
		return err
	}
	switch messageType {
	case 0: // FramebufferUpdate
		return c.readUpdate()
	case 1: // SetColourMapEntries
		header := make([]byte, 5)
		if _, err := io.ReadFull(c.reader, header); err != nil {
			return err
		}
		size := int(binary.BigEndian.Uint16(header[3:5]))
		_, err := c.reader.Discard(size * 6)
		return err
	case 2: // Bell
		return nil
	case 3: // ServerCutText
		header := make([]byte, 7)
		if _, err := io.ReadFull(c.reader, header); err != nil {
			return err
		}
		length := int(binary.BigEndian.Uint32(header[3:7]))
		_, err := c.reader.Discard(length)
		return err
	default:
		return fmt.Errorf("vnc: unsupported server message %d", messageType)
	}
}

func (c *Client) readUpdate() error {
	header := make([]byte, 3)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return err
	}
	rectangles := int(binary.BigEndian.Uint16(header[1:3]))
	for i := 0; i < rectangles; i++ {
		rect := make([]byte, 12)
		if _, err := io.ReadFull(c.reader, rect); err != nil {
			return err
		}
		x := int(binary.BigEndian.Uint16(rect[0:2]))
		y := int(binary.BigEndian.Uint16(rect[2:4]))
		width := int(binary.BigEndian.Uint16(rect[4:6]))
		height := int(binary.BigEndian.Uint16(rect[6:8]))
		encoding := int32(binary.BigEndian.Uint32(rect[8:12]))
		if encoding != encodingRaw {
			return fmt.Errorf("vnc: unsupported encoding %d", encoding)
		}
		pixels := make([]byte, width*height*4)
		if _, err := io.ReadFull(c.reader, pixels); err != nil {
			return err
		}
		for row := 0; row < height; row++ {
			for column := 0; column < width; column++ {
				source := (row*width + column) * 4
				target := c.img.PixOffset(x+column, y+row)
				c.img.Pix[target+0] = pixels[source+2]
				c.img.Pix[target+1] = pixels[source+1]
				c.img.Pix[target+2] = pixels[source+0]
				c.img.Pix[target+3] = 0xff
			}
		}
	}
	return c.requestUpdate()
}

// Pointer reports a mouse event in framebuffer coordinates.
func (c *Client) Pointer(x, y int, mask byte) error {
	message := make([]byte, 6)
	message[0] = 5
	message[1] = mask
	binary.BigEndian.PutUint16(message[2:4], uint16(x))
	binary.BigEndian.PutUint16(message[4:6], uint16(y))
	return c.writeMessage(message)
}

// Key reports a keyboard event using an X11 keysym.
func (c *Client) Key(keysym uint32, down bool) error {
	message := make([]byte, 8)
	message[0] = 4
	if down {
		message[1] = 1
	}
	binary.BigEndian.PutUint32(message[4:8], keysym)
	return c.writeMessage(message)
}

func (c *Client) writeMessage(payload []byte) error {
	_, err := c.conn.Write(payload)
	return err
}

// Close ends the session.
func (c *Client) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	return c.conn.Close()
}
=======
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
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
