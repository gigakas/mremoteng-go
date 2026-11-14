package ssh

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	"github.com/pkg/sftp"
	cryptossh "golang.org/x/crypto/ssh"
)

// RemoteEntry is one item of a remote directory listing.
type RemoteEntry struct {
	Name     string
	Size     int64
	Mode     fs.FileMode
	Modified time.Time
	IsDir    bool
}

// TransferProgress reports how many bytes a transfer has moved and the total
// size when it is known. A non-positive total means the size was unknown.
type TransferProgress func(done, total int64)

// RemoteFiles is an open SFTP session over SSH.
//
// The browser keeps one for the life of its dialog: re-authenticating on every
// directory change would cost a full handshake per click. The caller owns it and
// must Close it.
type RemoteFiles struct {
	client *cryptossh.Client
	sftp   *sftp.Client
}

// OpenRemoteFiles authenticates and starts the SFTP subsystem.
func OpenRemoteFiles(target Target, options Options) (*RemoteFiles, Result, error) {
	client, result, err := connect(target, options)
	if err != nil {
		return nil, Result{}, err
	}
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		_ = client.Close()
		return nil, Result{}, fmt.Errorf("ssh: start sftp: %w", err)
	}
	return &RemoteFiles{client: client, sftp: sftpClient}, result, nil
}

// Close ends the SFTP subsystem and the SSH connection.
func (r *RemoteFiles) Close() error {
	if r == nil {
		return nil
	}
	if r.sftp != nil {
		_ = r.sftp.Close()
	}
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// List returns the entries of one remote directory: subdirectories first, then
// files, each group sorted by name, so the caller only has to keep the path.
func (r *RemoteFiles) List(dir string) ([]RemoteEntry, error) {
	return listDir(r.sftp, dir)
}

// WorkingDir reports the directory the SFTP session starts in, which is the
// remote user's home directory.
func (r *RemoteFiles) WorkingDir() (string, error) {
	dir, err := r.sftp.Getwd()
	if err != nil {
		return "", fmt.Errorf("transfer: working directory: %w", err)
	}
	return dir, nil
}

// Upload copies localPath to remotePath, creating or truncating a file target and
// recreating a directory tree when localPath is a directory. Progress covers
// every byte moved, so a tree reports one running total.
func (r *RemoteFiles) Upload(localPath, remotePath string, progress TransferProgress) error {
	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("transfer: stat %s: %w", localPath, err)
	}
	if !info.IsDir() {
		return uploadFile(r.sftp, localPath, remotePath, progress)
	}
	counter := &byteCounter{total: localTreeSize(localPath), progress: progress}
	_, err = uploadTree(r.sftp, localPath, remotePath, counter, 0)
	return err
}

// Download copies remotePath to localPath, creating or truncating a file target
// and recreating a directory tree when remotePath is a directory.
func (r *RemoteFiles) Download(remotePath, localPath string, progress TransferProgress) error {
	info, err := r.sftp.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("transfer: stat %s: %w", remotePath, err)
	}
	if !info.IsDir() {
		return downloadFile(r.sftp, remotePath, localPath, progress)
	}
	counter := &byteCounter{total: remoteTreeSize(r.sftp, remotePath), progress: progress}
	_, err = downloadTree(r.sftp, remotePath, localPath, counter, 0)
	return err
}

// listDir is the testable core of List: it needs only an SFTP client.
func listDir(client *sftp.Client, dir string) ([]RemoteEntry, error) {
	if dir == "" {
		dir = "."
	}
	infos, err := client.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("transfer: list %s: %w", dir, err)
	}
	entries := make([]RemoteEntry, 0, len(infos))
	for _, info := range infos {
		name := info.Name()
		// The SFTP server may report the "." and ".." entries; the browser draws
		// its own parent row, so they are noise here.
		if name == "." || name == ".." {
			continue
		}
		isDir := info.IsDir()
		if info.Mode()&fs.ModeSymlink != 0 {
			// Follow the link once: a symlink to a directory should be navigable.
			if target, statErr := client.Stat(path.Join(dir, name)); statErr == nil {
				isDir = target.IsDir()
				info = target
			}
		}
		entries = append(entries, RemoteEntry{
			Name:     name,
			Size:     info.Size(),
			Mode:     info.Mode(),
			Modified: info.ModTime(),
			IsDir:    isDir,
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

// uploadFile streams one local file to the remote path.
func uploadFile(client *sftp.Client, localPath, remotePath string, progress TransferProgress) error {
	source, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("transfer: open %s: %w", localPath, err)
	}
	defer func() { _ = source.Close() }()

	info, err := source.Stat()
	if err != nil {
		return fmt.Errorf("transfer: stat %s: %w", localPath, err)
	}

	destination, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("transfer: create %s: %w", remotePath, err)
	}

	writer := io.Writer(destination)
	if progress != nil {
		writer = &progressWriter{writer: destination, total: info.Size(), progress: progress}
	}
	if _, err := io.Copy(writer, source); err != nil {
		_ = destination.Close()
		return fmt.Errorf("transfer: write %s: %w", remotePath, err)
	}
	if err := destination.Close(); err != nil {
		return fmt.Errorf("transfer: close %s: %w", remotePath, err)
	}
	return nil
}

// downloadFile streams one remote file into the local path.
func downloadFile(client *sftp.Client, remotePath, localPath string, progress TransferProgress) error {
	source, err := client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("transfer: open %s: %w", remotePath, err)
	}
	defer func() { _ = source.Close() }()

	info, err := source.Stat()
	if err != nil {
		return fmt.Errorf("transfer: stat %s: %w", remotePath, err)
	}

	destination, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("transfer: create %s: %w", localPath, err)
	}

	writer := io.Writer(destination)
	if progress != nil {
		writer = &progressWriter{writer: destination, total: info.Size(), progress: progress}
	}
	if _, err := io.Copy(writer, source); err != nil {
		_ = destination.Close()
		return fmt.Errorf("transfer: write %s: %w", localPath, err)
	}
	if err := destination.Close(); err != nil {
		return fmt.Errorf("transfer: close %s: %w", localPath, err)
	}
	return nil
}

