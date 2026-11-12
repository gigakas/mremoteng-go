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
