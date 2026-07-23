package ui_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
	"github.com/mRemoteNG/mremoteng-go/internal/ui"
)

// baseFakeProtocol implements protocol.Protocol only — the common bones
// every fake below embeds, so each only needs to add what makes it a
// Terminal/Framebuffer/Window protocol.
type baseFakeProtocol struct {
	mu         sync.Mutex
	connectErr error
	onClose    func()
}

func (f *baseFakeProtocol) Connect(context.Context) error { return f.connectErr }
func (f *baseFakeProtocol) Disconnect() error             { return nil }
func (f *baseFakeProtocol) Focus()                        {}
func (f *baseFakeProtocol) Resize(int, int)               {}
func (f *baseFakeProtocol) OnError(func(error))           {}
func (f *baseFakeProtocol) OnClose(cb func())             { f.onClose = cb }

type fakeTerminalProtocol struct {
	baseFakeProtocol
	mu       sync.Mutex
	written  bytes.Buffer
	readOnce chan []byte // one chunk delivered, then Read blocks/returns EOF
}

func newFakeTerminalProtocol() *fakeTerminalProtocol {
	return &fakeTerminalProtocol{readOnce: make(chan []byte, 1)}
}

func (f *fakeTerminalProtocol) Read(p []byte) (int, error) {
	chunk, ok := <-f.readOnce
	if !ok {
		return 0, fmt.Errorf("closed")
	}
	return copy(p, chunk), nil
}

func (f *fakeTerminalProtocol) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.written.Write(p)
}

func (f *fakeTerminalProtocol) writtenString() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.written.String()
}

var _ protocol.TerminalProtocol = (*fakeTerminalProtocol)(nil)

type fakeWindowProtocol struct {
	baseFakeProtocol
	handle uintptr
}

func (f *fakeWindowProtocol) NativeWindowHandle() uintptr { return f.handle }

var _ protocol.WindowProtocol = (*fakeWindowProtocol)(nil)

// fakePlainProtocol implements none of TerminalProtocol/
// FramebufferProtocol/WindowProtocol — the "no view available" case.
type fakePlainProtocol struct{ baseFakeProtocol }

var _ protocol.Protocol = (*fakePlainProtocol)(nil)

func tabContent(t *testing.T, tabs *ui.SessionTabs, index int) fyne.CanvasObject {
	t.Helper()
	items := tabs.Widget.Items
	if index >= len(items) {
		t.Fatalf("tab index %d out of range (%d tabs)", index, len(items))
	}
	return items[index].Content
}

func TestSessionTabs_Open_DispatchesTerminalProtocolToTerminal(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	win := test.NewWindow(nil)
	defer win.Close()

	tabs := ui.NewSessionTabs(win)
	p := newFakeTerminalProtocol()
	close(p.readOnce) // avoid leaking pumpTerminal's goroutine blocked on Read

	tabs.Open("test", p)

	if _, ok := tabContent(t, tabs, 0).(*ui.Terminal); !ok {
		t.Errorf("tab content = %T, want *ui.Terminal", tabContent(t, tabs, 0))
	}
}

func TestSessionTabs_Open_DispatchesFramebufferProtocolToFramebufferView(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	win := test.NewWindow(nil)
	defer win.Close()

	tabs := ui.NewSessionTabs(win)
	fb := newFakeFramebuffer()
	close(fb.frames)

	tabs.Open("test", fb)

	if _, ok := tabContent(t, tabs, 0).(*ui.FramebufferView); !ok {
		t.Errorf("tab content = %T, want *ui.FramebufferView", tabContent(t, tabs, 0))
	}
}

func TestSessionTabs_Open_DispatchesWindowProtocolToNativeWindowHost(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	win := test.NewWindow(nil)
	defer win.Close()

	tabs := ui.NewSessionTabs(win)
	p := &fakeWindowProtocol{}

	tabs.Open("test", p)

	if _, ok := tabContent(t, tabs, 0).(*ui.NativeWindowHost); !ok {
		t.Errorf("tab content = %T, want *ui.NativeWindowHost", tabContent(t, tabs, 0))
	}
}

func TestSessionTabs_Open_UnsupportedProtocolShowsALabel(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	win := test.NewWindow(nil)
	defer win.Close()

	tabs := ui.NewSessionTabs(win)
	tabs.Open("test", &fakePlainProtocol{})

	label, ok := tabContent(t, tabs, 0).(*widget.Label)
	if !ok {
		t.Fatalf("tab content = %T, want *widget.Label", tabContent(t, tabs, 0))
	}
	if label.Text == "" {
		t.Error("label has no explanatory text")
	}
}

func TestSessionTabs_Open_ConnectFailure_ShowsErrorInTab(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	win := test.NewWindow(nil)
	defer win.Close()

	tabs := ui.NewSessionTabs(win)
	p := &fakePlainProtocol{}
	p.connectErr = fmt.Errorf("boom")

	tabs.Open("test", p)

	deadline := time.Now().Add(5 * time.Second)
	for {
		if label, ok := tabContent(t, tabs, 0).(*widget.Label); ok && strings.HasPrefix(label.Text, "Connection failed:") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("tab content never showed the connect error, last = %v", tabContent(t, tabs, 0))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestSessionTabs_OnClose_RemovesTheTab(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	win := test.NewWindow(nil)
	defer win.Close()

	tabs := ui.NewSessionTabs(win)
	p := &fakePlainProtocol{}

	tabs.Open("test", p)
	if len(tabs.Widget.Items) != 1 {
		t.Fatalf("tab count after Open = %d, want 1", len(tabs.Widget.Items))
	}

	p.onClose()

	deadline := time.Now().Add(5 * time.Second)
	for len(tabs.Widget.Items) != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("tab was not removed after OnClose fired, count = %d", len(tabs.Widget.Items))
		}
		time.Sleep(10 * time.Millisecond)
	}
}
