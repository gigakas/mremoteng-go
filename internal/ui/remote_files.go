package ui

import (
	"context"
	"errors"
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
		runOnUI(func() {
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
		local:  newFilePane(localSource{}, "Upload to remote  \u2192"),
		remote: newFilePane(remoteSource{files: files, host: host}, "\u2190  Download to local"),
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

	status   *widget.Label
	progress *widget.ProgressBar
	stop     *widget.Button
	cancel   context.CancelFunc
	logBox   *fyne.Container
	log      []string
	popup    *modal
}

func (b *remoteBrowser) show() {
	b.status = widget.NewLabel("")
	b.status.Wrapping = fyne.TextWrapWord
	b.progress = widget.NewProgressBar()
	b.progress.Hide()
	b.stop = widget.NewButtonWithIcon("Stop", theme.CancelIcon(), b.stopTransfer)
	b.stop.Disable()
	b.logBox = container.NewVBox()
	logScroll := container.NewVScroll(b.logBox)
	logScroll.SetMinSize(fyne.NewSize(200, 84))

	closeButton := widget.NewButton("Close", b.close)

	panes := container.NewHSplit(b.local.content(), b.remote.content())
	panes.SetOffset(0.5)

	content := container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(),
			container.NewBorder(nil, nil, nil, b.stop, b.progress),
			logScroll, b.status, container.NewHBox(closeButton)),
		nil, nil,
		panes,
	)
	b.local.onSelect = func(entries []fileEntry) { b.describeSelection("Local", entries) }
	b.remote.onSelect = func(entries []fileEntry) { b.describeSelection("Remote", entries) }
	b.local.onTransfer = b.uploadSelected
	b.remote.onTransfer = b.downloadSelected
	b.local.onNewFolder = func() { b.createFolder(b.local, "Local") }
	b.remote.onNewFolder = func() { b.createFolder(b.remote, "Remote") }
	b.local.onDelete = func(entries []fileEntry) { b.deleteEntries(b.local, entries, "Local") }
	b.remote.onDelete = func(entries []fileEntry) { b.deleteEntries(b.remote, entries, "Remote") }
	b.local.onRename = func(entries []fileEntry) { b.renameEntry(b.local, entries, "Local") }
	b.remote.onRename = func(entries []fileEntry) { b.renameEntry(b.remote, entries, "Remote") }

	b.popup = b.screen.showModal(content, fyne.NewSize(1000, 620))
	b.local.start()
	b.remote.start()
}

// uploadSelected copies the file selected on the left into the remote directory
// shown on the right, keeping its name.
func (b *remoteBrowser) uploadSelected() {
	entries := b.local.selection()
	if len(entries) == 0 {
		b.status.SetText("Select a file or folder on the left to upload it.")
		return
	}
	files, remoteDir := b.files, b.remote.dir
	b.transferMany("Uploading", entries, func(ctx context.Context, entry fileEntry, progress sshclient.TransferProgress) error {
		return files.Upload(ctx, entry.Path, sshclient.JoinRemote(remoteDir, entry.Name), progress)
	})
}

// downloadSelected copies the file selected on the right into the local
// directory shown on the left, keeping its name.
func (b *remoteBrowser) downloadSelected() {
	entries := b.remote.selection()
	if len(entries) == 0 {
		b.status.SetText("Select a file or folder on the right to download it.")
		return
	}
	files, localDir := b.files, b.local.dir
	b.transferMany("Downloading", entries, func(ctx context.Context, entry fileEntry, progress sshclient.TransferProgress) error {
		return files.Download(ctx, entry.Path, filepath.Join(localDir, entry.Name), progress)
	})
}

