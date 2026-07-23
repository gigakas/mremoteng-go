//go:build windows

package rdp

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// rdpClientExe is the Windows FreeRDP client. Per docs/spike-win32.md,
// FreeRDP's GitHub releases no longer ship wfreerdp.exe; the nightly CI
// publishes sdl-freerdp.exe (SDL client, window class "SDL_app"), which is
// the validated Windows target. It's a runtime dependency, never bundled
// (see the blueprint's non-negotiable "never link libfreerdp").
const rdpClientExe = "sdl-freerdp.exe"

// dialogClassName is the window class of Windows' common dialog host
// (credential/trust prompts). findTopLevelForPID skips it — the spike
// found these show up as visible top-levels of the FreeRDP process
// alongside the actual session window.
const dialogClassName = "#32770"

const (
	findAdoptDeadline = 15 * time.Second
	findAdoptPoll     = 200 * time.Millisecond
)

func (s *Session) connectPlatform(ctx context.Context) error {
	path, err := exec.LookPath(rdpClientExe)
	if err != nil {
		return fmt.Errorf("rdp: %s not found on PATH (install FreeRDP; see docs/spike-win32.md's packaging note): %w", rdpClientExe, err)
	}

	cmd := exec.Command(path, s.buildArgs()...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("rdp: start %s: %w", rdpClientExe, err)
	}

	hwnd, err := findAndAdopt(ctx, uint32(cmd.Process.Pid))
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return fmt.Errorf("rdp: locate session window: %w", err)
	}

	s.mu.Lock()
	s.cmd = cmd
	s.hwnd = hwnd
	s.mu.Unlock()

	s.waitDone = make(chan struct{})
	go func() {
		cmd.Wait()
		close(s.waitDone)
		s.FireClose()
	}()

	return nil
}

// findAndAdopt polls for a visible top-level window owned by pid until one
// is found, ctx is done, or findAdoptDeadline elapses. A retry loop is
// required per the spike's finding #3: sdl-freerdp creates a provisional
// window and re-creates it during Direct3D renderer init, so the first
// handle discovered can already be gone by the time it would be acted on.
func findAndAdopt(ctx context.Context, pid uint32) (uintptr, error) {
	deadline := time.Now().Add(findAdoptDeadline)
	for {
		if hwnd, ok := findTopLevelForPID(pid); ok {
			return hwnd, nil
		}
		if time.Now().After(deadline) {
			return 0, fmt.Errorf("no session window found for pid %d within %s", pid, findAdoptDeadline)
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(findAdoptPoll):
		}
	}
}

// findTopLevelForPID enumerates top-level windows and returns the first
// one owned by pid whose class isn't the dialog host (see
// dialogClassName).
func findTopLevelForPID(pid uint32) (uintptr, bool) {
	var found windows.HWND
	cb := func(hwnd windows.HWND, _ uintptr) uintptr {
		var winPID uint32
		if _, err := windows.GetWindowThreadProcessId(hwnd, &winPID); err != nil || winPID != pid {
			return 1 // continue enumeration
		}

		var class [256]uint16
		n, err := windows.GetClassName(hwnd, &class[0], int32(len(class)))
		if err == nil && windows.UTF16ToString(class[:n]) == dialogClassName {
			return 1 // skip dialogs, keep looking
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

// Style/GetAncestor/SetWindowPos constants used by embedChild, values per
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
// Callers (the Phase 3 UI, or this package's own tests) are responsible
// for the DPI-awareness half of that requirement: the owning process must
// call SetProcessDpiAwarenessContext(PER_MONITOR_AWARE_V2) at startup, and
// SetThreadDpiHostingBehavior(MIXED) on the thread that creates parent
// *before* parent is created — "the hosting behavior is captured
// per-window at creation time; setting it later does nothing" (spike
// finding). EmbedChild cannot retroactively fix a parent window that was
// created without that behavior set, which is why it's documented here
// instead of attempted inside EmbedChild itself.
func EmbedChild(parent, child windows.HWND) error {
	if r, _, err := procSetParent.Call(uintptr(child), uintptr(parent)); r == 0 {
		return fmt.Errorf("rdp: SetParent: %w", err)
	}

	style, _, _ := procGetWindowLongPtrW.Call(uintptr(child), gwlStyle)
	style &^= wsPopup | wsCaption | wsThickFrame
	style |= wsChild
	if _, _, err := procSetWindowLongPtrW.Call(uintptr(child), gwlStyle, style); err != nil && err != syscall.Errno(0) {
		return fmt.Errorf("rdp: SetWindowLongPtrW: %w", err)
	}

	if r, _, err := procSetWindowPos.Call(
		uintptr(child), 0, 0, 0, 0, 0,
		swpNoMove|swpNoSize|swpNoZOrder|swpNoActivate|swpFrameChanged,
	); r == 0 {
		return fmt.Errorf("rdp: SetWindowPos: %w", err)
	}

	actualParent, _, _ := procGetAncestor.Call(uintptr(child), gaParent)
	if windows.HWND(actualParent) != parent {
		return fmt.Errorf("rdp: SetParent did not take effect (GetAncestor reports %x, want %x) — likely a DPI_AWARENESS_CONTEXT mismatch between parent and child, see EmbedChild's doc comment", actualParent, parent)
	}
	return nil
}

// SetMixedDpiHostingBehavior sets DPI_HOSTING_BEHAVIOR_MIXED on the
// calling thread. Per the spike, this must be called before the window
// that will act as an embedding parent is created, on the same thread
// that creates it — see EmbedChild's doc comment.
func SetMixedDpiHostingBehavior() error {
	if _, _, err := procSetThreadDpiHostingBehavior.Call(dpiHostingBehaviorMixed); err != nil && err != syscall.Errno(0) {
		return fmt.Errorf("rdp: SetThreadDpiHostingBehavior: %w", err)
	}
	return nil
}

// SetProcessMixedDpiAwareness sets PER_MONITOR_AWARE_V2 process DPI
// awareness. Must be called once, early at process startup, before any
// window is created — see EmbedChild's doc comment for why this and
// SetMixedDpiHostingBehavior are both required for SetParent to work
// across processes on Windows 10+. Exposed here (rather than left for
// every caller to bind the syscall itself) specifically because getting
// the DPI_AWARENESS_CONTEXT constant wrong cost the original spike
// several rounds of silent failures.
func SetProcessMixedDpiAwareness() error {
	if r, _, err := procSetProcessDpiAwarenessContext.Call(dpiAwarenessContextPerMonitorAwareV2); r == 0 {
		return fmt.Errorf("rdp: SetProcessDpiAwarenessContext: %w", err)
	}
	return nil
}
