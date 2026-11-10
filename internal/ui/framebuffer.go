package ui

import (
	"image"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
)

// FramebufferView renders a protocol.FramebufferProtocol's pushed frames
// (VNC, stage 2.4) and forwards pointer/keyboard input back to it.
//
// v1 simplification, stated up front: the image is shown at its native
// resolution (canvas.ImageFillOriginal) rather than scaled to fit the
// tab — this keeps widget-space coordinates identical to framebuffer
// pixel coordinates for pointer forwarding with no scale-factor math,
// avoiding a class of off-by-scale bugs, at the cost of not fitting
// arbitrary window sizes. Fit-to-window scaling (and the coordinate
// remapping it requires) is a natural v2 follow-up, not attempted here.
type FramebufferView struct {
	widget.BaseWidget

	image *canvas.Image
	fp    protocol.FramebufferProtocol

	mu      sync.Mutex
	buttons protocol.PointerButtons
}

// NewFramebufferView creates an empty view. Call Attach once a session is
// connected.
func NewFramebufferView() *FramebufferView {
	v := &FramebufferView{image: canvas.NewImageFromImage(nil)}
	v.image.FillMode = canvas.ImageFillOriginal
	v.ExtendBaseWidget(v)
	return v
}

// Attach starts rendering fp's pushed frames and enables input
// forwarding to it. Runs a goroutine that reads fp.Frames() until the
// channel closes (i.e. until the session ends, per
// protocol.FramebufferProtocol's documented contract) — every UI update
// it makes is marshaled onto Fyne's main goroutine via fyne.Do, since
// Frames() delivery happens on whatever goroutine the backend's own
// read loop runs on, not Fyne's.
func (v *FramebufferView) Attach(fp protocol.FramebufferProtocol) {
	v.fp = fp
	go func() {
		for frame := range fp.Frames() {
			img := frame
			fyne.Do(func() {
				v.image.Image = img
				v.image.Refresh()
			})
		}
	}()
}

// CreateRenderer implements fyne.Widget.
func (v *FramebufferView) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(v.image)
}

// CurrentImage returns the most recently rendered frame, or nil before
// the first one arrives. Exists mainly so tests can observe Attach's
// effect without reaching into unexported state.
func (v *FramebufferView) CurrentImage() image.Image {
	return v.image.Image
}

// Tapped implements fyne.Tappable: clicking the view requests keyboard
// focus, the same as Terminal.
func (v *FramebufferView) Tapped(*fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(v); c != nil {
		c.Focus(v)
	}
}

func (v *FramebufferView) FocusGained() {}
func (v *FramebufferView) FocusLost()   {}

// TypedRune implements fyne.Focusable. X11 keysyms for printable
// ASCII/Latin-1 characters are numerically identical to their character
// code (the X11 keysymdef.h convention), so no lookup table is needed
// for this case — only the non-printable keys below need one.
func (v *FramebufferView) TypedRune(r rune) {
	if v.fp == nil {
		return
	}
	v.fp.SendKey(uint32(r), true)
	v.fp.SendKey(uint32(r), false)
}

// TypedKey implements fyne.Focusable for non-printable keys, mapped to
// their X11 keysym (see framebufferKeysyms). Unmapped keys are ignored,
// the same documented v1 limitation as Terminal.TypedKey.
func (v *FramebufferView) TypedKey(e *fyne.KeyEvent) {
	if v.fp == nil {
		return
	}
	if sym, ok := framebufferKeysyms[e.Name]; ok {
		v.fp.SendKey(sym, true)
		v.fp.SendKey(sym, false)
	}
}

// X11 keysyms for non-printable keys (X11 keysymdef.h). Printable
// ASCII/Latin-1 keys don't need an entry — see TypedRune.
var framebufferKeysyms = map[fyne.KeyName]uint32{
	fyne.KeyBackspace: 0xff08,
	fyne.KeyTab:       0xff09,
	fyne.KeyReturn:    0xff0d,
	fyne.KeyEnter:     0xff0d,
	fyne.KeyEscape:    0xff1b,
	fyne.KeyDelete:    0xffff,
	fyne.KeyHome:      0xff50,
	fyne.KeyLeft:      0xff51,
	fyne.KeyUp:        0xff52,
	fyne.KeyRight:     0xff53,
	fyne.KeyDown:      0xff54,
	fyne.KeyPageUp:    0xff55,
	fyne.KeyPageDown:  0xff56,
	fyne.KeyEnd:       0xff57,
}

// MouseDown/MouseUp implement desktop.Mouseable.
func (v *FramebufferView) MouseDown(e *desktop.MouseEvent) {
	v.setButton(e.Button, true)
	v.sendPointer(e.Position)
}

func (v *FramebufferView) MouseUp(e *desktop.MouseEvent) {
	v.setButton(e.Button, false)
	v.sendPointer(e.Position)
}

// MouseMoved implements desktop.Hoverable: reports the current position
// with whatever buttons are already held, so a click-drag reaches the
// remote side as a held-button move, not just discrete down/up points.
// MouseIn/MouseOut are part of the same interface but have nothing
// meaningful to forward.
func (v *FramebufferView) MouseIn(*desktop.MouseEvent) {}
func (v *FramebufferView) MouseOut()                   {}
func (v *FramebufferView) MouseMoved(e *desktop.MouseEvent) {
	v.sendPointer(e.Position)
}

func (v *FramebufferView) setButton(b desktop.MouseButton, down bool) {
	var bit protocol.PointerButtons
	switch b {
	case desktop.MouseButtonPrimary:
		bit = protocol.PointerButtonLeft
	case desktop.MouseButtonSecondary:
		bit = protocol.PointerButtonRight
	case desktop.MouseButtonTertiary:
		bit = protocol.PointerButtonMiddle
	default:
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if down {
		v.buttons |= bit
	} else {
		v.buttons &^= bit
	}
}

func (v *FramebufferView) sendPointer(pos fyne.Position) {
	if v.fp == nil {
		return
	}
	v.mu.Lock()
	buttons := v.buttons
	v.mu.Unlock()
	v.fp.SendPointer(buttons, int(pos.X), int(pos.Y))
}

var (
	_ fyne.Widget       = (*FramebufferView)(nil)
	_ fyne.Focusable    = (*FramebufferView)(nil)
	_ fyne.Tappable     = (*FramebufferView)(nil)
	_ desktop.Mouseable = (*FramebufferView)(nil)
	_ desktop.Hoverable = (*FramebufferView)(nil)
)
