package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	sshclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/ssh"
)

// requestRemoteFiles opens the two-pane transfer window for the SSH session the
// user is working in.
func (s *screen) requestRemoteFiles() {
	panel := s.sshPanelForTransfer()
	if panel == nil {
		s.setStatus("Open an SSH session before browsing remote files.")
		return
	}
	options := sshclient.Options{Password: panel.passwordFor(), Timeout: 15 * time.Second}
	s.openRemoteFiles(panel.target(), options, panel.profile.Host)
}

// sshPanelForTransfer prefers the session that owns the keyboard and falls back
// to the selected tab, so the menu works without clicking the terminal first.
func (s *screen) sshPanelForTransfer() *sessionPanel {
	if panel := s.focusedPanel(); panel != nil && panel.profile.Protocol == profiles.SSH {
		return panel
	}
	selected := s.tabs.Selected()
	if selected == nil {
		return nil
	}
	for id, tab := range s.openTabs {
		if tab != selected {
			continue
		}
		for _, panel := range s.tabPanels(id) {
			if panel.profile.Protocol == profiles.SSH {
				return panel
			}
		}
	}
	return nil
}

// openRemoteFiles dials SFTP and shows the browser. An unknown host key asks for
// approval with the same wording as the terminal, instead of failing with a
// status message the user can miss.
func (s *screen) openRemoteFiles(target sshclient.Target, options sshclient.Options, host string) {
	s.setStatus("Connecting to " + host + " for file transfer ...")
	go func() {
		files, _, err := sshclient.OpenRemoteFiles(target, options)
		fyne.Do(func() {
			if err == nil {
				s.showRemoteBrowser(files, host)
				return
			}
			var unknown *sshclient.UnknownHostKeyError
			if errors.As(err, &unknown) {
				s.setProblem(fmt.Sprintf("Host key for %s is not in known_hosts (fingerprint %s).", unknown.HostPort, unknown.Fingerprint))
				s.confirm("Unverified SSH host key",
					fmt.Sprintf("The host key for %s is not in known_hosts.\n\nFingerprint: %s\n\nTrust it for this file transfer?",
						unknown.HostPort, unknown.Fingerprint),
					func() {
						approved := options
						approved.AcceptAnyHostKey = true
						s.openRemoteFiles(target, approved, host)
					}, nil)
				return
			}
			s.setProblem("Could not open the remote files: " + err.Error())
		})
	}()
}

// showRemoteBrowser builds the window once the session is up.
func (s *screen) showRemoteBrowser(files *sshclient.RemoteFiles, host string) {
	browser := &remoteBrowser{
		screen: s,
		files:  files,
		host:   host,
		local:  newFilePane(localSource{}),
		remote: newFilePane(remoteSource{files: files, host: host}),
	}
	browser.show()
}

// remoteBrowser is a two-pane transfer window: the local filesystem on the left,
// the remote host on the right. Each transfer acts on the file selected in the
// pane it comes from and lands in the other pane's current directory, which is
// the FileZilla gesture without the queue, the recursion or the resume.
type remoteBrowser struct {
	screen *screen
	files  *sshclient.RemoteFiles
	host   string
	local  *filePane
	remote *filePane

	status *widget.Label
	popup  *widget.PopUp
}

func (b *remoteBrowser) show() {
	b.status = widget.NewLabel("")
	b.status.Wrapping = fyne.TextWrapWord

	upload := widget.NewButton("Upload to remote  \u2192", b.uploadSelected)
	download := widget.NewButton("\u2190  Download to local", b.downloadSelected)
	closeButton := widget.NewButton("Close", b.close)

	panes := container.NewHSplit(b.local.content(), b.remote.content())
	panes.SetOffset(0.5)

	content := container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), b.status,
			container.NewHBox(upload, download, closeButton)),
		nil, nil,
		panes,
	)
	b.local.onSelect = func(entry fileEntry) { b.describeSelection("Local", entry) }
	b.remote.onSelect = func(entry fileEntry) { b.describeSelection("Remote", entry) }
	b.local.onNewFolder = func() { b.createFolder(b.local, "Local") }
	b.remote.onNewFolder = func() { b.createFolder(b.remote, "Remote") }
	b.local.onDelete = func(entry fileEntry) { b.deleteEntry(b.local, entry, "Local") }
	b.remote.onDelete = func(entry fileEntry) { b.deleteEntry(b.remote, entry, "Remote") }

	b.popup = b.screen.showModal(content, fyne.NewSize(1000, 620))
	b.local.start()
	b.remote.start()
}

