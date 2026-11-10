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
	}
	return nil
}

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
