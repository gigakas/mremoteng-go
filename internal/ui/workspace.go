package ui

import (
	"sort"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

// workspace owns the persisted document and nothing else: the connection
// profiles, the explicit folder list, the folder defaults and the folders the
// user collapsed. It deliberately has no Fyne dependency, so the persistence
// rules can be tested without a window, a canvas or a driver.
//
// It is not a cache of the window state: the screen keeps the selection, the
// tree index, the open tabs and the status line. The workspace only answers
// "what is saved" and "may we write".
type workspace struct {
	store     Store
	profiles  []profiles.Profile
	folders   []string
	defaults  []profiles.FolderDefault
	collapsed map[string]bool
	writable  bool
}

func newWorkspace(store Store) *workspace {
	return &workspace{store: store, collapsed: map[string]bool{}}
}

// load reads the store once. A load failure clears the state and leaves the
// workspace read-only, so broken data is never overwritten; the caller only has
// to report the error and disable the editor.
func (w *workspace) load() error {
	loaded, err := w.store.Load()
	if err != nil {
		w.writable = false
		w.profiles = nil
		w.folders = nil
		w.defaults = nil
		w.collapsed = map[string]bool{}
		return err
	}
	w.profiles = loaded.Profiles
	w.folders = loaded.Folders
	w.defaults = loaded.Defaults
	w.collapsed = map[string]bool{}
	for _, folder := range loaded.Collapsed {
		w.collapsed[folder] = true
	}
	w.writable = true
	return nil
}

// document builds the store document from the candidate slices plus the current
// defaults and collapsed state. Callers pass the state they are about to commit,
// which is what keeps "save before assigning" possible.
func (w *workspace) document(profs []profiles.Profile, folders []string) profiles.Document {
	return profiles.Document{
		Profiles:  profs,
		Folders:   folders,
		Defaults:  w.defaults,
		Collapsed: collapsedList(w.collapsed),
	}
}

// documentWithDefaults is document with an explicit defaults slice.
func (w *workspace) documentWithDefaults(profs []profiles.Profile, folders []string, defaults []profiles.FolderDefault) profiles.Document {
	document := w.document(profs, folders)
	document.Defaults = defaults
	return document
}

// save persists the current state. A read-only workspace or a missing store is a
// no-op, not an error: the window stays usable after a failed load.
func (w *workspace) save() error {
	if w.store == nil || !w.writable {
		return nil
	}
	return w.store.Save(w.document(w.profiles, w.folders))
}

// upsert returns profiles with profile added or replaced and leaves the receiver
// untouched, so a failed save keeps the previous slice in place.
func (w *workspace) upsert(profile profiles.Profile) []profiles.Profile {
	next := make([]profiles.Profile, 0, len(w.profiles)+1)
	replaced := false
	for _, existing := range w.profiles {
		if existing.ID == profile.ID {
			next = append(next, profile)
			replaced = true
			continue
		}
		next = append(next, existing)
	}
	if !replaced {
		next = append(next, profile)
	}
	return next
}

// commit writes the candidate state and only then replaces the in-memory one, so
// a failed write leaves the workspace exactly as it was.
func (w *workspace) commit(profs []profiles.Profile, folders []string) error {
	if err := w.store.Save(w.document(profs, folders)); err != nil {
		return err
	}
	w.profiles = profs
	w.folders = folders
	return nil
}

// putProfile stores profile and, when needed, its folder entry. The write goes
// first: if the store rejects it, nothing changes in memory.
func (w *workspace) putProfile(profile profiles.Profile) error {
	return w.commit(w.upsert(profile), addFolderPath(w.folders, profile.Folder))
}

// deleteProfile removes the profile with the given id.
func (w *workspace) deleteProfile(id string) error {
	next := make([]profiles.Profile, 0, len(w.profiles))
	for _, existing := range w.profiles {
		if existing.ID != id {
			next = append(next, existing)
		}
	}
	return w.commit(next, w.folders)
}

// commitWith writes the candidate state, including an explicit defaults slice,
// and only then replaces the in-memory one.
func (w *workspace) commitWith(profs []profiles.Profile, folders []string, defaults []profiles.FolderDefault) error {
	if err := w.store.Save(w.documentWithDefaults(profs, folders, defaults)); err != nil {
		return err
	}
	w.profiles = profs
	w.folders = folders
	w.defaults = defaults
	return nil
}

// addFolder records an explicit folder path. The caller validates the path and
// checks for an existing entry.
func (w *workspace) addFolder(path string) error {
	return w.commit(w.profiles, addFolderPath(w.folders, path))
}

// setFolderDefaults stores the shared values for one folder. A zero port means
// the field was left empty, which removes the entry when there is no username
// either.
func (w *workspace) setFolderDefaults(path, username string, port int) error {
	defaults := make([]profiles.FolderDefault, 0, len(w.defaults)+1)
	for _, item := range w.defaults {
		if item.Path != path {
			defaults = append(defaults, item)
		}
	}
	entry := profiles.FolderDefault{Path: path, Username: username, Port: port}
	if entry.Username != "" || entry.Port != 0 {
		defaults = append(defaults, entry)
	}
	return w.commitWith(w.profiles, w.folders, defaults)
}

// renameFolder moves old, and everything below it, to the new path. The caller
// validates the target and rejects collisions.
func (w *workspace) renameFolder(old, to string) error {
	nextFolders := make([]string, 0, len(w.folders))
	seen := map[string]bool{}
	for _, existing := range w.folders {
		renamed := existing
		if withinFolder(existing, old) {
			renamed = replaceFolderPrefix(existing, old, to)
		}
		if !seen[renamed] {
			seen[renamed] = true
			nextFolders = append(nextFolders, renamed)
		}
	}
	nextProfiles := make([]profiles.Profile, len(w.profiles))
	copy(nextProfiles, w.profiles)
	for i, profile := range nextProfiles {
		if withinFolder(profile.Folder, old) {
			nextProfiles[i].Folder = replaceFolderPrefix(profile.Folder, old, to)
		}
	}
	return w.commit(nextProfiles, nextFolders)
}

// deleteFolder removes a folder and every profile inside it.
func (w *workspace) deleteFolder(folder string) error {
	nextProfiles := make([]profiles.Profile, 0, len(w.profiles))
	for _, profile := range w.profiles {
		if withinFolder(profile.Folder, folder) {
			continue
		}
		nextProfiles = append(nextProfiles, profile)
	}
	nextFolders := make([]string, 0, len(w.folders))
	for _, existing := range w.folders {
		if withinFolder(existing, folder) {
			continue
		}
		nextFolders = append(nextFolders, existing)
	}
	return w.commit(nextProfiles, nextFolders)
}

// renameProfile stores an edited profile without touching the folder list: a
// rename never moves the connection, so it must not add folder entries.
func (w *workspace) renameProfile(profile profiles.Profile) error {
	return w.commit(w.upsert(profile), w.folders)
}

// putConnection stores an edited profile and, when it now lives in a folder,
// records that folder too.
func (w *workspace) putConnection(profile profiles.Profile) error {
	folders := w.folders
	if profile.Folder != "" {
		folders = addFolderPath(w.folders, profile.Folder)
	}
	return w.commit(w.upsert(profile), folders)
}

// collapsedList turns the set into a sorted slice, so the document is stable
// across runs.
func collapsedList(set map[string]bool) []string {
	result := make([]string, 0, len(set))
	for folder := range set {
		result = append(result, folder)
	}
	sort.Strings(result)
	return result
}
