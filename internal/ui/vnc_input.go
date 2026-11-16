package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// This file holds the VNC view's input side: it turns Fyne pointer and keyboard
// events into RFB messages. Fyne only calls a widget's handlers when it implements
// the whole interface (Mouseable needs both MouseDown and MouseUp, Hoverable needs
// MouseIn, MouseMoved and MouseOut), so each set is complete even where one half
// does little.

// RFB pointer button bits.
const (
	vncButtonLeft   byte = 1
	vncButtonMiddle byte = 2
	vncButtonRight  byte = 4
	vncWheelUp      byte = 8
	vncWheelDown    byte = 16
)

var (
	_ desktop.Mouseable = (*vncView)(nil)
	_ desktop.Hoverable = (*vncView)(nil)
	_ fyne.Scrollable   = (*vncView)(nil)
	_ fyne.Focusable    = (*vncView)(nil)
	_ desktop.Keyable   = (*vncView)(nil)
	_ fyne.Tabbable     = (*vncView)(nil)
)

// remotePoint maps a widget position to a framebuffer pixel. The image is drawn
// contained, so it may leave empty bands on two sides; the position is measured
// from the image, not from the widget, and clamped to the screen.
func (v *vncView) remotePoint(position fyne.Position) (x, y int) {
	width, height := v.source.Size()
	size := v.Size()
	if width == 0 || height == 0 || size.Width <= 0 || size.Height <= 0 {
		return 0, 0
	}
	scale := size.Width / float32(width)
	if other := size.Height / float32(height); other < scale {
		scale = other
	}
	offsetX := (size.Width - scale*float32(width)) / 2
	offsetY := (size.Height - scale*float32(height)) / 2
	x = int((position.X - offsetX) / scale)
	y = int((position.Y - offsetY) / scale)
	return clampInt(x, 0, width-1), clampInt(y, 0, height-1)
}

func (v *vncView) sendPointer(position fyne.Position, buttons byte) {
	v.buttons = buttons
	x, y := v.remotePoint(position)
	_ = v.source.Pointer(x, y, buttons)
}

// buttonBit is the RFB bit of a Fyne mouse button.
func buttonBit(button desktop.MouseButton) byte {
	switch button {
	case desktop.MouseButtonPrimary:
		return vncButtonLeft
	case desktop.MouseButtonTertiary:
		return vncButtonMiddle
	case desktop.MouseButtonSecondary:
		return vncButtonRight
	}
	return 0
}

// MouseDown presses a button and takes the keyboard, so typing right after a click
// reaches the remote machine.
func (v *vncView) MouseDown(event *desktop.MouseEvent) {
	v.focus()
	v.sendPointer(event.Position, v.buttons|buttonBit(event.Button))
}

// MouseUp releases a button.
func (v *vncView) MouseUp(event *desktop.MouseEvent) {
	v.sendPointer(event.Position, v.buttons&^buttonBit(event.Button))
}

// MouseIn, MouseMoved and MouseOut keep the remote pointer in step with the local
// one. A drag keeps reporting the buttons that are down.
func (v *vncView) MouseIn(event *desktop.MouseEvent)    { v.sendPointer(event.Position, v.buttons) }
func (v *vncView) MouseMoved(event *desktop.MouseEvent) { v.sendPointer(event.Position, v.buttons) }
func (v *vncView) MouseOut()                            {}

// Scrolled turns the wheel into button 4 (up) or 5 (down) clicks, which is how RFB
// carries it, one click per notch up to a few per event.
func (v *vncView) Scrolled(event *fyne.ScrollEvent) {
	bit := vncWheelUp
	delta := event.Scrolled.DY
	if delta < 0 {
		bit, delta = vncWheelDown, -delta
	}
	if delta == 0 {
		return
	}
	notches := clampInt(int(delta), 1, 5)
	x, y := v.remotePoint(event.Position)
	for i := 0; i < notches; i++ {
		_ = v.source.Pointer(x, y, v.buttons|bit)
		_ = v.source.Pointer(x, y, v.buttons)
	}
}

// focus gives the view the keyboard.
func (v *vncView) focus() {
	if canvas := fyne.CurrentApp().Driver().CanvasForObject(v); canvas != nil {
		canvas.Focus(v)
	}
}

func (v *vncView) FocusGained() {}

// FocusLost lets go of every key still down. A modifier left pressed would make
// the remote machine treat the next thing the user does as a shortcut.
func (v *vncView) FocusLost() {
	for keysym := range v.held {
		_ = v.source.Key(keysym, false)
	}
	v.held = map[uint32]bool{}
	v.modifiers = 0
}

// AcceptsTab keeps Tab for the remote machine instead of moving Fyne's focus.
func (v *vncView) AcceptsTab() bool { return true }

// TypedRune sends text as the characters typed, so the keyboard layout and Shift
// are already applied.
func (v *vncView) TypedRune(r rune) {
	if v.modifiers > 0 {
		// The key itself went out with its modifier; sending the character as
		// well would type it twice.
		return
	}
	keysym := vncRuneKeysym(r)
	_ = v.source.Key(keysym, true)
	_ = v.source.Key(keysym, false)
}

// TypedKey does nothing: KeyDown and KeyUp already carried the key.
func (v *vncView) TypedKey(*fyne.KeyEvent) {}

// KeyDown sends keys that are not text (arrows, Enter, function keys, modifiers),
// and any key pressed with Ctrl, Alt or Super, which Fyne does not report as typed
// text.
func (v *vncView) KeyDown(event *fyne.KeyEvent) { v.sendKey(event.Name, true) }

// KeyUp releases the key KeyDown pressed.
func (v *vncView) KeyUp(event *fyne.KeyEvent) { v.sendKey(event.Name, false) }

func (v *vncView) sendKey(name fyne.KeyName, down bool) {
	keysym, printable, ok := vncKeysym(name)
	if !ok {
		return
	}
	if mod, isModifier := vncModifierKeys[name]; isModifier && mod.changes {
		if down {
			v.modifiers++
		} else if v.modifiers > 0 {
			v.modifiers--
		}
	}
	if printable && v.modifiers == 0 {
		// Plain text arrives through TypedRune.
		return
	}
	if !down && !v.held[keysym] {
		return
	}
	if down {
		v.held[keysym] = true
	} else {
		delete(v.held, keysym)
	}
	_ = v.source.Key(keysym, down)
}
