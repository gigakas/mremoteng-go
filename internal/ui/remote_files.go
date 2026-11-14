package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	sshclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/ssh"
)

// requestRemoteFiles opens the single-pane SFTP browser for the SSH session the
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

// showRemoteBrowser builds the dialog once the session is up.
func (s *screen) showRemoteBrowser(files *sshclient.RemoteFiles, host string) {
	startDir, err := files.WorkingDir()
	if err != nil {
		startDir = "/"
	}
	browser := &remoteBrowser{screen: s, files: files, host: host, dir: startDir, selected: -1}
	browser.show()
}

// remoteBrowser is a modal remote directory view with upload and download by
// path. It owns one SFTP session and closes it when dismissed.
type remoteBrowser struct {
	screen   *screen
	files    *sshclient.RemoteFiles
	host     string
	dir      string
	entries  []sshclient.RemoteEntry
	selected int

	pathEntry *widget.Entry
	list      *widget.List
	status    *widget.Label
	popup     *widget.PopUp
}

func (b *remoteBrowser) show() {
	b.pathEntry = widget.NewEntry()
	b.pathEntry.SetText(b.dir)
	b.pathEntry.OnSubmitted = func(text string) { b.navigate(text) }

	up := widget.NewButtonWithIcon("Up", theme.MoveUpIcon(), func() { b.navigate(sshclient.ParentDir(b.dir)) })
	open := widget.NewButton("Open", func() { b.navigate(b.pathEntry.Text) })

	b.list = widget.NewList(b.listLength, b.createRow, b.updateRow)
	b.list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(b.entries) {
			return
		}
		if b.entries[id].IsDir {
			b.navigate(sshclient.JoinRemote(b.dir, b.entries[id].Name))
			return
		}
		b.selected = id
	}
	b.list.OnUnselected = func(widget.ListItemID) { b.selected = -1 }

	b.status = widget.NewLabel("")
	b.status.Wrapping = fyne.TextWrapWord

	upload := widget.NewButtonWithIcon("Upload file here", theme.UploadIcon(), b.uploadHere)
	download := widget.NewButtonWithIcon("Download selected", theme.DownloadIcon(), b.downloadSelected)
	closeButton := widget.NewButton("Close", func() { b.close() })

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Remote files on "+b.host, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewBorder(nil, nil, container.NewHBox(up, open), nil, b.pathEntry),
			widget.NewSeparator(),
		),
		container.NewVBox(widget.NewSeparator(), b.status,
			container.NewHBox(upload, download, closeButton)),
		nil, nil,
		b.list,
	)
	b.popup = b.screen.showModal(content, fyne.NewSize(760, 520))
	b.navigate(b.dir)
}

// navigate lists one directory and keeps the path bar in step.
func (b *remoteBrowser) navigate(dir string) {
	if strings.TrimSpace(dir) == "" {
		dir = b.dir
	}
	entries, err := b.files.List(dir)
	if err != nil {
		b.status.SetText("Could not list the directory: " + err.Error())
		return
	}
	b.dir = dir
	b.entries = entries
	b.selected = -1
	b.pathEntry.SetText(dir)
	b.list.UnselectAll()
	b.list.Refresh()
	b.status.SetText(fmt.Sprintf("%d entries in %s", len(entries), dir))
}

// uploadHere asks for a local file with the platform's own file chooser and
// copies it into the current remote directory. Typing a path was the one step
// that made the feature feel broken.
func (b *remoteBrowser) uploadHere() {
	picker := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		localPath := reader.URI().Path()
		_ = reader.Close()
		b.uploadFrom(localPath)
	}, b.screen.win)
	picker.Show()
}

// uploadFrom copies localPath into the current remote directory, keeping the
// base name, which is what the chooser and a drag-and-drop would both want.
func (b *remoteBrowser) uploadFrom(localPath string) {
	if strings.TrimSpace(localPath) == "" {
		return
	}
	name := filepath.Base(localPath)
	remotePath := sshclient.JoinRemote(b.dir, name)
	b.transfer("Uploading "+name, func(progress sshclient.TransferProgress) error {
		return b.files.Upload(localPath, remotePath, progress)
	})
}

// downloadSelected asks where to save the selected file with the platform's save
// chooser, then copies it there.
func (b *remoteBrowser) downloadSelected() {
	if b.selected < 0 || b.selected >= len(b.entries) {
		b.status.SetText("Select a file in the list to download it.")
		return
	}
	name := b.entries[b.selected].Name
	saver := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		localPath := writer.URI().Path()
		// The save chooser opens the destination; the download truncates and
		// refills it from the remote copy.
		_ = writer.Close()
		b.downloadTo(localPath)
	}, b.screen.win)
	saver.SetFileName(name)
	saver.Show()
}

// downloadTo copies the selected remote file to localPath.
func (b *remoteBrowser) downloadTo(localPath string) {
	if strings.TrimSpace(localPath) == "" || b.selected < 0 || b.selected >= len(b.entries) {
		return
	}
	name := b.entries[b.selected].Name
	remotePath := sshclient.JoinRemote(b.dir, name)
	b.transfer("Downloading "+name, func(progress sshclient.TransferProgress) error {
		return b.files.Download(remotePath, localPath, progress)
	})
}

// transfer runs one copy off the UI thread and reports progress in the dialog
// and the status bar.
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
			b.navigate(b.dir)
		})
	}()
}

func (b *remoteBrowser) close() {
	_ = b.files.Close()
	if b.popup != nil {
		b.popup.Hide()
		b.popup = nil
	}
}

// listLength satisfies widget.List: one row per entry.
func (b *remoteBrowser) listLength() int { return len(b.entries) }

// createRow builds one reusable row. The widget is a typed struct rather than a
// bare container because container.NewBorder keeps its nil slots, so reaching
// into Objects by index panicked as soon as a slot was empty.
func (b *remoteBrowser) createRow() fyne.CanvasObject {
	icon := widget.NewIcon(theme.FileIcon())
	name := widget.NewLabel("")
	size := widget.NewLabel("")
	size.Alignment = fyne.TextAlignTrailing
	date := widget.NewLabel("")
	date.Importance = widget.LowImportance
	box := container.NewBorder(nil, nil, container.NewCenter(icon), nil,
		container.NewBorder(nil, nil, name, container.NewHBox(size, date)))
	return &remoteRow{Container: box, icon: icon, name: name, size: size, date: date}
}

// updateRow fills one row for the entry at id.
func (b *remoteBrowser) updateRow(id widget.ListItemID, object fyne.CanvasObject) {
	row, ok := object.(*remoteRow)
	if !ok || id < 0 || id >= len(b.entries) {
		return
	}
	entry := b.entries[id]
	if entry.IsDir {
		row.icon.SetResource(theme.FolderIcon())
		row.name.SetText(entry.Name + "/")
		row.size.SetText("")
	} else {
		row.icon.SetResource(theme.FileIcon())
		row.name.SetText(entry.Name)
		row.size.SetText(formatSize(entry.Size))
	}
	row.date.SetText(formatModified(entry.Modified))
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

// remoteRow is one listing row: a container plus the widgets the update step
// needs, so nothing is recovered by index.
type remoteRow struct {
	*fyne.Container
	icon *widget.Icon
	name *widget.Label
	size *widget.Label
	date *widget.Label
}
