package ssh

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/sftp"
)

// newTestSFTP starts an SFTP server rooted at root on a loopback listener and
// returns a client wired to it. No SSH and no container: the transfer helpers
// only need the SFTP client, so the protocol is exercised in process. A real
// socket is used instead of io.Pipe because the pipes are unbuffered, and the
// client's INIT exchange deadlocks when both sides block on write.
func newTestSFTP(t *testing.T, root string) *sftp.Client {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		server, serverErr := sftp.NewServer(conn, sftp.WithServerWorkingDirectory(root))
		if serverErr != nil {
			_ = conn.Close()
			return
		}
		_ = server.Serve()
		_ = server.Close()
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	client, err := sftp.NewClientPipe(conn, conn)
	if err != nil {
		t.Fatalf("sftp client: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = conn.Close()
		<-serverDone
	})
	return client
}

func TestListDir_DirectoriesFirstThenFilesByName(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"zeta", "alpha"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	for _, file := range []string{"beta.txt", "alpha.txt"} {
		if err := os.WriteFile(filepath.Join(root, file), []byte("content"), 0o644); err != nil {
			t.Fatalf("write %s: %v", file, err)
		}
	}
	client := newTestSFTP(t, root)

	entries, err := listDir(client, ".")
	if err != nil {
		t.Fatalf("listDir: %v", err)
	}
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		got = append(got, entry.Name)
	}
	want := []string{"alpha", "zeta", "alpha.txt", "beta.txt"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("entries = %v, want directories first then files: %v", got, want)
	}
	if !entries[0].IsDir || entries[2].IsDir {
		t.Errorf("entry kinds not reported: %+v", entries)
	}
	if size := entries[2].Size; size != int64(len("content")) {
		t.Errorf("file size = %d, want %d", size, len("content"))
	}
}

func TestListDir_MissingDirectory_ReturnsError(t *testing.T) {
	client := newTestSFTP(t, t.TempDir())

	if _, err := listDir(client, "does-not-exist"); err == nil {
		t.Fatal("listDir reported no error for a missing directory")
	}
}

func TestUploadFile_CopiesContentAndReportsProgress(t *testing.T) {
	root := t.TempDir()
	local := filepath.Join(t.TempDir(), "payload.bin")
	body := strings.Repeat("remote file transfer\n", 512)
	if err := os.WriteFile(local, []byte(body), 0o644); err != nil {
		t.Fatalf("write local: %v", err)
	}
	client := newTestSFTP(t, root)

	var last, total int64
	err := uploadFile(client, local, filepath.Join(root, "up.bin"), func(done, size int64) {
		last, total = done, size
	})
	if err != nil {
		t.Fatalf("uploadFile: %v", err)
	}

	stored, err := os.ReadFile(filepath.Join(root, "up.bin"))
	if err != nil {
		t.Fatalf("read uploaded file: %v", err)
	}
	if string(stored) != body {
		t.Errorf("uploaded content differs: %d bytes vs %d", len(stored), len(body))
	}
	if total != int64(len(body)) || last != total {
		t.Errorf("progress = %d/%d, want %d/%d", last, total, len(body), len(body))
	}
}

func TestDownloadFile_CopiesContentAndReportsProgress(t *testing.T) {
	root := t.TempDir()
	body := strings.Repeat("download me\n", 256)
	if err := os.WriteFile(filepath.Join(root, "down.txt"), []byte(body), 0o644); err != nil {
		t.Fatalf("write remote: %v", err)
	}
	client := newTestSFTP(t, root)
	local := filepath.Join(t.TempDir(), "copy.txt")

	var last, total int64
	err := downloadFile(client, filepath.Join(root, "down.txt"), local, func(done, size int64) {
		last, total = done, size
	})
	if err != nil {
		t.Fatalf("downloadFile: %v", err)
	}

	stored, err := os.ReadFile(local)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(stored) != body {
		t.Errorf("downloaded content differs: %d bytes vs %d", len(stored), len(body))
	}
	if total != int64(len(body)) || last != total {
		t.Errorf("progress = %d/%d, want %d/%d", last, total, len(body), len(body))
	}
}

func TestUploadFile_MissingLocalFile_ReturnsError(t *testing.T) {
	client := newTestSFTP(t, t.TempDir())

	err := uploadFile(client, filepath.Join(t.TempDir(), "absent"), "target.bin", nil)
	if err == nil {
		t.Fatal("uploadFile reported no error for a missing local file")
	}
	if !strings.Contains(err.Error(), "open") {
		t.Errorf("error = %v, want it to name the open failure", err)
	}
}

func TestDownloadFile_MissingRemoteFile_ReturnsError(t *testing.T) {
	client := newTestSFTP(t, t.TempDir())

	err := downloadFile(client, "absent.txt", filepath.Join(t.TempDir(), "out.txt"), nil)
	if err == nil {
		t.Fatal("downloadFile reported no error for a missing remote file")
	}
}

func TestParentDir_StopsAtTheRoot(t *testing.T) {
	cases := map[string]string{
		"/":             "/",
		"/home":         "/",
		"/home/tester":  "/home",
		"/home/tester/": "/home",
		".":             "/",
		"/a/b/c":        "/a/b",
	}
	for input, want := range cases {
		if got := ParentDir(input); got != want {
			t.Errorf("ParentDir(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestJoinRemote_KeepsAbsoluteNames(t *testing.T) {
	if got, want := JoinRemote("/home/tester", "notes.txt"), "/home/tester/notes.txt"; got != want {
		t.Errorf("JoinRemote = %q, want %q", got, want)
	}
	if got, want := JoinRemote("/home/tester", "/etc/hosts"), "/etc/hosts"; got != want {
		t.Errorf("JoinRemote with an absolute name = %q, want %q", got, want)
	}
	if got, want := JoinRemote("/home/tester/", ".."), "/home"; got != want {
		t.Errorf("JoinRemote with .. = %q, want %q", got, want)
	}
}
