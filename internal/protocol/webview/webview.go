// Package webview hosts a WebKit view inside a native window this application owns.
//
// Fyne draws everything into its own window and cannot host a foreign toolkit, and
// WebKitGTK needs a GTK main loop that would have to share the application's. The view
// therefore runs in a helper process: the application creates the X11 window — the same
// way it does for FreeRDP — and the helper embeds a GtkPlug carrying a WebKitWebView
// into it. Nothing GTK or WebKit is linked into the application itself.
package webview

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// executableName is the helper this package looks for.
const executableName = "mremoteng-webview"

// ErrHelperMissing means no helper binary was found, so HTTP sessions have to fall
// back to the system browser.
var ErrHelperMissing = errors.New("webview: no " + executableName + " helper found")

// Options configures one view.
type Options struct {
	// Executable overrides the helper lookup; tests use it.
	Executable string
}

// Session is a running helper process.
type Session struct {
	cmd  *exec.Cmd
	done chan struct{}

	mu     sync.Mutex
	output bytes.Buffer
	err    error
}

// Launch starts the helper for a window id and a URL. The caller owns the window and
// must Close the session.
func Launch(windowID uint64, url string, options Options) (*Session, error) {
	if windowID == 0 {
		return nil, errors.New("webview: the window id is zero")
	}
	if strings.TrimSpace(url) == "" {
		return nil, errors.New("webview: the URL is empty")
	}
	executable := options.Executable
	if executable == "" {
		found, err := FindExecutable()
		if err != nil {
			return nil, err
		}
		executable = found
	}
	cmd := exec.Command(executable, "-window", fmt.Sprintf("%d", windowID), "-url", url)
	// The window belongs to X11 — through XWayland when the desktop is Wayland — so the
	// GTK backend has to be X11 as well: with the Wayland backend GtkPlug refuses to
	// embed and the helper dies.
	cmd.Env = append(os.Environ(), "GDK_BACKEND=x11")
	session := &Session{cmd: cmd, done: make(chan struct{})}
	output, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("webview: stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("webview: start %s: %w", executable, err)
	}
	go session.collect(output)
	go func() {
		session.err = cmd.Wait()
		close(session.done)
	}()
	return session, nil
}

// collect keeps the helper's diagnostics, bounded to the last 8 KiB.
func (s *Session) collect(reader interface{ Read([]byte) (int, error) }) {
	buffer := make([]byte, 2048)
	for {
		count, err := reader.Read(buffer)
		if count > 0 {
			s.mu.Lock()
			s.output.Write(buffer[:count])
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

// Output is what the helper wrote to stderr, for a failure report.
func (s *Session) Output() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.output.String()
}

// Wait blocks until the helper exits.
func (s *Session) Wait() error {
	<-s.done
	return s.err
}

// Close ends the helper if it is still running.
func (s *Session) Close() error {
	if s == nil {
		return nil
	}
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

// PID reports the helper's process id, or zero once it exited.
func (s *Session) PID() int {
	if s == nil || s.cmd.Process == nil {
		return 0
	}
	return s.cmd.Process.Pid
}

// FindExecutable looks for the helper next to the running binary first, so an installed
// or development build uses its own copy, then on the PATH.
func FindExecutable() (string, error) {
	if running, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(running), executableName)
		if usable(candidate) {
			return candidate, nil
		}
	}
	if found, err := exec.LookPath(executableName); err == nil {
		return found, nil
	}
	return "", ErrHelperMissing
}

// usable reports whether path is an executable file.
func usable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

// Summarize reduces the helper's stderr to one line worth showing: the last non-empty
// line, which is where GTK and WebKit put their complaint. An empty log means the view
// was closed normally.
func Summarize(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		cleaned := stripGtkPrefix(strings.TrimSpace(lines[index]))
		if cleaned == "" {
			continue
		}
		if runes := []rune(cleaned); len(runes) > maxSummary {
			cleaned = string(runes[:maxSummary-2]) + " …"
		}
		return cleaned
	}
	return ""
}

// maxSummary bounds the line, including the ellipsis.
const maxSummary = 240

// stripGtkPrefix removes the decoration GTK puts in front of its messages:
// "(process:pid): Gtk-WARNING **: 22:00:00.000: the message".
func stripGtkPrefix(line string) string {
	if strings.HasPrefix(line, "(") {
		if end := strings.Index(line, "): "); end >= 0 {
			line = strings.TrimSpace(line[end+3:])
		}
	}
	for _, prefix := range []string{"Gtk-WARNING **:", "Gtk-CRITICAL **:", "Gtk-Message:", "Gtk-WARNING:"} {
		line = strings.TrimSpace(strings.TrimPrefix(line, prefix))
	}
	// The timestamp is the first field when it is all digits, colons and dots.
	if head, tail, found := strings.Cut(line, ": "); found && looksLikeClock(head) {
		line = strings.TrimSpace(tail)
	}
	return line
}

// looksLikeClock reports the HH:MM:SS(.mmm) shape GTK stamps its messages with.
func looksLikeClock(value string) bool {
	if strings.Count(value, ":") < 2 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && character != ':' && character != '.' {
			return false
		}
	}
	return true
}
