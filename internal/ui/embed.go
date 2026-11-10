package ui

import (
	"errors"
	"fmt"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// embeddedArea is an X11 window placed inside the application window; FreeRDP
// renders its session into it through /parent-window.
type embeddedArea interface {
	// ID is the X11 window id handed to FreeRDP.
	ID() uint64
	// SetVisible maps or unmaps the area so a session never covers other tabs.
	SetVisible(visible bool)
	// Destroy removes the window and stops tracking geometry.
	Destroy()
}

// embedder creates an embedded area for an object inside win, clamped to the
// bounds of the container that is actually visible. It returns an error when the
// platform cannot provide one, so callers fall back to a separate window
// instead of trapping the user.
type embedder func(win fyne.Window, area, bounds fyne.CanvasObject) (embeddedArea, error)

// ErrNoX11 means the process is not running on an X11 server (for example a
// native Wayland session), so foreign windows cannot be embedded.
var ErrNoX11 = errors.New("ui: embedding needs an X11/XWayland session")

// resizeInterval is how often the embedded window geometry is re-checked. Fyne
// exposes no resize callback, and a 250 ms poll is cheap next to a desktop
// session.
const resizeInterval = 250 * time.Millisecond

// embedX11 creates a child window inside the Fyne window and keeps it aligned
// with area, so expanding or contracting the application rescales the session.
func embedX11(win fyne.Window, area, bounds fyne.CanvasObject) (embeddedArea, error) {
	if os.Getenv("DISPLAY") == "" {
		return nil, ErrNoX11
	}
	parent, err := nativeWindowID(win)
	if err != nil {
		return nil, err
	}
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, err
	}
	id, err := conn.NewId()
	if err != nil {
		conn.Close()
		return nil, err
	}

	embedded := &x11Area{
		conn:   conn,
		window: xproto.Window(id),
		parent: xproto.Window(parent),
		area:   area,
		bounds: bounds,
		driver: fyne.CurrentApp().Driver(),
		canvas: win.Canvas(),
		stop:   make(chan struct{}),
	}
	geometry := embedded.geometry()
	if geometry.width < 1 || geometry.height < 1 {
		conn.Close()
		return nil, errors.New("ui: the session area is not visible yet")
	}
	err = xproto.CreateWindowChecked(conn, 0, embedded.window, embedded.parent,
		geometry.x, geometry.y, geometry.width, geometry.height, 0,
		xproto.WindowClassInputOutput, xproto.WindowClassCopyFromParent, 0, nil).Check()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := xproto.MapWindowChecked(conn, embedded.window).Check(); err != nil {
		conn.Close()
		return nil, err
	}
	conn.Sync()

	embedded.last = geometry
	go embedded.track()
	return embedded, nil
}

type areaGeometry struct {
	x, y          int16
	width, height uint16
}

type x11Area struct {
	conn   *xgb.Conn
	window xproto.Window
	parent xproto.Window
	area   fyne.CanvasObject
	bounds fyne.CanvasObject
	driver fyne.Driver
	canvas fyne.Canvas
	stop   chan struct{}
	last   areaGeometry
}

func (a *x11Area) ID() uint64 { return uint64(a.window) }

// geometry reads the current area placement clamped to the visible bounds. It
// must run on the UI thread.
//
// Fyne draws everything into one X11 window, so a child window is not clipped by
// the widget hierarchy: without clamping, the session would spill over the
// sidebar, the Config panel and the status bar.
func (a *x11Area) geometry() areaGeometry {
	// Canvas coordinates already share the X11 window origin: do not subtract
	// the content origin, which sits below the in-window menu bar.
	area := rectOf(a.driver, a.area)
	bounds := rectOf(a.driver, a.bounds)
	visible := intersect(area, bounds)
	if visible.width < 1 || visible.height < 1 {
		return areaGeometry{}
	}
	// Fyne positions are logical points; X11 child windows use physical pixels.
	scale := float32(1)
	if a.canvas != nil {
		scale = a.canvas.Scale()
	}
	scaled := scaleRect(visible, scale)
	fmt.Fprintf(os.Stderr, "embed: scale=%.2f area=(%.0f,%.0f %.0fx%.0f) bounds=(%.0f,%.0f %.0fx%.0f) visible=(%.0f,%.0f %.0fx%.0f)\n",
		scale,
		area.x, area.y, area.width, area.height,
		bounds.x, bounds.y, bounds.width, bounds.height,
		visible.x, visible.y, visible.width, visible.height)
	return areaGeometry{
		x:      int16(scaled.x),
		y:      int16(scaled.y),
		width:  uint16(scaled.width),
		height: uint16(scaled.height),
	}
}

