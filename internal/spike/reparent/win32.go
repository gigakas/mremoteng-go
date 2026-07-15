//go:build spike && windows

package main

import (
	"fmt"
	"log"
	"runtime"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
	"golang.org/x/sys/windows"
)

const (
	defaultClient = "wfreerdp.exe"
	// SetParent-based adoption is the mechanism this stage must validate:
	// it is what the original mRemoteNG uses (PuTTYNG -hwndparent aside)
	// and it works for any external process. parent-window mode is also
	// available to compare, if this wfreerdp build honors the flag.
	defaultMode = "reparent"
)

var (
	kernel32         = windows.NewLazySystemDLL("kernel32.dll")
	procSetLastError = kernel32.NewProc("SetLastError")

	user32                       = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procSetParent                = user32.NewProc("SetParent")
	procGetWindowLongPtrW        = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW        = user32.NewProc("SetWindowLongPtrW")
	procMoveWindow               = user32.NewProc("MoveWindow")
	procGetClientRect            = user32.NewProc("GetClientRect")
	procIsWindow                 = user32.NewProc("IsWindow")
	procPostMessageW             = user32.NewProc("PostMessageW")
	procGetClassNameW            = user32.NewProc("GetClassNameW")
	procGetAncestor              = user32.NewProc("GetAncestor")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	// Win10 1607+/1803+; probed with Find() before use.
	procSetThreadDpiHostingBehavior     = user32.NewProc("SetThreadDpiHostingBehavior")
	procSetProcessDpiAwarenessContext   = user32.NewProc("SetProcessDpiAwarenessContext")
	procGetWindowDpiAwarenessContext    = user32.NewProc("GetWindowDpiAwarenessContext")
	procGetAwarenessFromDpiAwarenessCtx = user32.NewProc("GetAwarenessFromDpiAwarenessContext")
)

const (
	wsChild        = 0x40000000
	wsPopup        = 0x80000000
	wsCaption      = 0x00C00000
	wsThickframe   = 0x00040000
	wmClose        = 0x0010
	gaParent       = 1
	swpNoSize      = 0x0001
	swpNoMove      = 0x0002
	swpNoZorder    = 0x0004
	swpFramechange = 0x0020
	// DPI_HOSTING_BEHAVIOR_MIXED: allows parenting windows with a
	// different DPI awareness context (mstsc is per-monitor v1, this app
	// v2 — without this, SetParent silently no-ops on Win10+). The enum is
	// INVALID=-1, DEFAULT=0, MIXED=1.
	dpiHostingBehaviorMixed = 1
)

var gwlStyle = -16 // GWL_STYLE; negative index, hence not a untyped const

type rect struct{ left, top, right, bottom int32 }

// platformInit runs first thing in main, on the main OS thread (Fyne's glfw
// driver locks the main goroutine to it in init). Aligns the process DPI
// awareness with mstsc (per-monitor v2) and sets MIXED hosting behavior,
// both captured by the Fyne window at creation time. Everything is logged:
// remote debugging happens over pasted logs.
func platformInit() {
	if procSetProcessDpiAwarenessContext.Find() == nil {
		// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 == (handle) -4
		ret, _, errno := procSetProcessDpiAwarenessContext.Call(^uintptr(3))
		log.Printf("dpi: SetProcessDpiAwarenessContext(PMv2) ret=%d errno=%v (access-denied = already set, fine)", ret, errno)
	}
	if procSetThreadDpiHostingBehavior.Find() == nil {
		prev, _, _ := procSetThreadDpiHostingBehavior.Call(dpiHostingBehaviorMixed)
		log.Printf("dpi: SetThreadDpiHostingBehavior(MIXED) prev=%d (-1 = call failed)", int32(prev))
	}
}

func className(hwnd uintptr) string {
	var cls [64]uint16
	n, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&cls[0])), 64)
	return windows.UTF16ToString(cls[:n])
}

// dpiAwarenessOf logs a window's DPI awareness (0 unaware, 1 system, 2
// per-monitor) — the mismatch that makes SetParent refuse silently.
func dpiAwarenessOf(label string, hwnd uintptr) {
	if procGetWindowDpiAwarenessContext.Find() != nil {
		return
	}
	ctx, _, _ := procGetWindowDpiAwarenessContext.Call(hwnd)
	aw, _, _ := procGetAwarenessFromDpiAwarenessCtx.Call(ctx)
	log.Printf("dpi: %s window 0x%x awareness=%d", label, hwnd, aw)
}

// winEmbedder implements sessionEmbedder with Win32 user32 calls. Unlike
// the X11 variant there is no event connection: resize-follow and death
// detection poll at 200 ms, which is enough for a spike.
type winEmbedder struct {
	parent    windows.HWND
	child     windows.HWND
	topOffset int
}

func parentHandle(w fyne.Window) uintptr {
	var parent uintptr
	w.(driver.NativeWindow).RunNative(func(ctx any) {
		if win, ok := ctx.(driver.WindowsWindowContext); ok {
			parent = win.HWND
		}
	})
	return parent
}

func newSessionEmbedder() (sessionEmbedder, error) { return &winEmbedder{}, nil }

func (e *winEmbedder) setTopOffset(px int) { e.topOffset = px }

