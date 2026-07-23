package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// NativeWindowHost is a Fyne widget that reserves layout space for a
// window belonging to a protocol.WindowProtocol backend (RDP/AnyDesk,
// external processes; the webview backend, in-process) and keeps it
// positioned over that space — the same reparent-and-track-geometry
// pattern the Phase 0 spike validated for a single fixed window, now
// driven by Fyne's own layout instead of a one-shot placement.
//
// The embed/reposition mechanics are platform-specific (embed_windows.go
// implements Embed via internal/protocol/winembed; other platforms don't
// implement it yet — see embed_windows.go's package-level doc comment for
// the Windows details, and the stage audit for what's missing elsewhere).
// This file holds only the platform-neutral widget shell: a transparent
// placeholder CanvasObject that gives NativeWindowHost a layout footprint,
// since Fyne needs *something* to size/position even though the actual
// window content is drawn natively outside Fyne's own rendering.
type NativeWindowHost struct {
	widget.BaseWidget
	placeholder *canvas.Rectangle

	// embed holds the platform-specific embedding state (a *embedState on
	// Windows; unused, always nil, where Embed isn't implemented — see
	// nativewindow_windows.go / nativewindow_other.go). Typed any here,
	// not in this platform-neutral file, is a Windows-only type.
	embed any
}

// NewNativeWindowHost creates a host with no window embedded yet. Call
// Embed once a protocol.WindowProtocol backend has connected and produced
// a NativeWindowHandle.
func NewNativeWindowHost() *NativeWindowHost {
	h := &NativeWindowHost{placeholder: canvas.NewRectangle(color.Transparent)}
	h.ExtendBaseWidget(h)
	return h
}

// CreateRenderer implements fyne.Widget.
func (h *NativeWindowHost) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(h.placeholder)
}
