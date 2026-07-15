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
	"os/exec"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver"
	"fyne.io/fyne/v2/widget"
)

func main() {
	host := flag.String("host", "127.0.0.1:3389", "RDP host:port")
	user := flag.String("user", "abc", "RDP username")
	pass := flag.String("pass", "abc", "RDP password")
	flag.Parse()

	a := app.New()
	w := a.NewWindow("mremoteng-go spike 0.1 — X11 reparenting")
	w.Resize(fyne.NewSize(1024, 768))

	status := widget.NewLabel("Ready. Connect embeds xfreerdp below.")
	var emb *embedder

	connect := widget.NewButton("Connect", func() {
		if emb != nil {
			status.SetText("Already connected.")
			return
		}
		var parent uintptr
		w.(driver.NativeWindow).RunNative(func(ctx any) {
			if x11, ok := ctx.(driver.X11WindowContext); ok {
				parent = x11.WindowHandle
			}
		})
		if parent == 0 {
			status.SetText("Not running on X11/XWayland — no window handle.")
			return
		}

		cmd := exec.Command("xfreerdp",
			"/v:"+*host, "/u:"+*user, "/p:"+*pass,
			"/cert:ignore", "/size:1024x768")
		if err := cmd.Start(); err != nil {
			status.SetText("xfreerdp failed to start: " + err.Error())
			return
		}
		status.SetText(fmt.Sprintf("xfreerdp started (pid %d), waiting for its window…", cmd.Process.Pid))

		go func() {
			e, err := newEmbedder()
			if err != nil {
				fyne.Do(func() { status.SetText("X11 connect failed: " + err.Error()) })
				_ = cmd.Process.Kill()
				return
			}
			child, err := e.findWindowByPID(uint32(cmd.Process.Pid), 15*time.Second)
			if err != nil {
				fyne.Do(func() { status.SetText("child window not found: " + err.Error()) })
				e.close()
				_ = cmd.Process.Kill()
				return
			}
			if err := e.embed(child, uint32(parent)); err != nil {
				fyne.Do(func() { status.SetText("reparent failed: " + err.Error()) })
				e.close()
				_ = cmd.Process.Kill()
				return
			}
			emb = e
			fyne.Do(func() { status.SetText("Session embedded. Validate: resize, focus in/out, exit.") })

			// Keep the child sized to the Fyne window; report when it dies.
			go e.watchAndResize(func() {
				fyne.Do(func() {
					status.SetText("Session window gone (process exit detected). Panel cleaned up.")
				})
				emb = nil
			})
			_ = cmd.Wait() // reap; window destruction is the UI signal
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