// uploadSelected copies the file selected on the left into the remote directory
// shown on the right, keeping its name.
func (b *remoteBrowser) uploadSelected() {
	entry, ok := b.local.selection()
	if !ok {
		b.status.SetText("Select a file or folder on the left to upload it.")
		return
	}
	remotePath := sshclient.JoinRemote(b.remote.dir, entry.Name)
	b.transfer("Uploading "+entry.Name, func(progress sshclient.TransferProgress) error {
		return b.files.Upload(entry.Path, remotePath, progress)
	})
}

// downloadSelected copies the file selected on the right into the local
// directory shown on the left, keeping its name.
func (b *remoteBrowser) downloadSelected() {
	entry, ok := b.remote.selection()
	if !ok {
		b.status.SetText("Select a file or folder on the right to download it.")
		return
	}
	localPath := filepath.Join(b.local.dir, entry.Name)
	b.transfer("Downloading "+entry.Name, func(progress sshclient.TransferProgress) error {
		return b.files.Download(entry.Path, localPath, progress)
	})
}

// transfer runs one copy off the UI thread and reports progress in the dialog
// and the status bar. Both panes are re-listed afterwards, so the copied file
// appears without a manual refresh.
func (b *remoteBrowser) transfer(label string, run func(sshclient.TransferProgress) error) {
	b.status.SetText(label + " ...")
	b.screen.setStatus(label + " ...")
	go func() {
		progress := func(done, total int64) {
			fyne.Do(func() {
				b.status.SetText(fmt.Sprintf("%s: %s of %s", label, formatSize(done), formatSize(total)))
			})
		}
		err := run(progress)
		fyne.Do(func() {
			if err != nil {
				b.status.SetText(label + " failed: " + err.Error())
				b.screen.setProblem(label + " failed: " + err.Error())
				return
			}
			b.status.SetText(label + ": done.")
			b.screen.setStatus(label + ": done.")
			b.local.refresh()
			b.remote.refresh()
		})
	}()
}

// createFolder asks for a name and makes it in the pane's current directory.
func (b *remoteBrowser) createFolder(pane *filePane, side string) {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("new-folder")
	b.screen.showTextDialog("New folder", "Create", entry, func() {
		name := strings.TrimSpace(entry.Text)
		if name == "" {
			return
		}
		target := pane.source.Join(pane.dir, name)
		if err := pane.source.Mkdir(target); err != nil {
			b.status.SetText("Could not create the folder: " + err.Error())
			b.screen.setProblem("Could not create " + side + " folder: " + err.Error())
			return
		}
		pane.refresh()
		b.status.SetText(side + " folder created: " + target)
	})
}

// deleteEntry confirms, then deletes the selected file or folder. A folder goes
// with everything inside it, and the current directory is never a target.
func (b *remoteBrowser) deleteEntry(pane *filePane, entry fileEntry, side string) {
	if entry.Path == pane.dir {
		b.status.SetText("That is the directory being shown; select an entry inside it.")
		return
	}
	what := "file"
	if entry.IsDir {
		what = "folder and everything inside it"
	}
	b.screen.confirm("Delete "+side+" "+what+"?",
		fmt.Sprintf("Delete %q (%s)? This cannot be undone.", entry.Name, what),
		func() {
			if err := pane.source.Remove(entry.Path); err != nil {
				b.status.SetText("Could not delete: " + err.Error())
				b.screen.setProblem("Could not delete " + side + " entry: " + err.Error())
				return
			}
			pane.refresh()
			b.status.SetText("Deleted " + entry.Name + ".")
		}, nil)
}

// describeSelection says what the transfer buttons would move, so a click on a
// row is visibly a selection and not a silent no-op.
func (b *remoteBrowser) describeSelection(side string, entry fileEntry) {
	if entry.Name == "" {
		b.status.SetText("")
		return
	}
	kind := formatSize(entry.Size)
	if entry.IsDir {
		kind = "folder, copied recursively"
	}
	b.status.SetText(fmt.Sprintf("%s selection: %s (%s)", side, entry.Name, kind))
}

func (b *remoteBrowser) close() {
	_ = b.files.Close()
	if b.popup != nil {
		b.popup.Hide()
		b.popup = nil
	}
}

// formatSize renders a byte count for the list and the progress line.
func formatSize(bytes int64) string {
	if bytes < 0 {
		return "?"
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	value := float64(bytes)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", bytes, units[0])
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}

// formatModified renders a modification time as a short date, empty when the
// server did not report one.
func formatModified(when time.Time) string {
	if when.IsZero() {
		return ""
	}
	return when.Format("2006-01-02 15:04")
}
