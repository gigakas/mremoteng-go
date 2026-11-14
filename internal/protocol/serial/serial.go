// Package serial opens a serial device as a byte stream for the terminal view.
//
// The port is configured raw: no echo, no line discipline and no CR/LF translation,
// so the terminal sees exactly the bytes the device sent. Every setting is explicit,
// including the ones the zero value stands for, because a serial device that looks
// dead is usually a line setting nobody looked at.
package serial

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"golang.org/x/sys/unix"
)

// Defaults every serial device understands.
const (
	DefaultBaud     = 9600
	DefaultDataBits = 8
	DefaultParity   = "none"
	DefaultStopBits = 1
)

// ErrUnknownBaud means the requested speed has no termios constant.
var ErrUnknownBaud = errors.New("serial: unsupported baud rate")

// ErrInvalidConfig means the line settings cannot be expressed on a serial port.
var ErrInvalidConfig = errors.New("serial: invalid line settings")

// Config describes one serial port. The zero value means 9600 8N1.
type Config struct {
	// Device is the port to open, for example /dev/ttyS0 or /dev/ttyUSB0.
	Device string
	// Baud, DataBits, Parity and StopBits are the line settings.
	Baud     int
	DataBits int
	Parity   string
	StopBits int
}

// Normalized fills the zero values with the defaults.
func (c Config) Normalized() Config {
	if c.Baud == 0 {
		c.Baud = DefaultBaud
	}
	if c.DataBits == 0 {
		c.DataBits = DefaultDataBits
	}
	if c.Parity == "" {
		c.Parity = DefaultParity
	}
	if c.StopBits == 0 {
		c.StopBits = DefaultStopBits
	}
	c.Parity = strings.ToLower(c.Parity)
	return c
}

// Validate reports why the settings cannot be used.
func (c Config) Validate() error {
	config := c.Normalized()
	if strings.TrimSpace(config.Device) == "" {
		return fmt.Errorf("%w: the device path is empty", ErrInvalidConfig)
	}
	if _, err := baudConstant(config.Baud); err != nil {
		return err
	}
	if config.DataBits < 5 || config.DataBits > 8 {
		return fmt.Errorf("%w: %d data bits", ErrInvalidConfig, config.DataBits)
	}
	switch config.Parity {
	case "none", "even", "odd", "mark", "space":
	default:
		return fmt.Errorf("%w: parity %q", ErrInvalidConfig, config.Parity)
	}
	if config.StopBits != 1 && config.StopBits != 2 {
		return fmt.Errorf("%w: %d stop bits", ErrInvalidConfig, config.StopBits)
	}
	return nil
}

// Description names the port and its line settings for a status line.
func (c Config) Description() string {
	return strings.TrimSpace(c.Normalized().Device + " " + c.LineDescription())
}

// LineDescription is the conventional short form of the settings, "115200 8N1".
func (c Config) LineDescription() string {
	config := c.Normalized()
	parity := map[string]string{"none": "N", "even": "E", "odd": "O", "mark": "M", "space": "S"}[config.Parity]
	return fmt.Sprintf("%d %d%s%d", config.Baud, config.DataBits, parity, config.StopBits)
}

// Conn is an open serial port; it is a plain byte stream.
type Conn struct {
	file   *os.File
	device string
}

// Open configures and opens the port.
func Open(config Config) (*Conn, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	normalized := config.Normalized()
	// O_NOCTTY: opening a serial port must not become the controlling terminal, or
	// closing it would signal the application.
	file, err := os.OpenFile(normalized.Device, os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, fmt.Errorf("serial: open %s: %w", normalized.Device, err)
	}
	settings, err := Termios(normalized)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := unix.IoctlSetTermios(int(file.Fd()), unix.TCSETS, settings); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("serial: configure %s: %w", normalized.Device, err)
	}
	return &Conn{file: file, device: normalized.Device}, nil
}

// Device is the port this connection opened.
func (c *Conn) Device() string { return c.device }

// Read blocks until the device sends bytes.
func (c *Conn) Read(payload []byte) (int, error) { return c.file.Read(payload) }

// Write sends bytes to the device.
func (c *Conn) Write(payload []byte) (int, error) { return c.file.Write(payload) }

// Close releases the port.
func (c *Conn) Close() error { return c.file.Close() }

// Termios builds the raw line settings for config. It is separate from Open so the
// flag arithmetic can be tested without a device.
func Termios(config Config) (*unix.Termios, error) {
	normalized := config.Normalized()
	baud, err := baudConstant(normalized.Baud)
	if err != nil {
		return nil, err
	}
	settings := &unix.Termios{}
	settings.Cflag = unix.CREAD | unix.CLOCAL
	settings.Cflag |= baud
	settings.Ispeed = baud
	settings.Ospeed = baud
	switch normalized.DataBits {
	case 5:
		settings.Cflag |= unix.CS5
	case 6:
		settings.Cflag |= unix.CS6
	case 7:
		settings.Cflag |= unix.CS7
	case 8:
		settings.Cflag |= unix.CS8
	default:
		return nil, fmt.Errorf("%w: %d data bits", ErrInvalidConfig, normalized.DataBits)
	}
	switch normalized.Parity {
	case "none":
	case "even":
		settings.Cflag |= unix.PARENB
	case "odd":
		settings.Cflag |= unix.PARENB | unix.PARODD
	case "mark":
		settings.Cflag |= unix.PARENB | unix.PARODD | unix.CMSPAR
	case "space":
		settings.Cflag |= unix.PARENB | unix.CMSPAR
	default:
		return nil, fmt.Errorf("%w: parity %q", ErrInvalidConfig, normalized.Parity)
	}
	if normalized.StopBits == 2 {
		settings.Cflag |= unix.CSTOPB
	}
	// VMIN=1 VTIME=0 makes a read block until at least one byte is there, which is
	// what the terminal pump wants.
	settings.Cc[unix.VMIN] = 1
	settings.Cc[unix.VTIME] = 0
	// The line discipline stays out of the way: no echo, no signals, no CR/LF or
	// case translation, no flow control.
	settings.Lflag = 0
	settings.Iflag = 0
	settings.Oflag = 0
	return settings, nil
}

// baudConstant maps a speed to its termios constant.
func baudConstant(baud int) (uint32, error) {
	settings, known := baudConstants[baud]
	if !known {
		return 0, fmt.Errorf("%w: %d", ErrUnknownBaud, baud)
	}
	return settings, nil
}

// BaudRates lists the speeds this build accepts, slowest first.
func BaudRates() []int {
	rates := make([]int, 0, len(baudConstants))
	for rate := range baudConstants {
		rates = append(rates, rate)
	}
	sort.Ints(rates)
	return rates
}

// baudConstants is the speed table. They are plain numbers, so the termios struct can
// be built from them.
var baudConstants = map[int]uint32{
	1200:   unix.B1200,
	2400:   unix.B2400,
	4800:   unix.B4800,
	9600:   unix.B9600,
	19200:  unix.B19200,
	38400:  unix.B38400,
	57600:  unix.B57600,
	115200: unix.B115200,
	230400: unix.B230400,
}
