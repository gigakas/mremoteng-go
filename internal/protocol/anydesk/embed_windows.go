//go:build windows

package anydesk

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/mRemoteNG/mremoteng-go/internal/protocol/winembed"
)

// clientExe is the Windows AnyDesk client binary name.
const clientExe = "AnyDesk.exe"

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
		// Documented AnyDesk --with-password behavior: the password is
		// read from stdin. Not verified against a real client in this
		// environment — see the package doc comment.
		if _, err := io.WriteString(stdin, s.password+"\n"); err != nil {
			cmd.Process.Kill()
			cmd.Wait()
			return fmt.Errorf("anydesk: write password to stdin: %w", err)
		}
		stdin.Close()
	}

	hwnd, err := winembed.FindAndAdopt(ctx, uint32(cmd.Process.Pid), winembed.DefaultDeadline, winembed.DefaultPollInterval, winembed.DialogClassName)
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return fmt.Errorf("anydesk: locate session window: %w", err)
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
