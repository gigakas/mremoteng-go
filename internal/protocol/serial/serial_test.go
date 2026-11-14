package serial

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// ptyPair creates a pseudo terminal and returns its slave path, so the port can be
// exercised without any serial hardware.
func ptyPair(t *testing.T) (string, *os.File) {
	t.Helper()
	master, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatalf("open /dev/ptmx: %v", err)
	}
	number, err := unix.IoctlGetInt(master, unix.TIOCGPTN)
	if err != nil {
		t.Fatalf("read the pty number: %v", err)
	}
	if err := unix.IoctlSetPointerInt(master, unix.TIOCSPTLCK, 0); err != nil {
		t.Fatalf("unlock the pty: %v", err)
	}
	slave := fmt.Sprintf("/dev/pts/%d", number)
	file := os.NewFile(uintptr(master), "ptmx")
	t.Cleanup(func() { _ = file.Close() })
	return slave, file
}

func TestTermios_DefaultsToRaw8N1(t *testing.T) {
	settings, err := Termios(Config{Device: "/dev/ttyS0"})
	if err != nil {
		t.Fatalf("Termios: %v", err)
	}
	if settings.Lflag != 0 || settings.Iflag != 0 || settings.Oflag != 0 {
		t.Errorf("lflag=%#x iflag=%#x oflag=%#x, want a raw port", settings.Lflag, settings.Iflag, settings.Oflag)
	}
	if settings.Cflag&unix.CSIZE != unix.CS8 {
		t.Errorf("cflag = %#x, want 8 data bits", settings.Cflag)
	}
	if settings.Cflag&unix.PARENB != 0 {
		t.Error("default settings enabled parity")
	}
	if settings.Cflag&unix.CSTOPB != 0 {
		t.Error("default settings asked for two stop bits")
	}
	if settings.Ispeed != unix.B9600 || settings.Ospeed != unix.B9600 {
		t.Errorf("speed = %d/%d, want 9600", settings.Ispeed, settings.Ospeed)
	}
	if settings.Cc[unix.VMIN] != 1 || settings.Cc[unix.VTIME] != 0 {
		t.Errorf("VMIN=%d VTIME=%d, want a blocking read", settings.Cc[unix.VMIN], settings.Cc[unix.VTIME])
	}
}

func TestTermios_MapsParityStopBitsAndDataBits(t *testing.T) {
	cases := []struct {
		name    string
		config  Config
		check   func(*unix.Termios) bool
		message string
	}{
		{"even", Config{Device: "/dev/ttyS0", Parity: "even"}, func(term *unix.Termios) bool {
			return term.Cflag&unix.PARENB != 0 && term.Cflag&unix.PARODD == 0
		}, "even parity means PARENB without PARODD"},
		{"odd", Config{Device: "/dev/ttyS0", Parity: "odd"}, func(term *unix.Termios) bool {
			return term.Cflag&unix.PARENB != 0 && term.Cflag&unix.PARODD != 0
		}, "odd parity means PARENB and PARODD"},
		{"two stop bits", Config{Device: "/dev/ttyS0", StopBits: 2}, func(term *unix.Termios) bool {
			return term.Cflag&unix.CSTOPB != 0
		}, "two stop bits mean CSTOPB"},
		{"seven bits", Config{Device: "/dev/ttyS0", DataBits: 7}, func(term *unix.Termios) bool {
			return term.Cflag&unix.CSIZE == unix.CS7
		}, "seven data bits mean CS7"},
		{"115200", Config{Device: "/dev/ttyS0", Baud: 115200}, func(term *unix.Termios) bool {
			return term.Ispeed == unix.B115200
		}, "the speed reaches the termios struct"},
	}
	for _, test := range cases {
		settings, err := Termios(test.config)
		if err != nil {
			t.Errorf("%s: %v", test.name, err)
			continue
		}
		if !test.check(settings) {
			t.Errorf("%s: %s", test.name, test.message)
		}
	}
}

