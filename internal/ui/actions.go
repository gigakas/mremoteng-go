package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

// reload loads the store once at startup and applies it to the widgets. The
// data itself lives in the workspace; a load failure keeps the window readable
// but disables every write so broken data is never overwritten.
func (s *screen) reload() {
	if err := s.ws.load(); err != nil {
		s.setWritable(false)
		s.refreshTree()
		s.resetEditor()
		s.setProblem("Could not read saved connections: " + err.Error() + ". Saving is disabled until the file is fixed.")
		return
	}
	s.setWritable(true)
	s.refreshTree()
	s.suppressBranch = true
	if len(s.ws.profiles) > 0 {
		s.tree.OpenAllBranches()
	}
	s.applyCollapsed()
	s.suppressBranch = false
	s.resetEditor()
	if len(s.ws.profiles) == 0 {
		s.setStatus("No connections yet. Choose File > New connection to create one.")
	}
}

// branchChanged records a folder toggle and persists the workspace state.
func (s *screen) branchChanged(uid string, collapsed bool) {
	if s.index == nil || !isFolderUID(uid) {
		return
	}
	path := s.index.folderPath(uid)
	if collapsed {
		s.ws.collapsed[path] = true
	} else {
		delete(s.ws.collapsed, path)
	}

}

// applyCollapsed closes the folders that were collapsed last time.
func (s *screen) applyCollapsed() {
	for folder := range s.ws.collapsed {
		uid := folderUID(folder)
		if s.indexHas(uid) {
			s.tree.CloseBranch(uid)
		}
	}
}

// persistWorkspace saves profiles, folders, defaults and collapsed state.
func (s *screen) persistWorkspace() {
	if err := s.ws.save(); err != nil {
		s.setProblem("Could not save the workspace state: " + err.Error())
	}
}

func (s *screen) setWritable(writable bool) {
	s.props.setEnabled(writable)
}

func (s *screen) refreshTree() {
	query := strings.ToLower(strings.TrimSpace(s.filter.Text))
	s.index = buildTreeIndex(s.ws.profiles, s.ws.folders, query)
	s.rows = map[string]*treeRow{}
	s.tree.Refresh()
	s.setCounts()
	if s.selectedUID != "" && s.indexHas(s.selectedUID) {
		s.selectUID(s.selectedUID)
		return
	}
	s.suppressTree = true
	s.tree.UnselectAll()
	s.suppressTree = false
}

func (s *screen) treeSelected(uid widget.TreeNodeID) {
	if s.suppressTree {
		return
	}
	if isFolderUID(uid) {
		s.withDiscardGuard(func() { s.selectFolder(uid) })
		return
	}
	profile, known := s.index.profiles[uid]
	if !known || s.selected == profile.ID {
		return
	}
	s.withDiscardGuard(func() { s.openConnection(profile) })
}

// withDiscardGuard runs action, asking first when the Config panel holds
// unsaved edits. Declining restores the tree highlight to the loaded profile.
func (s *screen) withDiscardGuard(action func()) {
	if !s.isDirty() {
		action()
		return
	}
	s.confirm(discardTitle, "The Config panel has unsaved changes. Discard them?", action, func() {
		s.selectUID(s.selectedUID)
	})
}

func (s *screen) selectFolder(uid string) {
	s.selected = ""
	s.selectedUID = uid
	s.populate(profiles.Profile{Protocol: profiles.SSH, Port: profiles.DefaultPort(profiles.SSH)})
	s.setStatus(fmt.Sprintf("Folder %q groups connections whose Folder value is %q. Its contents are edited per connection.",
		s.index.label(uid), s.index.folderPath(uid)))
}

// resolve returns the profile with folder defaults applied.
func (s *screen) resolve(profile profiles.Profile) profiles.Profile {
	return profiles.Resolve(profile, s.ws.defaults)
}

func (s *screen) openConnection(profile profiles.Profile) {
	// Selecting another connection never reopens the Config panel: its
	// visibility belongs to the user (New connection or the settings icon).
	s.selected = profile.ID
	s.selectedUID = connectionUID(profile.ID)
	s.populate(profile)
	s.openSessionTab(profile)
	s.setStatus("Selected " + s.endpointSummary(profile) + ". Choose Connect in the session tab to open a terminal.")
}

func (s *screen) populate(profile profiles.Profile) {
	s.populating = true
	s.baseline = profile
	s.props.populate(profile)
	s.lastProtocol = profile.Protocol
	s.populating = false
	s.refreshInheritance()
}

func (s *screen) requestNew() {
	s.withDiscardGuard(s.startNew)
}

func (s *screen) startNew() {
	s.showProperties()
	s.resetEditor()
}

// resetEditor clears the Config panel without changing its visibility; startup
// uses it so the panel stays hidden until the user asks for it.
func (s *screen) resetEditor() {
	s.selected = ""
	s.selectedUID = ""
	s.suppressTree = true
	s.tree.UnselectAll()
	s.suppressTree = false
	s.populate(profiles.Profile{Protocol: profiles.SSH, Port: profiles.DefaultPort(profiles.SSH)})
	s.setStatus("New connection: fill in the Config panel, then Save.")
}

// refreshInheritance shows the inherited values as placeholders and locks the
// fields that come from folders.
func (s *screen) refreshInheritance() {
	effective := s.resolve(s.rawForm())
	s.props.user.SetPlaceHolder(effective.Username)
	s.props.port.SetPlaceHolder(strconv.Itoa(effective.Port))
	if s.props.inheritUser.Checked {
		s.props.user.Disable()
	} else {
		s.props.user.Enable()
	}
	if s.props.inheritPort.Checked {
		s.props.port.Disable()
	} else {
		s.props.port.Enable()
	}
}

