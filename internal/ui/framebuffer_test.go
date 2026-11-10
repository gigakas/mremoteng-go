package ui_test

import (
	"context"
	"image"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"

	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
	"github.com/mRemoteNG/mremoteng-go/internal/ui"
)

// fakeFramebuffer is a minimal protocol.FramebufferProtocol test double,
// the same spirit as internal/protocol's own fakeProtocol from stage 2.1
// — no real VNC connection, just enough to exercise FramebufferView's
// wiring.
type fakeFramebuffer struct {
	frames chan image.Image

	mu       sync.Mutex
	keys     []keyCall
	pointers []pointerCall
}

type keyCall struct {
	keysym uint32
	down   bool
}

type pointerCall struct {
	buttons protocol.PointerButtons
	x, y    int
}

func newFakeFramebuffer() *fakeFramebuffer {
	return &fakeFramebuffer{frames: make(chan image.Image, 4)}
}

func (f *fakeFramebuffer) Connect(context.Context) error { return nil }
func (f *fakeFramebuffer) Disconnect() error             { close(f.frames); return nil }
func (f *fakeFramebuffer) Focus()                        {}
func (f *fakeFramebuffer) Resize(int, int)               {}
func (f *fakeFramebuffer) OnError(func(error))           {}
func (f *fakeFramebuffer) OnClose(func())                {}
func (f *fakeFramebuffer) Frames() <-chan image.Image    { return f.frames }

func (f *fakeFramebuffer) SendKey(keysym uint32, down bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keys = append(f.keys, keyCall{keysym, down})
}

func (f *fakeFramebuffer) SendPointer(buttons protocol.PointerButtons, x, y int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pointers = append(f.pointers, pointerCall{buttons, x, y})
}

func (f *fakeFramebuffer) recordedKeys() []keyCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]keyCall(nil), f.keys...)
}

func (f *fakeFramebuffer) recordedPointers() []pointerCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]pointerCall(nil), f.pointers...)
}

var _ protocol.FramebufferProtocol = (*fakeFramebuffer)(nil)

func TestFramebufferView_Attach_RendersPushedFrames(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	win := test.NewWindow(nil)
	defer win.Close()

	view := ui.NewFramebufferView()
	win.SetContent(view)

	fb := newFakeFramebuffer()
	view.Attach(fb)

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	fb.frames <- img

	deadline := time.Now().Add(5 * time.Second)
	for view.CurrentImage() == nil {
		if time.Now().After(deadline) {
			t.Fatal("frame was not rendered within 5s")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if view.CurrentImage() != image.Image(img) {
		t.Error("rendered image is not the one pushed through Frames()")
	}
}

func TestFramebufferView_TypedRune_SendsPressAndRelease(t *testing.T) {
	view := ui.NewFramebufferView()
	fb := newFakeFramebuffer()
	view.Attach(fb)
	close(fb.frames)

	view.TypedRune('a')

	got := fb.recordedKeys()
	want := []keyCall{{'a', true}, {'a', false}}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("recorded keys = %v, want %v", got, want)
	}
}

func TestFramebufferView_TypedKey_MapsToX11Keysym(t *testing.T) {
	view := ui.NewFramebufferView()
	fb := newFakeFramebuffer()
	view.Attach(fb)
	close(fb.frames)

	view.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

	got := fb.recordedKeys()
	want := []keyCall{{0xff0d, true}, {0xff0d, false}}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("recorded keys = %v, want %v (XK_Return)", got, want)
	}
}

func TestFramebufferView_MouseDownMoveUp_TracksHeldButtons(t *testing.T) {
	view := ui.NewFramebufferView()
	fb := newFakeFramebuffer()
	view.Attach(fb)
	close(fb.frames)

	down := &desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(10, 20)}, Button: desktop.MouseButtonPrimary}
	view.MouseDown(down)

	moved := &desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(15, 25)}}
	view.MouseMoved(moved)

	up := &desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(15, 25)}, Button: desktop.MouseButtonPrimary}
	view.MouseUp(up)

	got := fb.recordedPointers()
	want := []pointerCall{
		{protocol.PointerButtonLeft, 10, 20},
		{protocol.PointerButtonLeft, 15, 25}, // moved while the button was still held
		{0, 15, 25},                          // released
	}
	if len(got) != len(want) {
		t.Fatalf("recorded pointer events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("pointer event %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
