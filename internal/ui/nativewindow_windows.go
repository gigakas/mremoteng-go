//go:build windows

package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
	"golang.org/x/sys/windows"

	"github.com/mRemoteNG/mremoteng-go/internal/protocol/winembed"
)

// embedState holds the Windows-specific pieces of NativeWindowHost:
// the reparented child's handle and the parent window it was embedded
// into, needed by reposition (called from Resize/Move) since Fyne's
// layout system, not application code, drives most geometry changes.
type embedState struct {
	parent windows.HWND
	child  windows.HWND
}

// Embed reparents handle (a protocol.WindowProtocol backend's
// NativeWindowHandle) into win using the exact DPI-aware recipe validated
// by the Phase 0 spike (internal/protocol/winembed.EmbedChild — the same
// function stage 2.5's RDP backend and stage 2.7's AnyDesk backend rely
// on for their own, non-UI tests; this is that mechanism's first
// production use with a real Fyne parent rather than a throwaway test
// window).
//
// win must implement driver.NativeWindow (true for Fyne's desktop
// windows) and the calling goroutine must be the one that created win —
// see winembed.EmbedChild's own doc comment for the DPI-awareness
// preconditions this depends on the *window's owner* having set up
// (SetProcessMixedDpiAwareness at startup, SetMixedDpiHostingBehavior
// before the window was created); NativeWindowHost has no way to verify
// those were done and does not attempt to redo them here.
func (h *NativeWindowHost) Embed(win fyne.Window, handle uintptr) error {
	nw, ok := win.(driver.NativeWindow)
	if !ok {
		return fmt.Errorf("ui: window does not expose native access (driver.NativeWindow)")
	}

	var parent windows.HWND
	var ctxErr error
	nw.RunNative(func(ctx any) {
		wctx, ok := ctx.(driver.WindowsWindowContext)
		if !ok {
			ctxErr = fmt.Errorf("ui: unexpected native window context %T, want driver.WindowsWindowContext", ctx)
			return
		}
		parent = windows.HWND(wctx.HWND)
	})
	if ctxErr != nil {
		return ctxErr
	}
	if parent == 0 {
		return fmt.Errorf("ui: RunNative did not provide a window handle")
	}

	child := windows.HWND(handle)
	if err := winembed.EmbedChild(parent, child); err != nil {
		return fmt.Errorf("ui: embed native window: %w", err)
	}

	h.embed = &embedState{parent: parent, child: child}
	h.reposition()
	return nil
}

// Resize implements fyne.CanvasObject, extended to keep an embedded
// child window's on-screen rectangle matching this widget's.
func (h *NativeWindowHost) Resize(size fyne.Size) {
	h.BaseWidget.Resize(size)
	h.placeholder.Resize(size)
	h.reposition()
}

// Move implements fyne.CanvasObject, extended the same way as Resize.
//
// Known limitation, stated plainly rather than assumed correct: Move's
// position is relative to this widget's immediate parent container, not
// necessarily the window's absolute client-area origin — accurate for a
// host placed directly in the window content, which is the only
// arrangement this has been exercised with. A host nested inside further
// layout containers (extra Border/HBox/VBox wrapping) would need those
// containers' cumulative offset added in, which Fyne has no simple public
// API to query and this code does not attempt — a real display would be
// needed to notice and fix any resulting misalignment, and none was
// available while writing this (see the stage audit).
func (h *NativeWindowHost) Move(pos fyne.Position) {
	h.BaseWidget.Move(pos)
	h.placeholder.Move(pos)
	h.reposition()
}

func (h *NativeWindowHost) reposition() {
	state, ok := h.embed.(*embedState)
	if !ok {
		return
	}
	pos := h.Position()
	size := h.Size()
	winembed.SetWindowPosition(state.child, int(pos.X), int(pos.Y), int(size.Width), int(size.Height))
}
