// Package rdp implements protocol.WindowProtocol for RDP by launching an
// external FreeRDP client (sdl-freerdp.exe on Windows, xfreerdp on Linux)
// and exposing its session window for the UI to embed by reparenting —
// never linking libfreerdp (GPLv2 vs Apache-2.0), per the blueprint's
// non-negotiable architectural principle. The reparenting mechanism itself
// follows the recipe validated by the Phase 0 spike (docs/spike-win32.md,
// docs/spike-x11.md); the platform-specific halves live in
// embed_windows.go and embed_linux.go.
package rdp

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"sync"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
)

func init() {
	protocol.Register(connection.ProtocolRDP, New)
}

// Session is an RDP session backed by an external FreeRDP client process.
//
// v1 scope, per the blueprint: single-monitor, no device redirection
// (disks/printers/clipboard) — those are controlled only via the CLI
// flags FreeRDP itself exposes, none of which are wired up here yet.
type Session struct {
	protocol.Lifecycle
	host     string
	port     int
	username string
	password string
	domain   string

	mu        sync.Mutex
	cmd       *exec.Cmd
	hwnd      uintptr
	waitDone  chan struct{}
	closeOnce sync.Once
}

// New builds a Session for info. It implements protocol.Constructor.
func New(_ *connection.ConnectionInfo, values connection.ConnectionValues) (protocol.Protocol, error) {
	if values.Hostname == "" {
		return nil, fmt.Errorf("rdp: hostname is required")
	}
	port := values.Port
	if port <= 0 {
		port = connection.DefaultPort(connection.ProtocolRDP)
	}
	return &Session{
		host:     values.Hostname,
		port:     port,
		username: values.Username,
		password: values.Password,
		domain:   values.Domain,
	}, nil
}

// buildArgs builds the FreeRDP command line. /smart-sizing is the
// universal-fallback resize strategy the spike validated (client-side
// scaling; /dynamic-resolution dropped the session against the xrdp test
// host on both platforms, so it's left as v2/per-host-opt-in). /cert:ignore
// is a v1 shortcut — there is no certificate-trust UI yet (Phase 3), so
// self-signed/unknown RDP host certificates are accepted unconditionally;
// recorded as a pending action in the stage audit, not hidden here.
//
// Known trade-off, also recorded rather than hidden: /p:<password> is
// visible to other processes on the same machine via the process list
// (tasklist/ps) for as long as the FreeRDP process runs. The blueprint
// accepts "CLI flags and .rdp files" as the v1 credential-passing
// mechanism; a temp .rdp file (as the mstsc fallback in the spike used)
// would avoid this specific exposure and is a reasonable v2 follow-up.
func (s *Session) buildArgs() []string {
	args := []string{
		"/v:" + net.JoinHostPort(s.host, strconv.Itoa(s.port)),
		"/smart-sizing",
		"/cert:ignore",
	}
	if s.username != "" {
		args = append(args, "/u:"+s.username)
	}
	if s.domain != "" {
		args = append(args, "/d:"+s.domain)
	}
	if s.password != "" {
		args = append(args, "/p:"+s.password)
	}
	return args
}

// Connect implements protocol.Protocol. The platform-specific half
// (connectPlatform) launches the FreeRDP client and locates its session
// window.
func (s *Session) Connect(ctx context.Context) error {
	return s.connectPlatform(ctx)
}

// Disconnect implements protocol.Protocol: kills the FreeRDP process and
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
// the web backend's Focus (stage 2.3).
func (s *Session) Focus() {}

// Resize implements protocol.Protocol as a no-op: /smart-sizing (see
// buildArgs) scales the FreeRDP client's rendering to whatever size its
// window ends up at, so there is no separate resize message to send —
// resizing the embedded window (a Phase 3 UI operation) is what drives it.
func (s *Session) Resize(width, height int) {}

// NativeWindowHandle implements protocol.WindowProtocol.
func (s *Session) NativeWindowHandle() uintptr {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hwnd
}

var _ protocol.WindowProtocol = (*Session)(nil)
