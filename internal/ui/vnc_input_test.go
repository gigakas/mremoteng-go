package ui

import (
	"image"
	"reflect"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

func newInputView(width, height int, size fyne.Size) (*vncView, *fakeVNC) {
	session := &fakeVNC{pixels: image.NewRGBA(image.Rect(0, 0, width, height))}
	view := newVNCView(session)
	view.Resize(size)
	return view, session
}

func mouseAt(x, y float32, button desktop.MouseButton) *desktop.MouseEvent {
	return &desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(x, y)}, Button: button}
}

func TestVNCView_Pointer_MapsThroughLetterboxBands(t *testing.T) {
	// A 100x50 screen in a 200x200 widget is drawn 200x100, centred: 50px bands
	// above and below. The centre of the widget is the centre of the screen.
	view, session := newInputView(100, 50, fyne.NewSize(200, 200))

	view.MouseMoved(mouseAt(100, 100, desktop.MouseButtonPrimary))
	view.MouseMoved(mouseAt(0, 50, desktop.MouseButtonPrimary))
	view.MouseMoved(mouseAt(199, 5, desktop.MouseButtonPrimary))

	want := [][3]int{{50, 25, 0}, {0, 0, 0}, {99, 0, 0}}
	if !reflect.DeepEqual(session.pointer, want) {
		t.Errorf("pointer = %v, want %v (band positions clamp to the screen edge)", session.pointer, want)
	}
}

func TestVNCView_Buttons_PressDragAndRelease(t *testing.T) {
	view, session := newInputView(100, 100, fyne.NewSize(100, 100))

	view.MouseDown(mouseAt(10, 10, desktop.MouseButtonPrimary))
	view.MouseMoved(mouseAt(40, 40, desktop.MouseButtonPrimary))
	view.MouseDown(mouseAt(40, 40, desktop.MouseButtonSecondary))
	view.MouseUp(mouseAt(40, 40, desktop.MouseButtonSecondary))
	view.MouseUp(mouseAt(40, 40, desktop.MouseButtonPrimary))

	want := [][3]int{{10, 10, 1}, {40, 40, 1}, {40, 40, 5}, {40, 40, 1}, {40, 40, 0}}
	if !reflect.DeepEqual(session.pointer, want) {
		t.Errorf("pointer = %v, want %v (a move during a drag keeps the buttons down)", session.pointer, want)
	}
}

func TestVNCView_Scroll_SendsWheelButtons(t *testing.T) {
	view, session := newInputView(100, 100, fyne.NewSize(100, 100))

	view.Scrolled(&fyne.ScrollEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(5, 5)}, Scrolled: fyne.NewDelta(0, 1)})
	view.Scrolled(&fyne.ScrollEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(5, 5)}, Scrolled: fyne.NewDelta(0, -1)})

	want := [][3]int{{5, 5, 8}, {5, 5, 0}, {5, 5, 16}, {5, 5, 0}}
	if !reflect.DeepEqual(session.pointer, want) {
		t.Errorf("pointer = %v, want wheel up then wheel down as buttons 4 and 5", session.pointer)
	}
}

func TestVNCView_TypedRune_SendsCharacterAsPressAndRelease(t *testing.T) {
	view, session := newInputView(10, 10, fyne.NewSize(10, 10))

	view.TypedRune('A')
	view.TypedRune('ñ')
	view.TypedRune('€')

	want := [][2]uint32{{'A', 1}, {'A', 0}, {0xf1, 1}, {0xf1, 0}, {0x010020ac, 1}, {0x010020ac, 0}}
	if !reflect.DeepEqual(session.keys, want) {
		t.Errorf("keys = %x, want %x (Latin-1 as is, other code points as 0x01000000+point)", session.keys, want)
	}
}

func TestVNCView_SpecialKeys_GoThroughKeyDownAndUp(t *testing.T) {
	view, session := newInputView(10, 10, fyne.NewSize(10, 10))

	view.KeyDown(&fyne.KeyEvent{Name: fyne.KeyReturn})
	view.KeyUp(&fyne.KeyEvent{Name: fyne.KeyReturn})
	view.KeyDown(&fyne.KeyEvent{Name: fyne.KeyLeft})
	view.KeyUp(&fyne.KeyEvent{Name: fyne.KeyLeft})

	want := [][2]uint32{{0xff0d, 1}, {0xff0d, 0}, {0xff51, 1}, {0xff51, 0}}
	if !reflect.DeepEqual(session.keys, want) {
		t.Errorf("keys = %x, want %x", session.keys, want)
	}
}

func TestVNCView_PlainLetter_IsNotSentTwice(t *testing.T) {
	view, session := newInputView(10, 10, fyne.NewSize(10, 10))

	// Fyne reports a plain letter as a key and as typed text; only the text goes out.
	view.KeyDown(&fyne.KeyEvent{Name: fyne.KeyA})
	view.TypedRune('a')
	view.KeyUp(&fyne.KeyEvent{Name: fyne.KeyA})

	want := [][2]uint32{{'a', 1}, {'a', 0}}
	if !reflect.DeepEqual(session.keys, want) {
		t.Errorf("keys = %x, want one press and release of a", session.keys)
	}
}

func TestVNCView_CtrlLetter_SendsModifierAndKey(t *testing.T) {
	view, session := newInputView(10, 10, fyne.NewSize(10, 10))

	// Fyne gives no typed text for Ctrl+C, so the keys themselves must go out.
	view.KeyDown(&fyne.KeyEvent{Name: "LeftControl"})
	view.KeyDown(&fyne.KeyEvent{Name: fyne.KeyC})
	view.KeyUp(&fyne.KeyEvent{Name: fyne.KeyC})
	view.KeyUp(&fyne.KeyEvent{Name: "LeftControl"})

	want := [][2]uint32{{0xffe3, 1}, {'c', 1}, {'c', 0}, {0xffe3, 0}}
	if !reflect.DeepEqual(session.keys, want) {
		t.Errorf("keys = %x, want %x", session.keys, want)
	}
}

func TestVNCView_FocusLost_ReleasesHeldKeys(t *testing.T) {
	view, session := newInputView(10, 10, fyne.NewSize(10, 10))
	view.KeyDown(&fyne.KeyEvent{Name: "LeftControl"})

	view.FocusLost()
	view.KeyUp(&fyne.KeyEvent{Name: "LeftControl"})

	want := [][2]uint32{{0xffe3, 1}, {0xffe3, 0}}
	if !reflect.DeepEqual(session.keys, want) {
		t.Errorf("keys = %x, want the control key released once, on focus loss", session.keys)
	}
	// With nothing held, a letter is text again.
	view.TypedRune('x')
	if got := session.keys[len(session.keys)-2:]; got[0][0] != 'x' {
		t.Errorf("after focus loss typed text was not sent: %x", session.keys)
	}
}
