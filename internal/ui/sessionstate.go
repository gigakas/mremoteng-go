package ui

import (
	"io"
	"sync"

	"fyne.io/fyne/v2"
)

// This file owns the connection state a session tab shows. Each protocol has
// its own lifecycle — SSH keeps a pty, the stream protocols share a socket, VNC
// drives a framebuffer and RDP waits on an external process — so there is no
// common session interface here. What is common is the bookkeeping around them:
// which buttons are usable, what the session area contains, and how a byte
// stream is pumped into the terminal.

// setConnecting disables Connect while a dial is in flight, so the user cannot
// start a second session from the same tab, and reports what is being
// contacted.
func (p *sessionPanel) setConnecting(status string) {
	p.connect.Disable()
	p.setStatus(status)
}

// setConnected flips the session controls between the live and the idle state.
//
// While a session is live the password field stays disabled because it was a
// one-shot input for that session, and the Config panel is hidden so the
// terminal owns the window height. Ending a session reverses both, whatever
// ended it: a user disconnect, a remote close, a failed launch or a crash of
// the child process.
func (p *sessionPanel) setConnected(connected bool) {
	if connected {
		p.connect.Disable()
		p.disconnect.Enable()
		p.password.Disable()
		p.maximiseTerminal()
		return
	}
	p.connect.Enable()
	p.disconnect.Disable()
	p.password.Enable()
	p.restoreConfig()
}

// installSlot replaces the session area with content. The session area is a
// stacked container, so assigning the single object and refreshing it is the
// whole operation; the terminal, the VNC view and the placeholders all go
// through here.
func (p *sessionPanel) installSlot(content fyne.CanvasObject) {
	p.slot.Objects = []fyne.CanvasObject{content}
	p.slot.Refresh()
}

// pumpBytes copies r into apply until it ends, then calls closed once.
//
// Both callbacks run on the Fyne goroutine: the readers run off the UI thread.
// Output that arrives while a delivery is already queued is appended to it, so
// a burst of reads between two frames reaches apply as one write and costs one
// repaint instead of one per read. Order is kept, and closed is queued after
// the last delivery.
func pumpBytes(r io.Reader, apply func([]byte), closed func()) {
	var (
		mu        sync.Mutex
		pending   []byte
		scheduled bool
	)
	deliver := func() {
		mu.Lock()
		data := pending
		pending = nil
		scheduled = false
		mu.Unlock()
		if len(data) > 0 {
			apply(data)
		}
	}
	buffer := make([]byte, 32*1024)
	for {
		n, err := r.Read(buffer)
		if n > 0 {
			// The read buffer is reused across iterations, so it is copied.
			mu.Lock()
			pending = append(pending, buffer[:n]...)
			queue := !scheduled
			scheduled = true
			mu.Unlock()
			if queue {
				runOnUI(deliver)
			}
		}
		if err != nil {
			runOnUI(closed)
			return
		}
	}
}