func TestConfig_Validate_RejectsWhatThePortCannotDo(t *testing.T) {
	if err := (Config{Device: "/dev/ttyS0"}).Validate(); err != nil {
		t.Errorf("the defaults were refused: %v", err)
	}
	cases := map[string]Config{
		"no device":      {},
		"unknown speed":  {Device: "/dev/ttyS0", Baud: 1234},
		"nine data bits": {Device: "/dev/ttyS0", DataBits: 9},
		"bad parity":     {Device: "/dev/ttyS0", Parity: "sometimes"},
		"three stops":    {Device: "/dev/ttyS0", StopBits: 3},
	}
	for name, config := range cases {
		if err := config.Validate(); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if !errors.Is((Config{Device: "/dev/ttyS0", Baud: 1}).Validate(), ErrUnknownBaud) {
		t.Error("an unknown speed does not report ErrUnknownBaud")
	}
}

func TestConfig_Description_NamesTheLineSettings(t *testing.T) {
	config := Config{Device: "/dev/ttyUSB0", Baud: 115200, DataBits: 8, Parity: "even", StopBits: 2}
	if got := config.Description(); got != "/dev/ttyUSB0 115200 8E2" {
		t.Errorf("description = %q, want the conventional short form", got)
	}
	if got := config.LineDescription(); got != "115200 8E2" {
		t.Errorf("line description = %q, want the settings without the device", got)
	}
}

// TestOpen_ExchangesBytesOverAPTYPair is the end-to-end check: the port really reads
// and writes, with the line discipline out of the way.
func TestOpen_ExchangesBytesOverAPTYPair(t *testing.T) {
	slave, master := ptyPair(t)
	conn, err := Open(Config{Device: slave})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if conn.Device() != slave {
		t.Errorf("device = %q, want %q", conn.Device(), slave)
	}

	if _, err := master.WriteString("hello"); err != nil {
		t.Fatalf("write to the port: %v", err)
	}
	received := make([]byte, 16)
	_ = master.SetReadDeadline(time.Now().Add(2 * time.Second))
	count, err := conn.Read(received)
	if err != nil {
		t.Fatalf("read from the port: %v", err)
	}
	if string(received[:count]) != "hello" {
		t.Errorf("received %q, want the bytes sent", received[:count])
	}

	if _, err := conn.Write([]byte("world")); err != nil {
		t.Fatalf("write to the device: %v", err)
	}
	_ = master.SetReadDeadline(time.Now().Add(2 * time.Second))
	back := make([]byte, 16)
	count, err = master.Read(back)
	if err != nil {
		t.Fatalf("read from the master: %v", err)
	}
	if string(back[:count]) != "world" {
		t.Errorf("master received %q, want the bytes written", back[:count])
	}
}

func TestOpen_MissingDevice_ReturnsError(t *testing.T) {
	if _, err := Open(Config{Device: "/dev/does-not-exist"}); err == nil {
		t.Fatal("opening a missing device was accepted")
	}
}

func TestBaudRates_AreSortedAndIncludeTheDefaults(t *testing.T) {
	rates := BaudRates()
	if len(rates) < 5 {
		t.Fatalf("rates = %v, want the common speeds", rates)
	}
	for index := 1; index < len(rates); index++ {
		if rates[index-1] >= rates[index] {
			t.Fatalf("rates are not sorted: %v", rates)
		}
	}
	found := false
	for _, rate := range rates {
		if rate == DefaultBaud {
			found = true
		}
	}
	if !found {
		t.Errorf("rates = %v, want the default %d", rates, DefaultBaud)
	}
}

func TestTermios_MapsFlowControl(t *testing.T) {
	hardware, err := Termios(Config{Device: "/dev/ttyS0", Flow: FlowRTSCTS})
	if err != nil {
		t.Fatalf("Termios: %v", err)
	}
	if hardware.Cflag&unix.CRTSCTS == 0 {
		t.Error("rtscts did not set CRTSCTS")
	}
	if hardware.Iflag&(unix.IXON|unix.IXOFF) != 0 {
		t.Error("rtscts also enabled the software handshake")
	}

	software, err := Termios(Config{Device: "/dev/ttyS0", Flow: FlowXONXOFF})
	if err != nil {
		t.Fatalf("Termios: %v", err)
	}
	if software.Iflag&unix.IXON == 0 || software.Iflag&unix.IXOFF == 0 {
		t.Error("xonxoff did not set IXON and IXOFF")
	}
	if software.Cflag&unix.CRTSCTS != 0 {
		t.Error("xonxoff also enabled the hardware handshake")
	}

	plain, err := Termios(Config{Device: "/dev/ttyS0"})
	if err != nil {
		t.Fatalf("Termios: %v", err)
	}
	if plain.Iflag != 0 || plain.Cflag&unix.CRTSCTS != 0 {
		t.Errorf("the default enabled a handshake: iflag=%#x cflag=%#x", plain.Iflag, plain.Cflag)
	}
	if (Config{Device: "/dev/ttyS0", Flow: "sometimes"}).Validate() == nil {
		t.Error("an unknown handshake was accepted")
	}
	if got := (Config{Device: "/dev/ttyS0", Flow: FlowRTSCTS}).LineDescription(); !strings.Contains(got, "rtscts") {
		t.Errorf("description = %q, want the handshake named", got)
	}
	controls := FlowControls()
	if len(controls) != 3 || controls[0] != FlowNone {
		t.Errorf("FlowControls = %v, want none first", controls)
	}
}
