package ui

import (
	"fmt"
	"image/color"
	"os"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// zoomTheme overrides only the text size, so the terminal can zoom without
// touching the rest of the window.
type zoomTheme struct {
	base fyne.Theme
	size float32
}

func (t zoomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return t.base.Color(name, variant)
}

func (t zoomTheme) Font(style fyne.TextStyle) fyne.Resource { return t.base.Font(style) }

func (t zoomTheme) Icon(name fyne.ThemeIconName) fyne.Resource { return t.base.Icon(name) }

func (t zoomTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameText {
		return t.size
	}
	return t.base.Size(name)
}

// zoomLevels documents the discrete steps used by the shortcuts and menu.

const (
	// zoomScrollStep is how much one wheel notch changes the text size.
	zoomScrollStep = 2
	// zoomScrollInterval throttles wheel zoom. Rebuilding the text grid is what
	// makes the glyphs follow the size, so a spinning wheel must not do it on
	// every event: that was the flicker that removed wheel zoom in the first
	// place.
	zoomScrollInterval = 120 * time.Millisecond
)

var (
	ctrlConnMu   sync.Mutex
	ctrlConn     *xgb.Conn
	ctrlConnRoot xproto.Window
)

// ctrlHeld reports whether Ctrl is down, for callers whose events carry no
// modifier information.
//
// Fyne's ScrollEvent has only positions, so the state comes from the X server.
// The connection is opened once and reused: doing it per event cost an X
// round-trip per wheel notch. Without a DISPLAY (a native Wayland session) this
// reports false and the discrete shortcuts remain the portable path.
func ctrlHeld() bool {
	if os.Getenv("DISPLAY") == "" {
		return false
	}
	ctrlConnMu.Lock()
	defer ctrlConnMu.Unlock()
	if ctrlConn == nil {
		conn, err := xgb.NewConn()
		if err != nil {
			return false
		}
		ctrlConn = conn
		ctrlConnRoot = xproto.Setup(conn).DefaultScreen(conn).Root
	}
	reply, err := xproto.QueryPointer(ctrlConn, ctrlConnRoot).Reply()
	if err != nil {
		return false
	}
	return reply.Mask&xproto.ModMaskControl != 0
}

// traceZoom records zoom decisions when MRE_TRACE_ZOOM is set, so a zoom that
// does nothing can be traced to the shortcut, the size change or the repaint
// without a debugger.
func traceZoom(format string, args ...any) {
	if os.Getenv("MRE_TRACE_ZOOM") == "" {
		return
	}
	file, err := os.OpenFile("/tmp/mremoteng-zoom.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()
	fmt.Fprintf(file, format+"\n", args...)
}
