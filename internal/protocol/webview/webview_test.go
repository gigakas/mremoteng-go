package webview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeHelper writes a script that records its arguments and environment, so the launch
// can be checked without GTK.
func fakeHelper(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "launch.log")
	path := filepath.Join(dir, executableName)
	script := "#!/bin/sh\n{ echo \"args=$*\"; echo \"gdk=$GDK_BACKEND\"; } > " + log + "\nsleep 5\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	return path, log
}

func waitForLog(t *testing.T, path string) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			return string(data)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("the helper was not launched: %s is still empty", path)
	return ""
}

func TestLaunch_PassesTheWindowTheURLAndTheX11Backend(t *testing.T) {
	helper, log := fakeHelper(t)

	session, err := Launch(4242, "https://intranet.example.com", Options{Executable: helper})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	recorded := waitForLog(t, log)
	if !strings.Contains(recorded, "-window 4242") || !strings.Contains(recorded, "-url https://intranet.example.com") {
		t.Errorf("launch = %q, want the window and the address", recorded)
	}
	// GTK has to use X11: the window belongs to it, and GtkPlug refuses to embed on
	// the Wayland backend.
	if !strings.Contains(recorded, "gdk=x11") {
		t.Errorf("launch = %q, want GDK_BACKEND=x11 in the helper's environment", recorded)
	}
	if session.PID() == 0 {
		t.Error("the session has no process id")
	}
}

func TestLaunch_RejectsWhatItCannotDo(t *testing.T) {
	if _, err := Launch(0, "https://example.com", Options{}); err == nil {
		t.Error("a zero window id was accepted")
	}
	if _, err := Launch(12, "  ", Options{}); err == nil {
		t.Error("an empty URL was accepted")
	}
	if _, err := Launch(12, "https://example.com", Options{Executable: "/nonexistent/helper"}); err == nil {
		t.Error("a missing helper was accepted")
	}
}

func TestSession_Close_EndsTheHelperAndWaitReportsIt(t *testing.T) {
	helper, _ := fakeHelper(t)
	session, err := Launch(7, "https://example.com", Options{Executable: helper})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	if err := session.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- session.Wait() }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Wait did not return after Close")
	}
	// Closing twice is what a disconnect plus an application exit does.
	if err := session.Close(); err != nil && !strings.Contains(err.Error(), "finished") {
		t.Errorf("second Close = %v, want it to be harmless", err)
	}
}

// TestFindExecutable_FindsTheHelperBesideTheApplication covers the lookup order the
// packaged layout depends on: the helper ships next to the binary.
func TestFindExecutable_FindsTheHelperBesideTheApplication(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, executableName), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !usable(filepath.Join(dir, executableName)) {
		t.Error("the helper beside the application was not considered usable")
	}
	if usable(filepath.Join(dir, "missing")) {
		t.Error("a missing file was considered usable")
	}
	if usable(dir) {
		t.Error("a directory was considered usable")
	}
}

func TestSummarize_PicksTheLastLineAndTrimsTheGtkPrefix(t *testing.T) {
	output := "some noise\n\n(WebKitWebProcess:9): Gtk-WARNING **: 22:00:00.000: cannot open display\n"
	if got := Summarize(output); got != "cannot open display" {
		t.Errorf("Summarize = %q, want the GTK message", got)
	}
	if got := Summarize("   \n\n"); got != "" {
		t.Errorf("Summarize(nothing) = %q, want an empty string", got)
	}
	long := strings.Repeat("x", 400)
	if got := Summarize(long); len([]rune(got)) > 241 {
		t.Errorf("Summarize kept %d characters, want it bounded", len([]rune(got)))
	}
}
