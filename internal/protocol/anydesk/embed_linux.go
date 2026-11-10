//go:build linux

package anydesk

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

// clientExe is the Linux AnyDesk client binary name.
const clientExe = "anydesk"

// connectPlatform launches the AnyDesk client and tracks its lifecycle.
//
// Window discovery/embedding is NOT implemented on this platform, for the
// same reason as RDP's Linux path (stage 2.5): no X server, no AnyDesk
// binary, no way to validate xgb-based EWMH-parsing code in this
// Windows-only development session. NativeWindowHandle() returns 0 rather
// than shipping unverified code; see the stage audit's pending actions.
// Linux AnyDesk has no /parent-window-style flag documented (unlike
// xfreerdp), so even with X11 access this would need the generic
// reparent-after-launch approach the spike explicitly flagged as
// "unfinished on purpose" (docs/spike-x11.md) — a real gap, not just an
// environment limitation, for whoever picks this up next.
func (s *Session) connectPlatform(ctx context.Context) error {
	path, err := exec.LookPath(clientExe)
	if err != nil {
		return fmt.Errorf("anydesk: %s not found on PATH (install the AnyDesk client): %w", clientExe, err)
	}

	cmd := exec.Command(path, s.buildArgs()...)

	var stdin io.WriteCloser
	if s.password != "" {
		stdin, err = cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("anydesk: stdin pipe: %w", err)
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("anydesk: start %s: %w", clientExe, err)
	}

	if stdin != nil {
		if _, err := io.WriteString(stdin, s.password+"\n"); err != nil {
			cmd.Process.Kill()
			cmd.Wait()
			return fmt.Errorf("anydesk: write password to stdin: %w", err)
		}
		stdin.Close()
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