// byteCounter turns per-file progress into one running total for a tree.
type byteCounter struct {
	total    int64
	progress TransferProgress
}

// wrap reports the progress of one file starting after moved bytes.
func (c *byteCounter) wrap(moved int64) TransferProgress {
	if c.progress == nil {
		return nil
	}
	return func(done, _ int64) { c.progress(moved+done, c.total) }
}

// localTreeSize sums the regular files under dir.
func localTreeSize(dir string) int64 {
	var total int64
	_ = filepath.WalkDir(dir, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if info, infoErr := entry.Info(); infoErr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

// remoteTreeSize sums the regular files under dir over SFTP.
func remoteTreeSize(client *sftp.Client, dir string) int64 {
	infos, err := client.ReadDir(dir)
	if err != nil {
		return 0
	}
	var total int64
	for _, info := range infos {
		if info.IsDir() {
			total += remoteTreeSize(client, path.Join(dir, info.Name()))
			continue
		}
		total += info.Size()
	}
	return total
}

// uploadTree recreates localDir under remoteDir and returns the bytes moved.
func uploadTree(client *sftp.Client, localDir, remoteDir string, counter *byteCounter, moved int64) (int64, error) {
	if err := client.MkdirAll(remoteDir); err != nil {
		return moved, fmt.Errorf("transfer: create directory %s: %w", remoteDir, err)
	}
	items, err := os.ReadDir(localDir)
	if err != nil {
		return moved, fmt.Errorf("transfer: read %s: %w", localDir, err)
	}
	for _, item := range items {
		local := filepath.Join(localDir, item.Name())
		remote := path.Join(remoteDir, item.Name())
		if item.IsDir() {
			moved, err = uploadTree(client, local, remote, counter, moved)
			if err != nil {
				return moved, err
			}
			continue
		}
		var size int64
		if info, infoErr := item.Info(); infoErr == nil {
			size = info.Size()
		}
		if err := uploadFile(client, local, remote, counter.wrap(moved)); err != nil {
			return moved, err
		}
		moved += size
	}
	return moved, nil
}

// downloadTree recreates remoteDir under localDir and returns the bytes moved.
func downloadTree(client *sftp.Client, remoteDir, localDir string, counter *byteCounter, moved int64) (int64, error) {
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return moved, fmt.Errorf("transfer: create directory %s: %w", localDir, err)
	}
	infos, err := client.ReadDir(remoteDir)
	if err != nil {
		return moved, fmt.Errorf("transfer: list %s: %w", remoteDir, err)
	}
	for _, info := range infos {
		name := info.Name()
		if name == "." || name == ".." {
			continue
		}
		remote := path.Join(remoteDir, name)
		local := filepath.Join(localDir, name)
		if info.IsDir() {
			moved, err = downloadTree(client, remote, local, counter, moved)
			if err != nil {
				return moved, err
			}
			continue
		}
		if err := downloadFile(client, remote, local, counter.wrap(moved)); err != nil {
			return moved, err
		}
		moved += info.Size()
	}
	return moved, nil
}

// progressWriter reports the running byte count after every write.
type progressWriter struct {
	writer   io.Writer
	done     int64
	total    int64
	progress TransferProgress
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.done += int64(n)
	w.progress(w.done, w.total)
	return n, err
}

// ParentDir returns the directory above remotePath, stopping at the root.
func ParentDir(remotePath string) string {
	cleaned := path.Clean(remotePath)
	if cleaned == "/" || cleaned == "." {
		return "/"
	}
	parent := path.Dir(cleaned)
	if parent == "." {
		return "/"
	}
	return parent
}

// JoinRemote joins a directory and a name without letting the name escape the
// directory, which is what a listing entry needs when it is opened.
func JoinRemote(dir, name string) string {
	if name == "" || name == "." {
		return path.Clean(dir)
	}
	if path.IsAbs(name) {
		return path.Clean(name)
	}
	return path.Join(dir, name)
}
