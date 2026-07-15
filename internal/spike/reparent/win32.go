//go:build spike && windows

package main

import (
	"fmt"
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
)

const (
	wsChild      = 0x40000000
	wsPopup      = 0x80000000
	wsCaption    = 0x00C00000
	wsThickframe = 0x00040000
	wmClose      = 0x0010
)

var gwlStyle = -16 // GWL_STYLE; negative index, hence not a untyped const

type rect struct{ left, top, right, bottom int32 }

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
		// Strip the frame and mark as child before adopting: a top-level
		// window keeps WM behaviors (own taskbar entry, move by caption)
		// that break embedding.
		style, _, _ := procGetWindowLongPtrW.Call(uintptr(child), uintptr(gwlStyle))
		newStyle := style&^uintptr(wsPopup|wsCaption|wsThickframe) | wsChild
		// ret==0 only signals failure when the last error is set (the
		// previous style can legitimately be 0).
		if ret, _, err := procSetWindowLongPtrW.Call(uintptr(child), uintptr(gwlStyle), newStyle); ret == 0 && err != windows.ERROR_SUCCESS {
			return fmt.Errorf("SetWindowLongPtr: %v", err)
		}
		if ret, _, err := procSetParent.Call(uintptr(child), uintptr(parent)); ret == 0 {
			return fmt.Errorf("SetParent: %v", err)
		}
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
