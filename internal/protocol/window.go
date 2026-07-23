package protocol

// WindowProtocol is implemented by backends whose session renders into a
// native OS window that the UI is expected to embed by reparenting — the
// pattern Phase 0 validated for external processes (SetParent on Windows,
// XReparentWindow on Linux) and reused here for anything that owns a
// native window handle, whether that window belongs to an external
// process (RDP, AnyDesk — later stages) or was created in-process (the
// HTTP/HTTPS webview, stage 2.3, via cgo).
type WindowProtocol interface {
	Protocol

	// NativeWindowHandle returns the platform window handle to reparent:
	// an HWND on Windows, an X11 Window ID on Linux. It is only valid
	// between a successful Connect and the session ending, and is the
	// zero value until the window actually exists.
	NativeWindowHandle() uintptr
}
