//go:build windows

package rdp

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/mRemoteNG/mremoteng-go/internal/protocol/winembed"
)

// rdpClientExe is the Windows FreeRDP client. Per docs/spike-win32.md,
// FreeRDP's GitHub releases no longer ship wfreerdp.exe; the nightly CI
// publishes sdl-freerdp.exe (SDL client, window class "SDL_app"), which is
// the validated Windows target. It's a runtime dependency, never bundled
// (see the blueprint's non-negotiable "never link libfreerdp").
const rdpClientExe = "sdl-freerdp.exe"

func (s *Session) connectPlatform(ctx context.Context) error {
	path, err := exec.LookPath(rdpClientExe)
	if err != nil {
		return fmt.Errorf("rdp: %s not found on PATH (install FreeRDP; see docs/spike-win32.md's packaging note): %w", rdpClientExe, err)
	}

	cmd := exec.Command(path, s.buildArgs()...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("rdp: start %s: %w", rdpClientExe, err)
	}

	hwnd, err := winembed.FindAndAdopt(ctx, uint32(cmd.Process.Pid), winembed.DefaultDeadline, winembed.DefaultPollInterval, winembed.DialogClassName)
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return fmt.Errorf("rdp: locate session window: %w", err)
	}

	s.mu.Lock()
	s.cmd = cmd
	s.hwnd = hwnd
	s.mu.Unlock()

	s.waitDone = make(chan struct{})
	go func() {
		cmd.Wait()
		close(s.waitDone)
		s.FireClose()
	}()

	return nil
}
