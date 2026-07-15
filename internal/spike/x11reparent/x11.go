//go:build spike

package main

import (
	"fmt"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// embedder owns the xgb connection and the pair of windows involved in the
// reparenting: parent (the Fyne window) and child (xfreerdp's window).
type embedder struct {
	conn   *xgb.Conn
	root   xproto.Window
	parent xproto.Window
	child  xproto.Window
}

func newEmbedder() (*embedder, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("connecting to X server: %w", err)
	}
	setup := xproto.Setup(conn)
	return &embedder{conn: conn, root: setup.DefaultScreen(conn).Root}, nil
}

func (e *embedder) close() { e.conn.Close() }

// atom resolves an atom name; the atom must already exist (onlyIfExists),
// which is always true for the EWMH atoms used here.
func (e *embedder) atom(name string) (xproto.Atom, error) {
	reply, err := xproto.InternAtom(e.conn, true, uint16(len(name)), name).Reply()
	if err != nil {
		return 0, fmt.Errorf("intern atom %s: %w", name, err)
	}
	if reply.Atom == 0 {
		return 0, fmt.Errorf("atom %s does not exist (no EWMH window manager?)", name)
	}
	return reply.Atom, nil
}

// findWindowByPID polls _NET_CLIENT_LIST until a top-level window whose
// _NET_WM_PID matches pid appears, the process exits, or the timeout
// expires. On timeout the error lists the PIDs that were visible, which is
// the diagnostic that matters (is the window there under another PID, or
// not there at all?).
func (e *embedder) findWindowByPID(pid uint32, timeout time.Duration, exited <-chan error) (xproto.Window, error) {
	clientList, err := e.atom("_NET_CLIENT_LIST")
	if err != nil {
		return 0, err
	}
	wmPid, err := e.atom("_NET_WM_PID")
	if err != nil {
		return 0, err
	}
	deadline := time.Now().Add(timeout)
	seen := map[uint32]bool{}
	for time.Now().Before(deadline) {
		select {
		case werr := <-exited:
			return 0, fmt.Errorf("xfreerdp exited before mapping a window (%v) — see its output in the spike log", werr)
		default:
		}
		reply, err := xproto.GetProperty(e.conn, false, e.root, clientList,
			xproto.AtomWindow, 0, 1<<16).Reply()
		if err == nil {
			for i := 0; i+4 <= len(reply.Value); i += 4 {
				win := xproto.Window(xgb.Get32(reply.Value[i:]))
				p, err := xproto.GetProperty(e.conn, false, win, wmPid,
					xproto.AtomCardinal, 0, 1).Reply()
				if err == nil && len(p.Value) >= 4 {
					if xgb.Get32(p.Value) == pid {
						return win, nil
					}
					seen[xgb.Get32(p.Value)] = true
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	pids := make([]uint32, 0, len(seen))
	for p := range seen {
		pids = append(pids, p)
	}
	return 0, fmt.Errorf("no window with _NET_WM_PID=%d after %s (client-list PIDs seen: %v)", pid, timeout, pids)
}

// findChildWindow polls QueryTree until parent has a child window (created
// there by xfreerdp's /parent-window flag), the process exits, or the
// timeout expires.
func (e *embedder) findChildWindow(parent xproto.Window, timeout time.Duration, exited <-chan error) (xproto.Window, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case werr := <-exited:
			return 0, fmt.Errorf("xfreerdp exited before mapping a window (%v) — see its output in the spike log", werr)
		default:
		}
		tree, err := xproto.QueryTree(e.conn, parent).Reply()
		if err == nil && len(tree.Children) > 0 {
			return tree.Children[len(tree.Children)-1], nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return 0, fmt.Errorf("no child window under parent %d after %s", parent, timeout)
}

// embed reparents child into parent at the panel origin and sizes it to the
// parent's current geometry (unmap → reparent → map, so the WM releases it).
// This is the generic mechanism for processes without a /parent-window
// equivalent (AnyDesk).
func (e *embedder) embed(child xproto.Window, parent uint32) error {
	if err := xproto.UnmapWindowChecked(e.conn, child).Check(); err != nil {
		return fmt.Errorf("unmap: %w", err)
	}
	if err := xproto.ReparentWindowChecked(e.conn, child, xproto.Window(parent), 0, embedTopOffset).Check(); err != nil {
		return fmt.Errorf("reparent: %w", err)
	}
	if err := xproto.MapWindowChecked(e.conn, child).Check(); err != nil {
		return fmt.Errorf("map: %w", err)
	}
	return e.adopt(child, parent)
}

// adopt registers an already-parented child, subscribes to the events that
// drive resize-follow and death detection, and does the initial sizing.
func (e *embedder) adopt(child xproto.Window, parent uint32) error {
	e.parent = xproto.Window(parent)
	e.child = child
	// Our own event mask on the parent (per-client in X11, so this does not
	// disturb GLFW's mask): we need its ConfigureNotify to follow resizes
	// and its SubstructureNotify to see the child's destruction.
	if err := xproto.ChangeWindowAttributesChecked(e.conn, e.parent,
		xproto.CwEventMask, []uint32{xproto.EventMaskStructureNotify | xproto.EventMaskSubstructureNotify}).Check(); err != nil {
		return fmt.Errorf("event mask: %w", err)
	}
	return e.resizeToParent()
}

// embedTopOffset leaves the Fyne toolbar row visible above the session.
const embedTopOffset = 40

func (e *embedder) resizeToParent() error {
	geo, err := xproto.GetGeometry(e.conn, xproto.Drawable(e.parent)).Reply()
	if err != nil {
		return fmt.Errorf("parent geometry: %w", err)
	}
	h := int(geo.Height) - embedTopOffset
	if h < 1 {
		h = 1
	}
	return xproto.ConfigureWindowChecked(e.conn, e.child,
		xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
		[]uint32{uint32(geo.Width), uint32(h)}).Check()
}

// watchAndResize loops on X events: parent ConfigureNotify → resize the
// child; child DestroyNotify → run onChildGone and stop. Returns when the
// connection closes or the child dies.
func (e *embedder) watchAndResize(onChildGone func()) {
	for {
		ev, err := e.conn.WaitForEvent()
		if err != nil {
			continue
		}
		if ev == nil {
			return // connection closed
		}
		switch t := ev.(type) {
		case xproto.ConfigureNotifyEvent:
			if t.Window == e.parent {
				_ = e.resizeToParent()
			}
		case xproto.DestroyNotifyEvent:
			if t.Window == e.child {
				onChildGone()
				e.close()
				return
			}
		}
	}
}

// killChild asks the X server to destroy the child window, which makes
// xfreerdp exit; the process reap happens in the cmd.Wait goroutine.
func (e *embedder) killChild() {
	_ = xproto.KillClientChecked(e.conn, uint32(e.child)).Check()
	e.close()
}