func scaleRect(r rect, scale float32) rect {
	if scale <= 0 {
		scale = 1
	}
	return rect{x: r.x * scale, y: r.y * scale, width: r.width * scale, height: r.height * scale}
}

type rect struct {
	x, y, width, height float32
}

func rectOf(driver fyne.Driver, object fyne.CanvasObject) rect {
	position := driver.AbsolutePositionForObject(object)
	size := object.Size()
	return rect{x: position.X, y: position.Y, width: size.Width, height: size.Height}
}

func intersect(a, b rect) rect {
	left := maxF(a.x, b.x)
	top := maxF(a.y, b.y)
	right := minF(a.x+a.width, b.x+b.width)
	bottom := minF(a.y+a.height, b.y+b.height)
	return rect{x: left, y: top, width: maxF(right-left, 0), height: maxF(bottom-top, 0)}
}

func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func minF(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// track keeps the embedded window aligned with its tab area.
func (a *x11Area) track() {
	ticker := time.NewTicker(resizeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-a.stop:
			return
		case <-ticker.C:
			fyne.Do(func() { a.fit() })
		}
	}
}

// fit applies the current geometry when it changed, and keeps the session
// window matched to it even when the client recreated that window.
func (a *x11Area) fit() {
	geometry := a.geometry()
	if geometry.width < 1 || geometry.height < 1 {
		return
	}
	if geometry == a.last {
		a.fitChildren(geometry)
		return
	}
	a.last = geometry
	mask := uint16(xproto.ConfigWindowX | xproto.ConfigWindowY | xproto.ConfigWindowWidth | xproto.ConfigWindowHeight)
	values := []uint32{
		uint32(uint16(geometry.x)),
		uint32(uint16(geometry.y)),
		uint32(geometry.width),
		uint32(geometry.height),
	}
	_ = xproto.ConfigureWindowChecked(a.conn, a.window, mask, values).Check()
	a.fitChildren(geometry)
	a.conn.Sync()
}

// fitChildren resizes the session window that FreeRDP created inside the area.
// FreeRDP asks for its own size (for example 1024x768); with +smart-sizing it
// scales its framebuffer to whatever size the window has, so matching the window
// to the tab is what keeps the session inside the container.
func (a *x11Area) fitChildren(geometry areaGeometry) {
	tree, err := xproto.QueryTree(a.conn, a.window).Reply()
	if err != nil {
		return
	}
	for _, child := range tree.Children {
		childGeometry, err := xproto.GetGeometry(a.conn, xproto.Drawable(child)).Reply()
		if err != nil {
			continue
		}
		if childGeometry.Width == geometry.width && childGeometry.Height == geometry.height &&
			childGeometry.X == 0 && childGeometry.Y == 0 {
			continue
		}
		mask := uint16(xproto.ConfigWindowX | xproto.ConfigWindowY | xproto.ConfigWindowWidth | xproto.ConfigWindowHeight)
		values := []uint32{0, 0, uint32(geometry.width), uint32(geometry.height)}
		_ = xproto.ConfigureWindowChecked(a.conn, child, mask, values).Check()
	}
}

func (a *x11Area) SetVisible(visible bool) {
	if visible {
		_ = xproto.MapWindowChecked(a.conn, a.window).Check()
	} else {
		_ = xproto.UnmapWindowChecked(a.conn, a.window).Check()
	}
	a.conn.Sync()
}

func (a *x11Area) Destroy() {
	select {
	case <-a.stop:
		return
	default:
		close(a.stop)
	}
	_ = xproto.DestroyWindowChecked(a.conn, a.window).Check()
	a.conn.Sync()
	a.conn.Close()
}

// nativeWindowID returns the X11 window id of a Fyne window.
func nativeWindowID(win fyne.Window) (uint64, error) {
	native, ok := win.(driver.NativeWindow)
	if !ok {
		return 0, ErrNoX11
	}
	var id uint64
	native.RunNative(func(context any) {
		if x11, ok := context.(driver.X11WindowContext); ok {
			id = uint64(x11.WindowHandle)
		}
	})
	if id == 0 {
		return 0, ErrNoX11
	}
	return id, nil
}
