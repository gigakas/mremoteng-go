// Package anydesk implements protocol.WindowProtocol for AnyDesk by
// launching the external, proprietary AnyDesk client and exposing its
// session window for the UI to embed by reparenting — the same
// external-process + reparent pattern as stage 2.5's RDP backend, per the
// blueprint ("Same external-process + reparent pattern as RDP").
//
// Important limitation, stated up front rather than discovered by
// surprise: AnyDesk is not installed in the environment this package was
// written in, and it was deliberately not downloaded and run here even
// though a portable build exists — unlike a compiler toolchain, AnyDesk
// is live remote-access software with its own account/ID/telemetry
// behavior, and installing and running it unattended in an automated
// session is a different category of action than fetching a build tool.
// Everything below follows AnyDesk's documented command-line interface
// (https://support.anydesk.com/knowledge/command-line-interface, as
// recorded in this author's general knowledge) but has not been verified
// against a real AnyDesk.exe. See the stage audit for the full account.
package anydesk

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
)

func init() {
	protocol.Register(connection.ProtocolAnyDesk, New)
}

// Session is an AnyDesk session backed by an external client process.
// clientExe (the platform-specific binary name — "AnyDesk.exe" on
// Windows, "anydesk" on Linux) is defined per-platform in
// embed_windows.go/embed_linux.go, a runtime dependency per the same
// "never link/bundle the external client" principle the blueprint states
// for RDP.
type Session struct {
	protocol.Lifecycle
	address  string
	password string

	mu        sync.Mutex
	cmd       *exec.Cmd
	hwnd      uintptr
	waitDone  chan struct{}
	closeOnce sync.Once
}

// New builds a Session for info. It implements protocol.Constructor. The
// connection model has no AnyDesk-specific fields; like the original
// mRemoteNG's PuTTY-based backends (see internal/protocol/serial's doc
// comment for the same pattern), Hostname carries the AnyDesk address
// (an AnyDesk-ID or alias, not a network hostname).
func New(_ *connection.ConnectionInfo, values connection.ConnectionValues) (protocol.Protocol, error) {
	if values.Hostname == "" {
		return nil, fmt.Errorf("anydesk: address is required (set in the connection's Hostname field)")
	}
	return &Session{
		address:  values.Hostname,
		password: values.Password,
	}, nil
}

// buildArgs builds the AnyDesk command line: launch and connect directly
// to address. --with-password reads a password for unattended access from
// the client's stdin (see Connect) rather than the command line, avoiding
// the process-list password exposure noted as a pending action for RDP
// (stage 2.5) — AnyDesk's documented CLI happens to support this directly
// where FreeRDP's does not.
func (s *Session) buildArgs() []string {
	args := []string{s.address}
	if s.password != "" {
		args = append(args, "--with-password")
	}
	return args
}

// Connect implements protocol.Protocol. The platform-specific half
// (connectPlatform) launches the AnyDesk client and locates its session
// window.
func (s *Session) Connect(ctx context.Context) error {
	return s.connectPlatform(ctx)
}

// Disconnect implements protocol.Protocol: kills the AnyDesk process and
// waits for the exit-watcher goroutine started by connectPlatform to fire
// OnClose, so a caller that gets a nil error back knows the session is
// fully torn down.
func (s *Session) Disconnect() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		cmd := s.cmd
		s.mu.Unlock()
		if cmd != nil && cmd.Process != nil {
			cmd.Process.Kill()
		}
	})
	if s.waitDone != nil {
		<-s.waitDone
	}
	return nil
}

// Focus implements protocol.Protocol as a no-op: once the Phase 3 UI
// embeds NativeWindowHandle into a tab, giving it input focus is a native
// window-management operation the UI performs on the handle, the same as
// the web (stage 2.3) and rdp (stage 2.5) backends' Focus.
func (s *Session) Focus() {}

// Resize implements protocol.Protocol as a no-op: AnyDesk, like FreeRDP's
// smart-sizing mode, scales its own rendering to whatever size its window
// ends up at once embedded.
func (s *Session) Resize(width, height int) {}

// NativeWindowHandle implements protocol.WindowProtocol.
func (s *Session) NativeWindowHandle() uintptr {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hwnd
}

var _ protocol.WindowProtocol = (*Session)(nil)
