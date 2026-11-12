package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	sshclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/ssh"
)

// requestRemoteFiles opens the single-pane SFTP browser for the SSH session that
// holds the keyboard. The password is the one-off value of that session, so a
// transfer never needs a second prompt.
func (s *screen) requestRemoteFiles() {
	panel := s.focusedPanel()
	if panel == nil || panel.profile.Protocol != profiles.SSH {
		s.setStatus("Open an SSH session, click its terminal, then browse remote files.")
		return
	}
	target := panel.target()
	options := sshclient.Options{Password: panel.passwordFor(), Timeout: 15 * time.Second}
	s.setStatus("Connecting to " + panel.profile.Host + " for file transfer ...")
	go func() {
		files, _, err := sshclient.OpenRemoteFiles(target, options)
		fyne.Do(func() {
			if err != nil {
				s.setProblem("Could not open the remote files: " + err.Error())
				return
			}
			startDir, dirErr := files.WorkingDir()
			if dirErr != nil {
				startDir = "/"
			}
			browser := &remoteBrowser{screen: s, files: files, host: panel.profile.Host, dir: startDir, selected: -1}
			browser.show()
		})
	}()
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

	b.list = widget.NewList(
		func() int { return len(b.entries) },
		func() fyne.CanvasObject {
			return container.NewBorder(nil, nil, widget.NewIcon(theme.FileIcon()), nil,
				container.NewHBox(widget.NewLabel("name"), widget.NewLabel("size"), widget.NewLabel("date")))
		},
		func(id widget.ListItemID, object fyne.CanvasObject) {
			if id < 0 || id >= len(b.entries) {
				return
			}
			entry := b.entries[id]
			row := object.(*fyne.Container)
			icon := row.Objects[0].(*widget.Icon)
			labels := row.Objects[1].(*fyne.Container).Objects
			if entry.IsDir {
				icon.SetResource(theme.FolderIcon())
				labels[0].(*widget.Label).SetText(entry.Name + "/")
				labels[1].(*widget.Label).SetText("")
				labels[2].(*widget.Label).SetText(formatModified(entry.Modified))
				return
			}
			icon.SetResource(theme.FileIcon())
			labels[0].(*widget.Label).SetText(entry.Name)
			labels[1].(*widget.Label).SetText(formatSize(entry.Size))
			labels[2].(*widget.Label).SetText(formatModified(entry.Modified))
		},
	)
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

// uploadHere asks for a local path and copies it into the current directory.
func (b *remoteBrowser) uploadHere() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("/home/you/file.txt")
	b.screen.showTextDialog("Upload a local file", "Upload", entry, func() {
		localPath := strings.TrimSpace(entry.Text)
		if localPath == "" {
			return
		}
		remotePath := sshclient.JoinRemote(b.dir, filepath.Base(localPath))
		b.transfer("Uploading "+filepath.Base(localPath), func(progress sshclient.TransferProgress) error {
			return b.files.Upload(localPath, remotePath, progress)
		})
	})
}

// downloadSelected asks for a local path and copies the selected file to it.
func (b *remoteBrowser) downloadSelected() {
	if b.selected < 0 || b.selected >= len(b.entries) {
		b.status.SetText("Select a file in the list to download it.")
		return
	}
	name := b.entries[b.selected].Name
	entry := widget.NewEntry()
	entry.SetPlaceHolder(filepath.Join("$HOME", name))
	b.screen.showTextDialog("Download "+name, "Download", entry, func() {
		localPath := strings.TrimSpace(entry.Text)
		if localPath == "" {
			return
		}
		remotePath := sshclient.JoinRemote(b.dir, name)
		b.transfer("Downloading "+name, func(progress sshclient.TransferProgress) error {
			return b.files.Download(remotePath, localPath, progress)
		})
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
