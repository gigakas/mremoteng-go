<<<<<<< HEAD
// Package rdp launches FreeRDP as an external process. RDP is never linked into
// the application: the client binary is a runtime dependency, credentials reach
// it through stdin (never through argv) and the session window is embedded with
// FreeRDP's /parent-window instead of ad-hoc reparenting.
package rdp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Target is the RDP endpoint.
type Target struct {
	Host     string
	Port     int
	Username string
	Domain   string
}

// Options controls one launch. Nothing here is persisted.
type Options struct {
	// Password is written to FreeRDP's stdin via /from-stdin.
	Password string
	// ParentWindowID embeds the session in that X11 window; zero opens a
	// separate top-level window.
	ParentWindowID uint64
	// Executable overrides discovery (useful for tests).
	Executable string
	// AuthOnly verifies credentials and exits instead of opening a session.
	AuthOnly bool
	// IgnoreCertificate passes /cert:ignore. It is required when a server
	// presents a changed or self-signed certificate and must stay an explicit
	// user choice.
	IgnoreCertificate bool
	// Security selects the RDP security mode ("negotiate", "tls", "rdp"...).
	// Empty keeps the client default. "rdp" is legacy and only for servers
	// without TLS, so the UI asks for it explicitly.
	Security string
	// Width and Height request a session size when not embedded.
	Width  int
	Height int

	// Clipboard, Printers and Audio forward those devices to the session.
	Clipboard bool
	Printers  bool
	Audio     bool
	// Drives are shared as "name:path" entries.
	Drives []string
	// DynamicResolution lets the server drive the resolution. It breaks some
	// servers (xrdp), so it is an explicit choice.
	DynamicResolution bool
	// MultiMonitor spans every local monitor; only meaningful when not embedded.
	MultiMonitor bool
	// Gateway routes the session through an RD Gateway. The gateway password is
	// not sent: FreeRDP prompts for it.
	Gateway Gateway
}

// Gateway describes an RD Gateway hop.
type Gateway struct {
	Host     string
	Username string
	Domain   string
}

// ErrClientMissing means no FreeRDP binary was found.
var ErrClientMissing = errors.New("rdp: no FreeRDP client found (xfreerdp3, xfreerdp)")

// candidateExecutables is ordered from newest to oldest client.
var candidateExecutables = []string{"xfreerdp3", "xfreerdp", "wlfreerdp"}

