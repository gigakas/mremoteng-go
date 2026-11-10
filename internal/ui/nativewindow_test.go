package ui_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/mRemoteNG/mremoteng-go/internal/ui"
)

func TestNativeWindowHost_ImplementsFyneWidget(t *testing.T) {
	var _ fyne.Widget = ui.NewNativeWindowHost()
}

// TestNativeWindowHost_Embed_RequiresNativeWindowAccess is the boundary
// this package can actually test automatically: Fyne's headless test
// driver (used throughout this phase, since real windows can't be seen
// in this environment — see blueprint/phase-3-ui.md's phase-wide note)
// does not implement driver.NativeWindow, so Embed must fail clearly
// against it rather than panic or silently no-op.
//
// What this does NOT test: actual reparenting against a real Fyne
// window's real HWND. That would need a live (non-headless) Fyne window,
// which — unlike the webview probe in stage 2.3 — isn't practical to spin
// up from inside a `go test` run without a full ShowAndRun() event loop.
// The underlying mechanism (winembed.EmbedChild) already has its own
// real integration test against a hand-built Win32 window
// (internal/protocol/winembed/winembed_test.go, written in stage 2.5);
// what's untested here is specifically the last step of wiring that
// mechanism to a genuine Fyne-owned window. See the stage audit.
func TestNativeWindowHost_Embed_RequiresNativeWindowAccess(t *testing.T) {
	host := ui.NewNativeWindowHost()
	win := test.NewWindow(nil)
	defer win.Close()

	err := host.Embed(win, 0x1234)
	if err == nil {
		t.Fatal("expected Embed to fail against a headless test window")
	}
}

func TestNativeWindowHost_ResizeAndMove_DoNotPanicWithoutAnEmbeddedWindow(t *testing.T) {
	host := ui.NewNativeWindowHost()
	host.Resize(fyne.NewSize(200, 100))
	host.Move(fyne.NewPos(10, 20))
}
