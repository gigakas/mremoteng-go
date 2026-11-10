package ui

import (
	"fyne.io/fyne/v2"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// Preference keys for the remembered window geometry.
const (
	prefWidth  = "window.width"
	prefHeight = "window.height"
	prefX      = "window.x"
	prefY      = "window.y"
)

// minWindowSize keeps a restored window usable even if the preferences are
// stale (for example after a monitor change).
var minWindowSize = fyne.NewSize(720, 480)

// RestoreWindowSize returns the remembered size, clamped to a usable minimum.
func RestoreWindowSize(prefs fyne.Preferences, fallback fyne.Size) fyne.Size {
	width := prefs.FloatWithFallback(prefWidth, float64(fallback.Width))
	height := prefs.FloatWithFallback(prefHeight, float64(fallback.Height))
	size := fyne.NewSize(float32(width), float32(height))
	if size.Width < minWindowSize.Width {
		size.Width = minWindowSize.Width
	}
	if size.Height < minWindowSize.Height {
		size.Height = minWindowSize.Height
	}
	return size
}

// SaveWindowSize stores the current window size.
func SaveWindowSize(prefs fyne.Preferences, size fyne.Size) {
	prefs.SetFloat(prefWidth, float64(size.Width))
	prefs.SetFloat(prefHeight, float64(size.Height))
}

// SaveWindowPosition stores the window's screen position when X11 is available.
func SaveWindowPosition(win fyne.Window, prefs fyne.Preferences) {
	conn, window, err := x11Conn(win)
	if err != nil {
		return
	}
	defer conn.Close()
	setup := xproto.Setup(conn)
	root := setup.DefaultScreen(conn).Root
	position, err := xproto.TranslateCoordinates(conn, window, root, 0, 0).Reply()
	if err != nil {
		return
	}
	prefs.SetInt(prefX, int(position.DstX))
	prefs.SetInt(prefY, int(position.DstY))
}

// RestoreWindowPosition moves the window back to its remembered position.
func RestoreWindowPosition(win fyne.Window, prefs fyne.Preferences) {
	x := prefs.IntWithFallback(prefX, -1)
	y := prefs.IntWithFallback(prefY, -1)
	if x < 0 || y < 0 {
		return
	}
	conn, window, err := x11Conn(win)
	if err != nil {
		return
	}
	defer conn.Close()
	mask := uint16(xproto.ConfigWindowX | xproto.ConfigWindowY)
	values := []uint32{uint32(int16(x)), uint32(int16(y))}
	_ = xproto.ConfigureWindowChecked(conn, window, mask, values).Check()
	conn.Sync()
}

// x11Conn opens a connection and resolves the window id.
func x11Conn(win fyne.Window) (*xgb.Conn, xproto.Window, error) {
	id, err := nativeWindowID(win)
	if err != nil {
		return nil, 0, err
	}
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, 0, err
	}
	return conn, xproto.Window(id), nil
}

// WindowClass is the stable X11 class used to match the window with its desktop
// entry; GLFW otherwise reports the window title, which carries the version.
const WindowClass = "mRemoteNG Go"

// SetWindowClass sets WM_CLASS so desktop shells can match the running window to
// the installed .desktop file.
func SetWindowClass(win fyne.Window) {
	conn, window, err := x11Conn(win)
	if err != nil {
		return
	}
	defer conn.Close()
	classAtom, err := xproto.InternAtom(conn, false, uint16(len("WM_CLASS")), "WM_CLASS").Reply()
	if err != nil {
		return
	}
	stringAtom, err := xproto.InternAtom(conn, false, uint16(len("STRING")), "STRING").Reply()
	if err != nil {
		return
	}
	data := []byte("mremoteng-go\x00" + WindowClass + "\x00")
	_ = xproto.ChangePropertyChecked(conn, xproto.PropModeReplace, window, classAtom.Atom,
		stringAtom.Atom, 8, uint32(len(data)), data).Check()
	conn.Sync()
}
