// Command x11reparent is the Phase 0 spike: prove that an external
// xfreerdp process can be embedded into a Fyne window on X11 by
// reparenting its window with pure-Go xgb (stage 0.1).
//
// Throwaway code: deleted when Phase 0 closes. Never touches libfreerdp
// (GPLv2 restriction) — external process only.
//
// Built only with the "spike" tag so ./scripts/check.sh stays green for
// agents without the Fyne C build deps installed.
//
// Usage:
//
//	go run -tags spike ./internal/spike/x11reparent -host 127.0.0.1:3389 -user abc -pass abc

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
	"fyne.io/fyne/v2/driver"
	"fyne.io/fyne/v2/widget"
	"github.com/BurntSushi/xgb/xproto"
)

func main() {
	host := flag.String("host", "127.0.0.1:3389", "RDP host:port")
	user := flag.String("user", "abc", "RDP username")
	pass := flag.String("pass", "abc", "RDP password")
	mode := flag.String("mode", "parent-window",
		"embedding mode: parent-window (xfreerdp /parent-window flag) or reparent (generic xgb ReparentWindow, the AnyDesk-style fallback)")
	flag.Parse()

	a := app.New()
	w := a.NewWindow("mremoteng-go spike 0.1 — X11 reparenting")
	w.Resize(fyne.NewSize(1024, 768))

	status := widget.NewLabel("Ready. Connect embeds xfreerdp below.")
	setStatus := func(s string) { // log too, so failures are diagnosable
		log.Println("status:", s)
		status.SetText(s)
	}
	var emb *embedder

	connect := widget.NewButton("Connect", func() {
		if emb != nil {
			setStatus("Already connected.")
			return
		}
		var parent uintptr
		w.(driver.NativeWindow).RunNative(func(ctx any) {
			if x11, ok := ctx.(driver.X11WindowContext); ok {
				parent = x11.WindowHandle
			}
		})
		if parent == 0 {
			setStatus("Not running on X11/XWayland — no window handle.")
			return
		}

		args := []string{"/v:" + *host, "/u:" + *user, "/p:" + *pass,
			"/cert:ignore", "/size:1024x768"}
		if *mode == "parent-window" {
			// xfreerdp creates its window as a child of ours from the
			// start: no WM involvement, no race with window re-creation.
			args = append(args, fmt.Sprintf("/parent-window:%d", parent))
		}
		cmd := exec.Command("xfreerdp", args...)
		cmd.Stdout = os.Stdout // xfreerdp output goes to the spike's own log
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			setStatus("xfreerdp failed to start: " + err.Error())
			return
		}
		setStatus(fmt.Sprintf("xfreerdp started (pid %d, mode %s), waiting for its window…", cmd.Process.Pid, *mode))

		// Reap exactly once, whatever happens later; the channel lets the
		// window search abort early if xfreerdp dies first.
		exited := make(chan error, 1)
		go func() { exited <- cmd.Wait() }()

		go func() {
			e, err := newEmbedder()
			if err != nil {
				fyne.Do(func() { setStatus("X11 connect failed: " + err.Error()) })
				_ = cmd.Process.Kill()
				return
			}
			fail := func(what string, err error) {
				fyne.Do(func() { setStatus(what + ": " + err.Error()) })
				e.close()
				_ = cmd.Process.Kill()
			}

			var child xproto.Window
			if *mode == "parent-window" {
				child, err = e.findChildWindow(xproto.Window(parent), 20*time.Second, exited)
				if err != nil {
					fail("child window not found", err)
					return
				}
				err = e.adopt(child, uint32(parent))
			} else {
				child, err = e.findWindowByPID(uint32(cmd.Process.Pid), 20*time.Second, exited)
				if err != nil {
					fail("child window not found", err)
					return
				}
				err = e.embed(child, uint32(parent))
			}
			if err != nil {
				fail("embed failed", err)
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

	w.SetContent(container.NewBorder(
		container.NewHBox(connect, disconnect, status), nil, nil, nil,
		widget.NewLabel(""), // embed area: the child covers the window body
	))
	w.SetOnClosed(func() {
		if emb != nil {
			emb.killChild()
		}
	})

	log.Println("spike 0.1: validate resize, keyboard focus in/out and process-exit cleanup")
	w.ShowAndRun()
}
