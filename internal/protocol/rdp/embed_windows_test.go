//go:build windows

package rdp

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"testing"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Real Win32 bindings needed only to build a throwaway parent window for
// TestEmbedChild_ReparentsARealExternalWindow — verified against mingw's
// actual windows.h via a small C probe (sizeof/offsetof) before writing
// this, not from memory, given how easy it is to get struct layout wrong.
var (
	kernel32            = windows.NewLazySystemDLL("kernel32.dll")
	procGetModuleHandle = kernel32.NewProc("GetModuleHandleW")

	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procUnregisterClassW = user32.NewProc("UnregisterClassW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
)

// wndClassExW mirrors WNDCLASSEXW (winuser.h); field offsets confirmed
// against mingw's own headers with a throwaway C program before writing
// this (sizeof 80, offsets 0/4/8/16/20/24/32/40/48/56/64/72 on amd64).
type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

func utf16PtrFromString(s string) *uint16 {
	u := utf16.Encode([]rune(s + "\x00"))
	return &u[0]
}

// createTestWindow registers a throwaway window class (WndProc is the
// real native DefWindowProcW, not a Go callback — no need to write one for
// a window that only needs to exist, not pump messages) and creates one
// top-level window from it, with DPI_HOSTING_BEHAVIOR_MIXED set on the
// calling thread first — the exact precondition EmbedChild's doc comment
// requires of whoever creates the parent window. Returns the window and a
// cleanup func.
func createTestWindow(t *testing.T, className string) windows.HWND {
	t.Helper()

	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)

	if err := SetMixedDpiHostingBehavior(); err != nil {
		t.Fatalf("SetMixedDpiHostingBehavior: %v", err)
	}

	hInstance, _, _ := procGetModuleHandle.Call(0)

	class := wndClassExW{
		lpfnWndProc:   procDefWindowProcW.Addr(),
		hInstance:     hInstance,
		lpszClassName: utf16PtrFromString(className),
	}
	class.cbSize = uint32(unsafe.Sizeof(class))

	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&class)))
	if atom == 0 {
		t.Fatalf("RegisterClassExW: %v", err)
	}
	t.Cleanup(func() {
		procUnregisterClassW.Call(uintptr(unsafe.Pointer(class.lpszClassName)), hInstance)
	})

	const wsOverlappedWindow = 0x00CF0000 // WS_OVERLAPPEDWINDOW
	windowName := utf16PtrFromString("mremoteng-go test window")
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(class.lpszClassName)),
		uintptr(unsafe.Pointer(windowName)),
		wsOverlappedWindow,
		0, 0, 200, 200,
		0, 0, hInstance, 0,
	)
	if hwnd == 0 {
		t.Fatalf("CreateWindowExW: %v", err)
	}
	t.Cleanup(func() { procDestroyWindow.Call(hwnd) })

	return windows.HWND(hwnd)
}

// externalTestTarget is the classic Win32 GUI app used as a stand-in
// external process in these tests, the same control-target methodology
// the spike itself used with Notepad. Modern Windows 11's notepad.exe is
// no longer a direct target: it's an MSIX-packaged app whose launcher
// process exits/redirects, so cmd.Process.Pid does not belong to the
// process that actually owns the window (confirmed with
// Get-Process before writing this — MainWindowHandle was 0 on the
// launched pid, non-zero on a second, separate "Notepad" process).
// mspaint.exe was checked the same way and is not redirected.
const externalTestTarget = "mspaint.exe"

func TestFindTopLevelForPID_ExternalProcess(t *testing.T) {
	cmd := exec.Command(externalTestTarget)
	if err := cmd.Start(); err != nil {
		t.Skipf("%s not available in this environment: %v", externalTestTarget, err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hwnd, err := findAndAdopt(ctx, uint32(cmd.Process.Pid))
	if err != nil {
		t.Fatalf("findAndAdopt: %v", err)
	}
	if hwnd == 0 {
		t.Fatal("findAndAdopt returned a zero window handle with no error")
	}

	var gotPID uint32
	if _, err := windows.GetWindowThreadProcessId(windows.HWND(hwnd), &gotPID); err != nil {
		t.Fatalf("GetWindowThreadProcessId: %v", err)
	}
	if int(gotPID) != cmd.Process.Pid {
		t.Errorf("found window belongs to pid %d, want %d", gotPID, cmd.Process.Pid)
	}
}

// TestEmbedChild_ReparentsARealExternalWindow is the real test of the
// spike's validated recipe: a self-created parent window (with
// DPI_HOSTING_BEHAVIOR_MIXED correctly set before creation, exactly as
// EmbedChild's doc comment requires) and a real external process's window
// (see externalTestTarget, standing in for FreeRDP) as the child. If DPI
// awareness weren't handled correctly, SetParent would silently no-op on
// Windows 10+ per the spike's finding #1 — EmbedChild's own GetAncestor
// verification is what would catch that, and this test asserts on the
// error it returns rather than only on SetParent's return value.
func TestEmbedChild_ReparentsARealExternalWindow(t *testing.T) {
	if err := SetProcessMixedDpiAwareness(); err != nil {
		t.Fatalf("SetProcessMixedDpiAwareness: %v", err)
	}

	parent := createTestWindow(t, fmt.Sprintf("mremoteng-go-test-parent-%d", time.Now().UnixNano()))

	cmd := exec.Command(externalTestTarget)
	if err := cmd.Start(); err != nil {
		t.Skipf("%s not available in this environment: %v", externalTestTarget, err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	childHWND, err := findAndAdopt(ctx, uint32(cmd.Process.Pid))
	if err != nil {
		t.Fatalf("findAndAdopt: %v", err)
	}

	if err := EmbedChild(parent, windows.HWND(childHWND)); err != nil {
		t.Fatalf("EmbedChild: %v", err)
	}

	actualParent, _, _ := procGetAncestor.Call(childHWND, gaParent)
	if windows.HWND(actualParent) != parent {
		t.Errorf("GetAncestor after EmbedChild = %x, want parent %x", actualParent, parent)
	}
}
