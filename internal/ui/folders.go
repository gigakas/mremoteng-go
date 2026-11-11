package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

// withinFolder reports whether path is folder or lives below it.
func withinFolder(path, folder string) bool {
	if folder == "" {
		return path == ""
	}
	return path == folder || strings.HasPrefix(path, folder+"/")
}

// replaceFolderPrefix rewrites path when it is inside old.
func replaceFolderPrefix(path, old, new string) string {
	if new == "" {
		return strings.TrimPrefix(strings.TrimPrefix(path, old), "/")
	}
	return new + strings.TrimPrefix(path, old)
}

// folderSelection returns the selected folder path, if a folder is selected.
func (s *screen) folderSelection() (string, bool) {
	if s.selectedUID == "" || !isFolderUID(s.selectedUID) {
		return "", false
	}
	return s.index.folderPath(s.selectedUID), true
}

// requestNewSubfolder creates a folder inside parent.
func (s *screen) requestNewSubfolder(parent string) {
	entry := widget.NewEntry()
	entry.SetText(parent + "/")
	s.showTextDialog("New folder", "Create", entry, func() { s.addFolder(entry.Text) })
}

// requestRenameFolder asks for the new path of the selected folder.
func (s *screen) requestRenameFolder() {
	current, ok := s.folderSelection()
	if !ok {
		s.setStatus("Select a folder in the tree to rename it.")
		return
	}
	entry := widget.NewEntry()
	entry.SetText(current)
	s.showTextDialog("Rename folder", "Rename", entry, func() { s.renameFolder(current, entry.Text) })
}

// textDialogWidth keeps these dialogs compact: the standard Fyne form dialog
// grows with its content and can overflow, while a fixed width never scrolls.
const (
	textDialogWidth  = 440
	textDialogHeight = 130
)

// showTextDialog asks for a single line of text in a fixed-size popup.
func (s *screen) showTextDialog(title, confirm string, entry *widget.Entry, onConfirm func()) {
	var popup *widget.PopUp
	content := container.NewVBox(
		widget.NewLabel(title),
		entry,
		container.NewHBox(
			widget.NewButton(confirm, func() {
				popup.Hide()
				onConfirm()
			}),
			widget.NewButton("Cancel", func() { popup.Hide() }),
		),
	)
	popup = s.showModal(content, fyne.NewSize(textDialogWidth, textDialogHeight))
}

// requestFolderDefaults edits the values a folder shares with its descendants.
func (s *screen) requestFolderDefaults(path string) {
	username := widget.NewEntry()
	port := widget.NewEntry()
	for _, item := range s.defaults {
		if item.Path == path {
			username.SetText(item.Username)
			if item.Port != 0 {
				port.SetText(strconv.Itoa(item.Port))
			}
		}
	}
	username.SetPlaceHolder("Shared username")
	port.SetPlaceHolder("Shared port")
	var popup *widget.PopUp
	content := container.NewVBox(
		widget.NewLabelWithStyle("Folder defaults: "+path, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		username,
		port,
		container.NewHBox(
			widget.NewButton("Save", func() {
				popup.Hide()
				s.saveFolderDefaults(path, username.Text, port.Text)
			}),
			widget.NewButton("Cancel", func() { popup.Hide() }),
		),
	)
	popup = s.showModal(content, fyne.NewSize(textDialogWidth, 190))
}

// saveFolderDefaults persists the shared values for one folder.
func (s *screen) saveFolderDefaults(path, username, portText string) {
	if !s.requireWritable("Editing folder defaults") {
		return
	}
	defaults := make([]profiles.FolderDefault, 0, len(s.defaults)+1)
	for _, item := range s.defaults {
		if item.Path != path {
			defaults = append(defaults, item)
		}
	}
	item := profiles.FolderDefault{Path: path, Username: strings.TrimSpace(username)}
	if portText != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(portText))
		if err != nil {
			s.setProblem("Folder default port must be a number.")
			return
		}
		item.Port = parsed
	}
	if item.Username != "" || item.Port != 0 {
		defaults = append(defaults, item)
	}
	if err := s.store.Save(s.documentWithDefaults(s.all, s.folders, defaults)); err != nil {
		s.setProblem("Could not save the folder defaults: " + err.Error())
		return
	}
	s.defaults = defaults
	if s.selected != "" {
		if profile, known := s.profileByID(s.selected); known {
			s.populate(profile)
		}
	}
	s.setStatus("Folder defaults saved for " + path + ".")
}

// renameFolder moves folder (and everything below it) to a new path.
func (s *screen) renameFolder(old, raw string) {
	if !s.requireWritable("Renaming folders") {
		return
	}
	folder := strings.Trim(strings.TrimSpace(raw), "/")
	segments, err := profiles.FolderSegments(folder)
	if err != nil || len(segments) == 0 {
		s.setProblem("Cannot rename folder: use names like Production or Production/Web.")
		return
	}
	if folder == old {
		return
	}
	if withinFolder(folder, old) {
		s.setProblem("Cannot rename " + old + " into itself.")
		return
	}
	for _, existing := range s.folders {
		if existing == folder || strings.HasPrefix(existing, folder+"/") && !withinFolder(existing, old) {
			s.setProblem("Cannot rename folder: " + folder + " already exists.")
			return
		}
	}

	nextFolders := make([]string, 0, len(s.folders))
	seen := map[string]bool{}
	for _, existing := range s.folders {
		renamed := existing
		if withinFolder(existing, old) {
			renamed = replaceFolderPrefix(existing, old, folder)
		}
		if !seen[renamed] {
			seen[renamed] = true
			nextFolders = append(nextFolders, renamed)
		}
	}
	nextProfiles := make([]profiles.Profile, len(s.all))
	copy(nextProfiles, s.all)
	for i, profile := range nextProfiles {
		if withinFolder(profile.Folder, old) {
			nextProfiles[i].Folder = replaceFolderPrefix(profile.Folder, old, folder)
		}
	}
	if err := s.store.Save(s.document(nextProfiles, nextFolders)); err != nil {
		s.setProblem("Could not rename folder: " + err.Error())
		return
	}
	s.all = nextProfiles
	s.folders = nextFolders
	if s.selected != "" {
		if profile, known := s.profileByID(s.selected); known {
			s.baseline = profile
		}
	}
	s.refreshTree()
	s.selectUID(folderUID(folder))
	s.setStatus("Folder renamed to " + folder + ".")
}

