//go:build windows

// Package winembed implements the Win32 external-window-embedding
// mechanism validated by the Phase 0 spike (docs/spike-win32.md):
// find a process's session window by PID (with retry, since some clients
// recreate their window during startup) and reparent it into a host
// window via the DPI-aware SetParent recipe. It is shared by every
// backend that embeds an external process's window — RDP (stage 2.5) was
// the first; the spike's own notes anticipated AnyDesk (stage 2.7) reusing
// the same retry loop ("The same loop re-embeds when a client re-creates
// its window later (AnyDesk prep, stage 2.7)").
package winembed

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// DialogClassName is the window class of Windows' common dialog host
// (credential/trust prompts). FindTopLevelForPID skips it by default —
// the spike found these show up as visible top-levels of the target
// process alongside its actual session window.
const DialogClassName = "#32770"

const (
	// DefaultDeadline/DefaultPollInterval match the values the spike
	// validated against sdl-freerdp's window-recreation-during-init
	// behavior; callers with different needs can call FindAndAdopt with
	// their own.
	DefaultDeadline     = 15 * time.Second
	DefaultPollInterval = 200 * time.Millisecond
)

// FindAndAdopt polls for a visible top-level window owned by pid — skipping
// any window of class skipClass (pass DialogClassName for the common case)
// — until one is found, ctx is done, or deadline elapses. A retry loop is
// required per the spike's finding: some clients (sdl-freerdp confirmed;
// AnyDesk anticipated) create a provisional window and recreate it during
// their own startup, so the first handle discovered can already be gone by
// the time it would be acted on.
func FindAndAdopt(ctx context.Context, pid uint32, deadline time.Duration, pollInterval time.Duration, skipClass string) (uintptr, error) {
	giveUpAt := time.Now().Add(deadline)
	for {
		if hwnd, ok := FindTopLevelForPID(pid, skipClass); ok {
			return hwnd, nil
		}
		if time.Now().After(giveUpAt) {
			return 0, fmt.Errorf("no session window found for pid %d within %s", pid, deadline)
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// FindTopLevelForPID enumerates top-level windows and returns the first
// one owned by pid whose class isn't skipClass.
func FindTopLevelForPID(pid uint32, skipClass string) (uintptr, bool) {
	var found windows.HWND
	cb := func(hwnd windows.HWND, _ uintptr) uintptr {
		var winPID uint32
		if _, err := windows.GetWindowThreadProcessId(hwnd, &winPID); err != nil || winPID != pid {
			return 1 // continue enumeration
		}

		if skipClass != "" {
			var class [256]uint16
			n, err := windows.GetClassName(hwnd, &class[0], int32(len(class)))
			if err == nil && windows.UTF16ToString(class[:n]) == skipClass {
				return 1 // skip, keep looking
			}
		}

		found = hwnd
		return 0 // stop enumeration
	}
	windows.EnumWindows(syscall.NewCallback(cb), nil)
	return uintptr(found), found != 0
}

// Win32 APIs not wrapped by golang.org/x/sys/windows, bound the standard
// LazyDLL way.
var (
	user32                            = windows.NewLazySystemDLL("user32.dll")
	procSetParent                     = user32.NewProc("SetParent")
	procGetWindowLongPtrW             = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW             = user32.NewProc("SetWindowLongPtrW")
	procSetWindowPos                  = user32.NewProc("SetWindowPos")
	procGetAncestor                   = user32.NewProc("GetAncestor")
	procSetThreadDpiHostingBehavior   = user32.NewProc("SetThreadDpiHostingBehavior")
	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
)

// Style/GetAncestor/SetWindowPos constants used by EmbedChild, values per
// the Win32 SDK (winuser.h).
const (
	gwlStyle = ^uintptr(16 - 1) // GWL_STYLE = -16, as an unsigned uintptr (two's complement)

	wsPopup      = 0x80000000
	wsCaption    = 0x00C00000
	wsThickFrame = 0x00040000
	wsChild      = 0x40000000

	swpNoMove       = 0x0002
	swpNoSize       = 0x0001
	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	swpFrameChanged = 0x0020

	gaParent = 1 // GA_PARENT

	dpiHostingBehaviorMixed = 1 // per the spike: INVALID=-1, DEFAULT=0, MIXED=1 (passing 2 fails silently)

	// dpiAwarenessContextPerMonitorAwareV2 is
	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2, defined by the Win32 SDK
	// as ((DPI_AWARENESS_CONTEXT)-4) — a pseudo-handle, not a small int.
	dpiAwarenessContextPerMonitorAwareV2 = ^uintptr(4 - 1)
)

// EmbedChild reparents child into parent following the exact recipe the
// Phase 0 spike validated on a real Windows 11 VM (docs/spike-win32.md):
// SetParent first, then strip the top-level styles (WS_POPUP | WS_CAPTION
// | WS_THICKFRAME) and add WS_CHILD, then SetWindowPos with
// SWP_FRAMECHANGED to make the restyle take effect, then verify with
// GetAncestor rather than trusting SetParent's return value alone — on
// Windows 10+, SetParent silently no-ops (returns non-NULL, GetLastError
// clear, window stays top-level) when the caller and target windows have
// mismatched DPI_AWARENESS_CONTEXTs.
//
// Callers are responsible for the DPI-awareness half of that requirement:
// the owning process must call SetProcessMixedDpiAwareness at startup, and
// SetMixedDpiHostingBehavior on the thread that creates parent *before*
// parent is created — "the hosting behavior is captured per-window at
// creation time; setting it later does nothing" (spike finding).
// EmbedChild cannot retroactively fix a parent window that was created
// without that behavior set, which is why it's documented here instead of
// attempted inside EmbedChild itself.
func EmbedChild(parent, child windows.HWND) error {
	if r, _, err := procSetParent.Call(uintptr(child), uintptr(parent)); r == 0 {
		return fmt.Errorf("winembed: SetParent: %w", err)
	}

	style, _, _ := procGetWindowLongPtrW.Call(uintptr(child), gwlStyle)
	style &^= wsPopup | wsCaption | wsThickFrame
	style |= wsChild
	if _, _, err := procSetWindowLongPtrW.Call(uintptr(child), gwlStyle, style); err != nil && err != syscall.Errno(0) {
		return fmt.Errorf("winembed: SetWindowLongPtrW: %w", err)
	}

	if r, _, err := procSetWindowPos.Call(
		uintptr(child), 0, 0, 0, 0, 0,
		swpNoMove|swpNoSize|swpNoZOrder|swpNoActivate|swpFrameChanged,
	); r == 0 {
		return fmt.Errorf("winembed: SetWindowPos: %w", err)
	}

	actualParent, _, _ := procGetAncestor.Call(uintptr(child), gaParent)
	if windows.HWND(actualParent) != parent {
		return fmt.Errorf("winembed: SetParent did not take effect (GetAncestor reports %x, want %x) — likely a DPI_AWARENESS_CONTEXT mismatch between parent and child, see EmbedChild's doc comment", actualParent, parent)
	}
	return nil
}

// SetMixedDpiHostingBehavior sets DPI_HOSTING_BEHAVIOR_MIXED on the
// calling thread. Per the spike, this must be called before the window
// that will act as an embedding parent is created, on the same thread
// that creates it — see EmbedChild's doc comment.
func SetMixedDpiHostingBehavior() error {
	if _, _, err := procSetThreadDpiHostingBehavior.Call(dpiHostingBehaviorMixed); err != nil && err != syscall.Errno(0) {
		return fmt.Errorf("winembed: SetThreadDpiHostingBehavior: %w", err)
	}
	return nil
}

// SetProcessMixedDpiAwareness sets PER_MONITOR_AWARE_V2 process DPI
// awareness. Must be called once, early at process startup, before any
// window is created — see EmbedChild's doc comment for why this and
// SetMixedDpiHostingBehavior are both required for SetParent to work
// across processes on Windows 10+. Win32 only allows this to succeed once
// per process; a second call fails with access-denied.
func SetProcessMixedDpiAwareness() error {
	if r, _, err := procSetProcessDpiAwarenessContext.Call(dpiAwarenessContextPerMonitorAwareV2); r == 0 {
		return fmt.Errorf("winembed: SetProcessDpiAwarenessContext: %w", err)
	}
	return nil
}

// SetWindowPosition moves and resizes an already-embedded child window
// within its parent's client area (x/y/width/height are parent-client-
// relative pixels, matching what SetParent establishes). Intended to be
// called on every layout change of whatever host widget owns child, to
// keep it tracking a resizable/movable UI region — EmbedChild itself only
// places the window once, at embed time.
func SetWindowPosition(child windows.HWND, x, y, width, height int) error {
	if r, _, err := procSetWindowPos.Call(
		uintptr(child), 0,
		uintptr(x), uintptr(y), uintptr(width), uintptr(height),
		swpNoZOrder|swpNoActivate,
	); r == 0 {
		return fmt.Errorf("winembed: SetWindowPos (reposition): %w", err)
	}
	return nil
}
