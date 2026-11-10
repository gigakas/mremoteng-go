//go:build linux

package rdp

import (
	"context"
	"fmt"
	"os/exec"
)

// rdpClientExe is the Linux FreeRDP client (docs/spike-x11.md). Unlike the
// Windows sdl-freerdp target, xfreerdp supports /parent-window:<xid>,
// letting the session window be created directly as a child — no
// find-and-adopt race, in principle, once a real parent window ID is
// available.
const rdpClientExe = "xfreerdp"

// connectPlatform launches xfreerdp and tracks its lifecycle.
//
// Window discovery/embedding is NOT implemented on this platform yet —
// NativeWindowHandle() returns 0. This was a deliberate scope cut, not an
// oversight: the validated Linux mechanism (docs/spike-x11.md) needs X11
// protocol access (the deleted Phase 0 spike used
// github.com/BurntSushi/xgb, walking _NET_CLIENT_LIST/_NET_WM_PID) that
// this Windows-only development session has no way to exercise — no X
// server, no xfreerdp binary, no way to verify xgb code actually works
// rather than merely compiles. Writing untested EWMH-parsing code and
// calling the platform "done" would be worse than leaving this as an
// honest, explicit gap for whoever picks it up with a real Linux/X11
// environment; see the stage audit's pending actions.
func (s *Session) connectPlatform(ctx context.Context) error {
	path, err := exec.LookPath(rdpClientExe)
	if err != nil {
		return fmt.Errorf("rdp: %s not found on PATH (install FreeRDP): %w", rdpClientExe, err)
	}

	cmd := exec.Command(path, s.buildArgs()...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("rdp: start %s: %w", rdpClientExe, err)
	}

	s.mu.Lock()
	s.cmd = cmd
	s.mu.Unlock()

	s.waitDone = make(chan struct{})
	go func() {
		cmd.Wait()
		close(s.waitDone)
		s.FireClose()
	}()

	return nil
}