// transfer runs one copy off the UI thread and reports progress in the dialog
// and the status bar. Both panes are re-listed afterwards, so the copied file
// appears without a manual refresh.
func (b *remoteBrowser) transfer(label string, run func(context.Context, sshclient.TransferProgress) error) {
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	b.startProgress(label)
	go func() {
		defer cancel()
		progress := func(done, total int64) {
			runOnUI(func() {
				b.setProgress(done, total)
				b.status.SetText(progressText(label, done, total))
			})
		}
		err := run(ctx, progress)
		runOnUI(func() {
			b.cancel = nil
			if errors.Is(err, context.Canceled) {
				b.finishProgress()
				b.status.SetText(label + " cancelled.")
				b.screen.setStatus(label + " cancelled.")
				b.logf("warn", label+" cancelled")
				b.local.refresh()
				b.remote.refresh()
				return
			}
			if err != nil {
				b.finishProgress()
				b.status.SetText(label + " failed: " + err.Error())
				b.screen.setProblem(label + " failed: " + err.Error())
				b.logf("error", label+" failed: "+err.Error())
				return
			}
			b.finishProgress()
			b.status.SetText(label + ": done.")
			b.screen.setStatus(label + ": done.")
			b.logf("info", label+": done.")
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

// renameEntry gives one selected entry a new name inside its own directory. One
// field can only name one thing, so a multiple selection is reported instead of
// half-renamed.
func (b *remoteBrowser) renameEntry(pane *filePane, entries []fileEntry, side string) {
	if len(entries) != 1 {
		b.status.SetText("Select one " + strings.ToLower(side) + " entry to rename.")
		return
	}
	entry := entries[0]
	name := widget.NewEntry()
	name.SetText(entry.Name)
	b.screen.showTextDialog("Rename "+side+" entry", "Rename", name, func() {
		next := strings.TrimSpace(name.Text)
		if !validEntryName(next) {
			b.status.SetText("A name cannot be empty, contain a path separator, or be . or ..")
			return
		}
		if next == entry.Name {
			return
		}
		target := pane.source.Join(pane.dir, next)
		if err := pane.source.Rename(entry.Path, target); err != nil {
			b.status.SetText("Could not rename " + entry.Name + ": " + err.Error())
			b.screen.setProblem("Could not rename " + side + " entry: " + err.Error())
			return
		}
		pane.refresh()
		b.status.SetText("Renamed " + entry.Name + " to " + next + ".")
	})
}

// validEntryName rejects what the single-field dialog cannot mean: an empty name,
// a path separator (which would move the entry rather than rename it) and the two
// directory shorthands.
func validEntryName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	return !strings.ContainsAny(name, "/\\\x00")
}

// deleteEntries confirms, then deletes the selected entries. A folder goes with
// everything inside it, and the current directory is never a target.
func (b *remoteBrowser) deleteEntries(pane *filePane, entries []fileEntry, side string) {
	targets := make([]fileEntry, 0, len(entries))
	folders := 0
	for _, entry := range entries {
		if entry.Path == pane.dir {
			continue
		}
		if entry.IsDir {
			folders++
		}
		targets = append(targets, entry)
	}
	if len(targets) == 0 {
		b.status.SetText("That is the directory being shown; select an entry inside it.")
		return
	}
	what := fmt.Sprintf("%d item(s)", len(targets))
	if folders > 0 {
		what = fmt.Sprintf("%d item(s), including %d folder(s) that go with everything inside them", len(targets), folders)
	}
	b.screen.confirm("Delete "+side+" selection?",
		fmt.Sprintf("Delete %s? This cannot be undone.", what),
		func() {
			for _, entry := range targets {
				if err := pane.source.Remove(entry.Path); err != nil {
					b.status.SetText("Could not delete " + entry.Name + ": " + err.Error())
					b.screen.setProblem("Could not delete " + side + " entry: " + err.Error())
					return
				}
			}
			pane.refresh()
			b.status.SetText(fmt.Sprintf("Deleted %d item(s).", len(targets)))
		}, nil)
}

// startProgress shows the bar at zero for a new transfer.
func (b *remoteBrowser) startProgress(label string) {
	if b.progress == nil {
		return
	}
	b.progress.Min = 0
	b.progress.Max = 1
	b.progress.SetValue(0)
	b.progress.Show()
	if b.stop != nil {
		b.stop.Enable()
	}
	b.status.SetText(label + " ...")
	b.screen.setStatus(label + " ...")
	b.logf("info", label+" started")
}

// transferMany copies every selected entry in order, reusing the single-transfer
// machinery so the bar, the Stop button and the log behave the same. A cancel
// stops the queue between entries as well as inside a file.
func (b *remoteBrowser) transferMany(verb string, entries []fileEntry, copyOne func(context.Context, fileEntry, sshclient.TransferProgress) error) {
	label := fmt.Sprintf("%s %d item(s)", verb, len(entries))
	b.transfer(label, func(ctx context.Context, progress sshclient.TransferProgress) error {
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := copyOne(ctx, entry, progress); err != nil {
				return fmt.Errorf("%s: %w", entry.Name, err)
			}
		}
		return nil
	})
}

// stopTransfer cancels the running copy. The button is disabled when idle, so a
// stray press cannot cancel somebody else's transfer.
func (b *remoteBrowser) stopTransfer() {
	if b.cancel == nil {
		return
	}
	b.cancel()
	b.cancel = nil
	b.status.SetText("Stopping ...")
}

// setProgress moves the bar; an unknown total leaves it at zero rather than
// claiming a percentage nobody measured.
func (b *remoteBrowser) setProgress(done, total int64) {
	if b.progress == nil || total <= 0 {
		return
	}
	b.progress.Max = float64(total)
	b.progress.SetValue(float64(done))
}

// finishProgress parks the bar at full so the finished transfer is visible, and
// logs the outcome.
func (b *remoteBrowser) finishProgress() {
	if b.progress == nil {
		return
	}
	if b.stop != nil {
		b.stop.Disable()
	}
	if b.progress.Max <= 0 {
		b.progress.Max = 1
	}
	b.progress.SetValue(b.progress.Max)
}

// logf appends one line to the window log and to the application activity log,
// so a transfer is recorded whether the window is open or not.
func (b *remoteBrowser) logf(kind, text string) {
	line := time.Now().Format("15:04:05") + "  " + kind + "  " + text
	b.log = append(b.log, line)
	if len(b.log) > transferLogLimit {
		b.log = b.log[len(b.log)-transferLogLimit:]
	}
	if b.logBox == nil {
		return
	}
	b.logBox.Objects = b.logBox.Objects[:0]
	for _, entry := range b.log {
		label := widget.NewLabel(entry)
		label.TextStyle = fyne.TextStyle{Monospace: true}
		label.Importance = widget.LowImportance
		b.logBox.Add(label)
	}
	b.logBox.Refresh()
	if b.screen != nil {
		b.screen.log.add(kind, text)
	}
}

// transferLogLimit bounds the window log, like the activity log does.
const transferLogLimit = 100

// progressText is the status line shown under the bar.
func progressText(label string, done, total int64) string {
	if total <= 0 {
		return fmt.Sprintf("%s: %s", label, formatSize(done))
	}
	return fmt.Sprintf("%s: %s of %s (%d%%)", label, formatSize(done), formatSize(total), done*100/total)
}

// describeSelection says what the transfer buttons would move, so a click on a
// row is visibly a selection and not a silent no-op.
func (b *remoteBrowser) describeSelection(side string, entries []fileEntry) {
	switch len(entries) {
	case 0:
		b.status.SetText("")
	case 1:
		entry := entries[0]
		kind := formatSize(entry.Size)
		if entry.IsDir {
			kind = "folder, copied recursively"
		}
		b.status.SetText(fmt.Sprintf("%s selection: %s (%s)", side, entry.Name, kind))
	default:
		folders := 0
		for _, entry := range entries {
			if entry.IsDir {
				folders++
			}
		}
		b.status.SetText(fmt.Sprintf("%s selection: %d items (%d folder(s), %d file(s))",
			side, len(entries), folders, len(entries)-folders))
	}
}

func (b *remoteBrowser) close() {
	if b.cancel != nil {
		b.cancel()
		b.cancel = nil
	}
	files := b.files
	b.files = nil
	if b.popup != nil {
		b.popup.Hide()
		b.popup = nil
	}
	if files == nil {
		return
	}
	// Closing an SSH session writes a disconnect and waits for the socket, which
	// on a remote host is long enough to freeze the window behind it.
	go func() { _ = files.Close() }()
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