func (s *screen) rawForm() profiles.Profile {
	return s.props.rawForm(s.baseline.ID, profiles.Protocol(s.props.proto.Selected))
}

func (s *screen) formProfile() (profiles.Profile, error) {
	return s.props.formProfile(s.baseline.ID, profiles.Protocol(s.props.proto.Selected))
}

func (s *screen) isDirty() bool {
	return s.rawForm() != s.baseline
}

func (s *screen) save() {
	if !s.requireWritable("Saving") {
		return
	}
	profile, err := s.formProfile()
	if err != nil {
		s.setProblem("Cannot save: " + err.Error())
		return
	}
	if profile.ID == "" {
		id, idErr := profiles.NewID()
		if idErr != nil {
			s.setProblem("Cannot save: " + idErr.Error())
			return
		}
		profile.ID = id
	}
	if err := s.ws.putProfile(profile); err != nil {
		s.setProblem("Could not save: " + err.Error())
		return
	}
	s.selected = profile.ID
	s.selectedUID = connectionUID(profile.ID)
	s.populate(profile)
	s.refreshTree()
	s.selectUID(s.selectedUID)
	s.openSessionTab(profile)
	s.setStatus("Saved " + s.endpointSummary(profile) + ".")
}

func (s *screen) revert() {
	s.populate(s.baseline)
	s.setStatus("Changes reverted.")
}

func (s *screen) requestDelete() {
	if !s.requireWritable("Deleting") {
		return
	}
	target, known := s.profileByID(s.selected)
	if !known {
		s.setStatus("Select a connection in the tree to delete it.")
		return
	}
	s.confirm("Delete connection?", fmt.Sprintf("Delete %q? This cannot be undone.", target.Name), func() {
		s.deleteProfile(target)
	}, nil)
}

func (s *screen) deleteProfile(target profiles.Profile) {
	if err := s.ws.deleteProfile(target.ID); err != nil {
		s.setProblem("Could not delete: " + err.Error())
		return
	}
	if tab, open := s.openTabs[target.ID]; open {
		for _, panel := range s.tabPanels(target.ID) {
			panel.shutdown()
		}
		delete(s.openTabs, target.ID)
		delete(s.panels, target.ID)
		s.tabs.Remove(tab)
	}
	s.startNew()
	s.refreshTree()
	s.setStatus("Deleted " + target.Name + ".")
}

// addFolderPath records a folder and its ancestors so the tree keeps existing
// after a connection moves away.
func addFolderPath(folders []string, folder string) []string {
	segments, err := profiles.FolderSegments(folder)
	if err != nil || len(segments) == 0 {
		return folders
	}
	known := make(map[string]bool, len(folders))
	for _, existing := range folders {
		known[existing] = true
	}
	result := append([]string(nil), folders...)
	path := ""
	for _, segment := range segments {
		if path == "" {
			path = segment
		} else {
			path += "/" + segment
		}
		if !known[path] {
			known[path] = true
			result = append(result, path)
		}
	}
	return result
}

// requestNewFolder asks for a folder path and persists it even while it is
// still empty.
func (s *screen) requestNewFolder() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Production/Web")
	s.showTextDialog("New folder", "Create", entry, func() { s.addFolder(entry.Text) })
}

func (s *screen) addFolder(raw string) {
	if !s.requireWritable("Creating folders") {
		return
	}
	folder := strings.Trim(strings.TrimSpace(raw), "/")
	segments, err := profiles.FolderSegments(folder)
	if err != nil || len(segments) == 0 {
		s.setProblem("Cannot create folder: use names like Production or Production/Web.")
		return
	}
	folders := addFolderPath(s.ws.folders, folder)
	if len(folders) == len(s.ws.folders) {
		s.setStatus("Folder " + folder + " already exists.")
		return
	}
	if err := s.ws.store.Save(s.ws.document(s.ws.profiles, folders)); err != nil {
		s.setProblem("Could not create folder: " + err.Error())
		return
	}
	s.ws.folders = folders
	s.refreshTree()
	// Selecting the new node makes Fyne scroll it into view, which shows the
	// tree's horizontal scrollbar on narrow panes; the status bar reports it
	// instead.
	s.setStatus("Folder " + folder + " created.")
}

func (s *screen) profileByID(id string) (profiles.Profile, bool) {
	for _, profile := range s.ws.profiles {
		if profile.ID == id {
			return profile, true
		}
	}
	return profiles.Profile{}, false
}

func (s *screen) closeRequested() {
	s.withDiscardGuard(s.closeWindow)
}

// closeWindow remembers the geometry before closing the window.
func (s *screen) closeWindow() {
	prefs := s.app.Preferences()
	SaveWindowSize(prefs, s.win.Canvas().Size())
	SaveWindowPosition(s.win, prefs)
	s.persistWorkspace()
	s.shutdownSessions()
	s.win.Close()
}

func (s *screen) setStatus(text string) {
	s.status.Importance = widget.MediumImportance
	s.status.SetText(text)
	s.log.add("info", text)
	s.logConsole.refresh(s)
}

func (s *screen) setProblem(text string) {
	s.status.Importance = widget.DangerImportance
	s.status.SetText(text)
	s.log.add("error", text)
	s.logConsole.refresh(s)
}

// requireWritable reports whether the profile store may be modified. When it
// may not, it explains why in the status bar using the caller's action, so the
// user learns that the loaded file failed to read rather than that a button is
// broken.
func (s *screen) requireWritable(action string) bool {
	if s.ws.writable {
		return true
	}
	s.setProblem(action + " is disabled because the existing file could not be read.")
	return false
}