// FindExecutable returns the FreeRDP client to use.
func FindExecutable() (string, error) {
	for _, name := range candidateExecutables {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", ErrClientMissing
}

// Session is a running FreeRDP process.
type Session struct {
	cmd  *exec.Cmd
	done chan struct{}
	err  error

	mu      sync.Mutex
	output  bytes.Buffer
	started time.Time
}

// Launch starts FreeRDP. The caller owns the process and must Close it.
func Launch(target Target, options Options) (*Session, error) {
	executable := options.Executable
	if executable == "" {
		found, err := FindExecutable()
		if err != nil {
			return nil, err
		}
		executable = found
	}
	args := Arguments(target, options)
	cmd := exec.Command(executable, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("rdp: stdin pipe: %w", err)
	}
	session := &Session{cmd: cmd, done: make(chan struct{}), started: time.Now()}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("rdp: stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("rdp: start %s: %w", executable, err)
	}
	go session.collect(stdout)
	go func() {
		// Credentials are sent once; FreeRDP prompts on its own if it needs more.
		_, _ = io.WriteString(stdin, options.Password+"\n")
		_ = stdin.Close()
		session.err = cmd.Wait()
		close(session.done)
	}()
	return session, nil
}

// collect keeps the client output for diagnostics, bounded to the last 8 KiB.
func (s *Session) collect(r io.Reader) {
	buffer := make([]byte, 4096)
	for {
		n, err := r.Read(buffer)
		if n > 0 {
			s.mu.Lock()
			s.output.Write(buffer[:n])
			if s.output.Len() > 8*1024 {
				trimmed := s.output.Bytes()[s.output.Len()-8*1024:]
				s.output.Reset()
				s.output.Write(trimmed)
			}
			s.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

// Output returns the captured client output.
func (s *Session) Output() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.output.String()
}

// Uptime reports how long the client has been running.
func (s *Session) Uptime() time.Duration {
	if s == nil {
		return 0
	}
	return time.Since(s.started)
}

// Wait blocks until the session ends and reports its error.
func (s *Session) Wait() error {
	<-s.done
	return s.err
}

// Close terminates the session if it is still running.
func (s *Session) Close() error {
	select {
	case <-s.done:
		return nil
	default:
	}
	if s.cmd.Process != nil {
		return s.cmd.Process.Kill()
	}
	return nil
}

// PID reports the process identifier, or zero when the process already exited.
func (s *Session) PID() int {
	if s == nil || s.cmd.Process == nil {
		return 0
	}
	return s.cmd.Process.Pid
}

// Arguments builds the FreeRDP command line. Tests rely on this being free of
// secrets: the password never appears here.
func Arguments(target Target, options Options) []string {
	port := target.Port
	if port == 0 {
		port = 3389
	}
	args := []string{fmt.Sprintf("/v:%s:%d", target.Host, port)}
	if target.Username != "" {
		args = append(args, "/u:"+target.Username)
	}
	if target.Domain != "" {
		args = append(args, "/d:"+target.Domain)
	}
	args = append(args, "/from-stdin:force")
	if options.IgnoreCertificate {
		args = append(args, "/cert:ignore")
	} else {
		args = append(args, "/cert:tofu")
	}
	if options.Security != "" {
		args = append(args, "/sec:"+options.Security)
	}
	if options.AuthOnly {
		args = append(args, "+auth-only")
	}
	if options.Clipboard {
		args = append(args, "+clipboard")
	}
	if options.Printers {
		args = append(args, "/printer")
	}
	if options.Audio {
		args = append(args, "/sound", "/audio-mode:0")
	}
	for _, drive := range options.Drives {
		drive = strings.TrimSpace(drive)
		if drive == "" {
			continue
		}
		args = append(args, "/drive:"+drive)
	}
	if options.DynamicResolution {
		args = append(args, "/dynamic-resolution")
	}
	if options.MultiMonitor {
		args = append(args, "/multimon")
	}
	if options.Gateway.Host != "" {
		args = append(args, "/g:"+options.Gateway.Host)
		if options.Gateway.Username != "" {
			args = append(args, "/gu:"+options.Gateway.Username)
		}
		if options.Gateway.Domain != "" {
			args = append(args, "/gd:"+options.Gateway.Domain)
		}
	}
	if options.ParentWindowID != 0 {
		args = append(args, fmt.Sprintf("/parent-window:%d", options.ParentWindowID), "+smart-sizing")
	} else if options.Width > 0 && options.Height > 0 {
		args = append(args, fmt.Sprintf("/size:%dx%d", options.Width, options.Height))
=======
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
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
	}
	return args
}

<<<<<<< HEAD
// CommandLine renders the argument list for logs and error messages. It never
// contains the password because credentials travel through stdin.
func CommandLine(args []string) string {
	return strings.Join(args, " ")
}

// Summarize reduces FreeRDP output to one bounded, actionable line: the first
// message mentioning a cause, otherwise the first non-empty line.
func Summarize(output string) string {
	interesting := []string{"Message:", "certificate", "ERRCONNECT", "Authentication", "password", "refused"}
	for _, line := range strings.Split(output, "\n") {
		cleaned := strings.TrimSpace(stripLogPrefixes(line))
		if cleaned == "" {
			continue
		}
		for _, marker := range interesting {
			if strings.Contains(cleaned, marker) {
				return clip(cleaned)
			}
		}
	}
	for _, line := range strings.Split(output, "\n") {
		if cleaned := strings.TrimSpace(stripLogPrefixes(line)); cleaned != "" {
			return clip(cleaned)
		}
	}
	return "no diagnostic output"
}

// stripLogPrefixes removes FreeRDP's timestamp and logger prefixes.
func stripLogPrefixes(line string) string {
	line = strings.TrimSpace(line)
	for strings.HasPrefix(line, "[") {
		end := strings.Index(line, "]")
		if end < 0 {
			break
		}
		line = strings.TrimSpace(line[end+1:])
	}
	return line
}

// clip bounds a message to a readable length.
func clip(text string) string {
	const limit = 240
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	runes := []rune(text)
	return string(runes[:limit]) + " …"
}

// AuthTimeout is the default bound for an auth-only probe.
const AuthTimeout = 20 * time.Second

// Probe verifies host reachability and credentials with FreeRDP's auth-only
// mode, which exits without opening a desktop. The returned text is FreeRDP's
// combined output, useful for diagnostics.
func Probe(target Target, options Options) (string, error) {
	executable := options.Executable
	if executable == "" {
		found, err := FindExecutable()
		if err != nil {
			return "", err
		}
		executable = found
	}
	options.AuthOnly = true
	ctx, cancel := context.WithTimeout(context.Background(), AuthTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, executable, Arguments(target, options)...)
	cmd.Stdin = strings.NewReader(options.Password + "\n")
	output, err := cmd.CombinedOutput()
	text := string(output)
	if ctx.Err() == context.DeadlineExceeded {
		return text, errors.New("rdp: authentication timed out")
	}
	if err != nil {
		return text, fmt.Errorf("rdp: authentication failed: %w: %s", err, Summarize(text))
	}
	return text, nil
}
=======
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
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