func (e *winEmbedder) embedSession(parent uintptr, pid uint32, mode string, timeout time.Duration, exited <-chan error) error {
	e.parent = windows.HWND(parent)
	child, err := findTopLevelByPID(pid, timeout, exited)
	if err != nil {
		return err
	}
	e.child = child

	if mode != "parent-window" {
		// SetThreadDpiHostingBehavior is per-OS-thread and must cover the
		// SetParent call: pin the goroutine to this thread.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		if procSetThreadDpiHostingBehavior.Find() == nil {
			prev, _, _ := procSetThreadDpiHostingBehavior.Call(dpiHostingBehaviorMixed)
			log.Printf("dpi: embed-thread hosting(MIXED) prev=%d (-1 = call failed)", int32(prev))
		}
		dpiAwarenessOf("parent", parent)
		dpiAwarenessOf("child", uintptr(child))
		log.Printf("child class=%q", className(uintptr(child)))

		// Adopt first, restyle after — the original mRemoteNG order for
		// PuTTY embedding; some windows refuse SetParent once WS_CHILD is
		// applied while still top-level. SetParent legitimately returns
		// NULL when the previous parent was NULL; only a set last-error
		// means failure (clear it first).
		procSetLastError.Call(0)
		ret, _, errno := procSetParent.Call(uintptr(child), uintptr(parent))
		log.Printf("SetParent ret=0x%x errno=%v", ret, errno)
		if ret == 0 && errno != windows.ERROR_SUCCESS {
			return fmt.Errorf("SetParent: %v", errno)
		}
		// Trust nothing: verify the child actually hangs under us now
		// (SetParent can no-op without an error, e.g. DPI-context refusal).
		// One retry after a beat: some clients re-assert their window right
		// after startup.
		verified := false
		for attempt := 1; attempt <= 2; attempt++ {
			got, _, _ := procGetAncestor.Call(uintptr(child), gaParent)
			if got == uintptr(parent) {
				verified = true
				break
			}
			log.Printf("verify attempt %d: child parent=0x%x want 0x%x, class now %q", attempt, got, parent, className(uintptr(child)))
			if attempt == 1 {
				time.Sleep(300 * time.Millisecond)
				procSetLastError.Call(0)
				ret, _, errno := procSetParent.Call(uintptr(child), uintptr(parent))
				log.Printf("SetParent retry ret=0x%x errno=%v", ret, errno)
			}
		}
		if !verified {
			return fmt.Errorf("SetParent did not take effect after retry (child 0x%x class %q)", uintptr(child), className(uintptr(child)))
		}
		// Now strip the frame and mark as child: a top-level style keeps WM
		// behaviors (own taskbar entry, move by caption) that break
		// embedding. FRAMECHANGED makes the style change take effect.
		style, _, _ := procGetWindowLongPtrW.Call(uintptr(child), uintptr(gwlStyle))
		newStyle := style&^uintptr(wsPopup|wsCaption|wsThickframe) | wsChild
		if ret, _, err := procSetWindowLongPtrW.Call(uintptr(child), uintptr(gwlStyle), newStyle); ret == 0 && err != windows.ERROR_SUCCESS {
			return fmt.Errorf("SetWindowLongPtr: %v", err)
		}
		procSetWindowPos.Call(uintptr(child), 0, 0, 0, 0, 0,
			swpNoMove|swpNoSize|swpNoZorder|swpFramechange)
	}
	return e.resizeToParent()
}

// findTopLevelByPID polls visible top-level windows until one belongs to
// pid, the process exits, or the timeout expires. In parent-window mode the
// window is a child of ours, but EnumWindows only lists top-levels — for
// that mode wfreerdp's window is found via the same PID poll once mapped,
// so both modes share this lookup.
func findTopLevelByPID(pid uint32, timeout time.Duration, exited <-chan error) (windows.HWND, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case werr := <-exited:
			return 0, fmt.Errorf("client exited before mapping a window (%v) — see its output in the spike log", werr)
		default:
		}
		var found windows.HWND
		cb := windows.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
			var wpid uint32
			procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&wpid)))
			if wpid == pid {
				if vis, _, _ := procIsWindowVisible.Call(hwnd); vis != 0 {
					// Skip standard dialogs (class #32770): mstsc shows
					// trust/credential prompts before the session window
					// exists; adopting one leaves the real session outside.
					var cls [64]uint16
					n, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&cls[0])), 64)
					if windows.UTF16ToString(cls[:n]) == "#32770" {
						return 1
					}
					found = windows.HWND(hwnd)
					return 0 // stop enumeration
				}
			}
			return 1 // continue
		})
		procEnumWindows.Call(cb, 0)
		if found != 0 {
			return found, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return 0, fmt.Errorf("no visible window for pid %d after %s", pid, timeout)
}

func (e *winEmbedder) resizeToParent() error {
	var r rect
	if ret, _, err := procGetClientRect.Call(uintptr(e.parent), uintptr(unsafe.Pointer(&r))); ret == 0 {
		return fmt.Errorf("GetClientRect: %v", err)
	}
	h := int(r.bottom) - e.topOffset
	if h < 1 {
		h = 1
	}
	if ret, _, err := procMoveWindow.Call(uintptr(e.child), 0, uintptr(e.topOffset),
		uintptr(int(r.right)), uintptr(h), 1); ret == 0 {
		return fmt.Errorf("MoveWindow: %v", err)
	}
	return nil
}

func (e *winEmbedder) watchAndResize(onChildGone func()) {
	for {
		time.Sleep(200 * time.Millisecond)
		if alive, _, _ := procIsWindow.Call(uintptr(e.child)); alive == 0 {
			onChildGone()
			return
		}
		_ = e.resizeToParent()
	}
}

func (e *winEmbedder) killChild() {
	procPostMessageW.Call(uintptr(e.child), wmClose, 0, 0)
}

func (e *winEmbedder) close() {}