// requestDeleteFolder confirms removing the selected folder.
func (s *screen) requestDeleteFolder() {
	current, ok := s.folderSelection()
	if !ok {
		s.setStatus("Select a folder in the tree to delete it.")
		return
	}
	affected := 0
	for _, profile := range s.all {
		if withinFolder(profile.Folder, current) {
			affected++
		}
	}
	message := fmt.Sprintf("Delete folder %q?", current)
	if affected > 0 {
		message = fmt.Sprintf("Delete folder %q and its %d connection(s)? This cannot be undone.", current, affected)
	}
	s.confirm("Delete folder", message, func() { s.deleteFolder(current) }, nil)
}

func (s *screen) deleteFolder(folder string) {
	if !s.requireWritable("Deleting folders") {
		return
	}
	nextProfiles := make([]profiles.Profile, 0, len(s.all))
	for _, profile := range s.all {
		if withinFolder(profile.Folder, folder) {
			s.closeSessionTab(profile.ID)
			continue
		}
		nextProfiles = append(nextProfiles, profile)
	}
	nextFolders := make([]string, 0, len(s.folders))
	for _, existing := range s.folders {
		if withinFolder(existing, folder) {
			continue
		}
		nextFolders = append(nextFolders, existing)
	}
	if err := s.store.Save(s.document(nextProfiles, nextFolders)); err != nil {
		s.setProblem("Could not delete folder: " + err.Error())
		return
	}
	s.all = nextProfiles
	s.folders = nextFolders
	s.resetEditor()
	s.refreshTree()
	s.setStatus("Folder " + folder + " deleted.")
}

// requestRenameConnection edits the connection name with the same mouse-first
// dialog used for folders.
func (s *screen) requestRenameConnection() {
	profile, known := s.profileByID(s.selected)
	if !known {
		s.setStatus("Select a connection in the tree to rename it.")
		return
	}
	entry := widget.NewEntry()
	entry.SetText(profile.Name)
	s.showTextDialog("Rename connection", "Rename", entry, func() {
		s.renameConnection(profile, entry.Text)
	})
}

// renameConnection applies a new connection name and refreshes tree and tab.
func (s *screen) renameConnection(profile profiles.Profile, raw string) {
	if !s.requireWritable("Renaming connections") {
		return
	}
	updated := profile
	updated.Name = strings.TrimSpace(raw)
	if err := updated.Validate(); err != nil {
		s.setProblem("Cannot rename connection: " + err.Error())
		return
	}
	next := s.upsert(updated)
	if err := s.store.Save(s.document(next, s.folders)); err != nil {
		s.setProblem("Could not rename connection: " + err.Error())
		return
	}
	s.all = next
	if tab, open := s.openTabs[profile.ID]; open {
		tab.Text = updated.Name
		s.tabs.Refresh()
	}
	if s.selected == profile.ID {
		s.populate(updated)
	}
	s.refreshTree()
	s.selectUID(connectionUID(profile.ID))
	s.setStatus("Renamed connection to " + updated.Name + ".")
}

// moveConnection places profile in folder; an empty path moves it to the root.
func (s *screen) moveConnection(profile profiles.Profile, raw string) {
	if !s.requireWritable("Moving connections") {
		return
	}
	folder := strings.Trim(strings.TrimSpace(raw), "/")
	if folder != "" {
		if segments, err := profiles.FolderSegments(folder); err != nil || len(segments) == 0 {
			s.setProblem("Cannot move the connection: use names like Production or Production/Web.")
			return
		}
	}
	updated := profile
	updated.Folder = folder
	next := s.upsert(updated)
	folders := s.folders
	if folder != "" {
		folders = addFolderPath(s.folders, folder)
	}
	if err := s.store.Save(s.document(next, folders)); err != nil {
		s.setProblem("Could not move the connection: " + err.Error())
		return
	}
	s.all = next
	s.folders = folders
	if s.selected == profile.ID {
		s.populate(updated)
	}
	s.refreshTree()
	s.selectUID(connectionUID(profile.ID))
	if folder == "" {
		s.setStatus("Moved " + profile.Name + " to the root.")
		return
	}
	s.setStatus("Moved " + profile.Name + " to " + folder + ".")
}

// closeSessionTab closes and forgets the tab of a removed connection.
func (s *screen) closeSessionTab(id string) {
	if tab, open := s.openTabs[id]; open {
		delete(s.openTabs, id)
		delete(s.panels, id)
		s.tabs.Remove(tab)
	}
	if s.selected == id {
		s.resetEditor()
	}
}
