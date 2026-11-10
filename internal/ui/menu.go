package ui

import (
	"sort"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

func itoa(value int) string { return strconv.Itoa(value) }

// rowMenu builds the context menu for a tree row.
func (s *screen) rowMenu(uid string) *fyne.Menu {
	if isFolderUID(uid) {
		folder := s.index.folderPath(uid)
		return fyne.NewMenu(folder,
			fyne.NewMenuItem("Folder defaults", func() {
				s.withDiscardGuard(func() { s.requestFolderDefaults(folder) })
			}),
			fyne.NewMenuItem("New folder inside", func() {
				s.withDiscardGuard(func() { s.createFolderInside(folder) })
			}),
			fyne.NewMenuItem("Rename folder", func() {
				s.withDiscardGuard(func() {
					s.focusFolder(uid)
					s.requestRenameFolder()
				})
			}),
			fyne.NewMenuItem("Delete folder", func() {
				s.withDiscardGuard(func() {
					s.focusFolder(uid)
					s.requestDeleteFolder()
				})
			}),
		)
	}
	profile, known := s.index.profiles[uid]
	if !known {
		return fyne.NewMenu("")
	}
	moveTo := fyne.NewMenuItem("Move to", nil)
	moveTo.ChildMenu = s.moveToMenu(profile)
	return fyne.NewMenu(profile.Name,
		fyne.NewMenuItem("Rename", func() {
			s.withDiscardGuard(func() {
				s.focusConnection(profile)
				s.requestRenameConnection()
			})
		}),
		fyne.NewMenuItem("Open", func() {
			s.withDiscardGuard(func() { s.openConnection(profile) })
		}),
		moveTo,
		fyne.NewMenuItem("Delete connection", func() {
			s.withDiscardGuard(func() {
				s.focusConnection(profile)
				s.requestDelete()
			})
		}),
	)
}

// moveToMenu lists the root and every folder, so a connection can be moved with
// the mouse alone.
func (s *screen) moveToMenu(profile profiles.Profile) *fyne.Menu {
	items := []*fyne.MenuItem{
		fyne.NewMenuItem("(root)", func() {
			s.withDiscardGuard(func() { s.moveConnection(profile, "") })
		}),
	}
	folders := append([]string(nil), s.folders...)
	sort.Strings(folders)
	for _, folder := range folders {
		folder := folder
		label := folder
		if profile.Folder == folder {
			label += " (current)"
		}
		items = append(items, fyne.NewMenuItem(label, func() {
			s.withDiscardGuard(func() { s.moveConnection(profile, folder) })
		}))
	}
	return fyne.NewMenu("Move to", items...)
}

// createFolderInside adds a uniquely named folder without asking for a path.
func (s *screen) createFolderInside(parent string) {
	name := "New folder"
	candidate := name
	if parent != "" {
		candidate = parent + "/" + name
	}
	for i := 2; s.folderExists(candidate); i++ {
		candidate = name + " " + itoa(i)
		if parent != "" {
			candidate = parent + "/" + name + " " + itoa(i)
		}
	}
	s.addFolder(candidate)
}

func (s *screen) folderExists(path string) bool {
	for _, folder := range s.folders {
		if folder == path {
			return true
		}
	}
	return false
}

// showRowMenu displays the context menu at the pointer position.
func (s *screen) showRowMenu(uid string, position fyne.Position) {
	canvas := fyne.CurrentApp().Driver().CanvasForObject(s.tree)
	if canvas == nil {
		return
	}
	widget.NewPopUpMenu(s.rowMenu(uid), canvas).ShowAtPosition(position)
}

// rowActivated opens a connection or selects a folder from a row click.
func (s *screen) rowActivated(uid string) {
	if isFolderUID(uid) {
		s.withDiscardGuard(func() {
			s.selectUID(uid)
			s.selectFolder(uid)
		})
		return
	}
	profile, known := s.index.profiles[uid]
	if !known {
		return
	}
	s.withDiscardGuard(func() {
		s.openConnection(profile)
		s.selectUID(connectionUID(profile.ID))
	})
}

// focusFolder selects a folder without triggering the discard guard twice.
func (s *screen) focusFolder(uid string) {
	s.suppressTree = true
	s.tree.Select(uid)
	s.suppressTree = false
	s.selected = ""
	s.selectedUID = uid
}

// focusConnection selects a connection without opening its tab.
func (s *screen) focusConnection(profile profiles.Profile) {
	s.suppressTree = true
	s.tree.Select(connectionUID(profile.ID))
	s.suppressTree = false
	s.selected = profile.ID
	s.selectedUID = connectionUID(profile.ID)
}
