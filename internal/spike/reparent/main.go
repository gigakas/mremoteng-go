// Command reparent is the Phase 0 spike: prove that an external FreeRDP
// client process can be embedded into a Fyne window — stage 0.1 on
// Linux/X11 (xgb reparenting, x11.go) and stage 0.2 on Windows/Win32
// (SetParent, win32.go).
//
// Throwaway code: deleted when Phase 0 closes. Never touches libfreerdp
// (GPLv2 restriction) — external process only.
//
// Built only with the "spike" tag so ./scripts/check.sh stays green for
// agents without the Fyne C build deps installed.
//
// Usage:
//
//	go run -tags spike ./internal/spike/reparent -host 127.0.0.1:3389 -user abc -pass abc

//go:build spike

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// sessionEmbedder is what each OS variant provides: locate the session
// window, put it under our parent window, follow resizes, detect its death.
type sessionEmbedder interface {
	setTopOffset(px int)
	// embedSession finds the client's window (strategy depends on mode) and
	// places it inside parent below the toolbar offset.
	embedSession(parent uintptr, pid uint32, mode string, timeout time.Duration, exited <-chan error) error
	// watchAndResize blocks, keeping the child sized to the parent, and
	// calls onChildGone when the session window is destroyed.
	watchAndResize(onChildGone func())
	killChild()
	close()
}

func main() {
	host := flag.String("host", "127.0.0.1:3389", "RDP host:port")
	user := flag.String("user", "abc", "RDP username")
	pass := flag.String("pass", "abc", "RDP password")
	client := flag.String("client", defaultClient, "FreeRDP client executable (name in PATH or full path)")
	mode := flag.String("mode", defaultMode,
		"embedding mode: parent-window (FreeRDP /parent-window flag) or reparent (adopt the client's top-level window, the AnyDesk-style fallback)")
	flag.Parse()

	a := app.New()
	w := a.NewWindow("mremoteng-go spike 0.1/0.2 — window embedding")
	w.Resize(fyne.NewSize(1024, 768))

	status := widget.NewLabel("Ready. Connect embeds the RDP session below.")
	setStatus := func(s string) { // log too, so failures are diagnosable
		log.Println("status:", s)
		status.SetText(s)
	}
	var emb sessionEmbedder
	var topBar *fyne.Container

	connect := widget.NewButton("Connect", func() {
		if emb != nil {
			setStatus("Already connected.")
			return
		}
		parent := parentHandle(w)
		if parent == 0 {
			setStatus("No native window handle available on this backend.")
			return
		}

		args := []string{"/v:" + *host, "/u:" + *user, "/p:" + *pass,
			"/cert:ignore", "/size:1024x768",
			// client-side scaling on resize: unlike /dynamic-resolution it
			// needs no server support (xrdp in the test container chokes on
			// the disp channel and drops the connection)
			"/smart-sizing"}
		if *mode == "parent-window" {
			// the client creates its window as a child of ours from the
			// start: no WM involvement, no race with window re-creation.
			args = append(args, fmt.Sprintf("/parent-window:%d", parent))
		}
		cmd := exec.Command(*client, args...)
		cmd.Stdout = os.Stdout // client output goes to the spike's own log
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			setStatus(*client + " failed to start: " + err.Error())
			return
		}
		setStatus(fmt.Sprintf("%s started (pid %d, mode %s), waiting for its window…", *client, cmd.Process.Pid, *mode))

		// Reap exactly once, whatever happens later; the channel lets the
		// window search abort early if the client dies first.
		exited := make(chan error, 1)
		go func() { exited <- cmd.Wait() }()

		// Native geometry is in physical pixels; Fyne sizes are logical
		// points, so the toolbar height must be scaled (HiDPI displays).
		offsetPx := int(topBar.Size().Height*w.Canvas().Scale()) + 4

		go func() {
			e, err := newSessionEmbedder()
			if err != nil {
				fyne.Do(func() { setStatus("embedder init failed: " + err.Error()) })
				_ = cmd.Process.Kill()
				return
			}
			e.setTopOffset(offsetPx)
			if err := e.embedSession(parent, uint32(cmd.Process.Pid), *mode, 20*time.Second, exited); err != nil {
				fyne.Do(func() { setStatus("embed failed: " + err.Error()) })
				e.close()
				_ = cmd.Process.Kill()
				return
			}
			emb = e
			fyne.Do(func() { setStatus("Session embedded (mode " + *mode + "). Validate: resize, focus in/out, exit.") })

			// Keep the child sized to the Fyne window; report when it dies.
			go e.watchAndResize(func() {
				fyne.Do(func() {
					setStatus("Session window gone (process exit detected). Panel cleaned up.")
				})
				emb = nil
			})
		}()
	})

	disconnect := widget.NewButton("Disconnect", func() {
		if emb == nil {
			status.SetText("Nothing to disconnect.")
			return
		}
		emb.killChild()
		emb = nil
		status.SetText("Disconnected.")
	})

	topBar = container.NewHBox(connect, disconnect, status)
	w.SetContent(container.NewBorder(
		topBar, nil, nil, nil,
		widget.NewLabel(""), // embed area: the child covers the window body
	))
	w.SetOnClosed(func() {
		if emb != nil {
			emb.killChild()
		}
	})

	log.Println("spike: validate resize, keyboard focus in/out and process-exit cleanup")
	w.ShowAndRun()
}
