package ui

import (
	"sort"
	"strings"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

// Tree node identifiers. Folders are implicit: they exist while at least one
// connection references them, which keeps the store free of empty placeholders.
const (
	folderPrefix     = "f:"
	connectionPrefix = "c:"
)

func folderUID(path string) string   { return folderPrefix + path }
func connectionUID(id string) string { return connectionPrefix + id }
func isFolderUID(uid string) bool    { return strings.HasPrefix(uid, folderPrefix) }

// treeIndex is the filtered, sorted view of the connection tree.
type treeIndex struct {
	profiles    map[string]profiles.Profile // connection uid -> profile
	children    map[string][]string         // parent uid -> child uids
	folders     map[string]string           // folder uid -> display name
	connections int
}

func buildTreeIndex(all []profiles.Profile, folders []string, query string) *treeIndex {
	index := &treeIndex{
		profiles: map[string]profiles.Profile{},
		children: map[string][]string{},
		folders:  map[string]string{},
	}
	if query == "" {
		for _, folder := range folders {
			index.ensureFolder(folder)
		}
	}
	matched := make([]profiles.Profile, 0, len(all))
	for _, profile := range all {
		if query == "" || matchesQuery(profile, query) {
			matched = append(matched, profile)
		}
	}
	sort.SliceStable(matched, func(i, j int) bool {
		return strings.ToLower(matched[i].Name) < strings.ToLower(matched[j].Name)
	})
	for _, profile := range matched {
		parentUID := ""
		path := ""
		if segments, err := profiles.FolderSegments(profile.Folder); err == nil {
			for _, segment := range segments {
				if path == "" {
					path = segment
				} else {
					path += "/" + segment
				}
				uid := folderUID(path)
				if _, known := index.folders[uid]; !known {
					index.folders[uid] = segment
					index.children[parentUID] = append(index.children[parentUID], uid)
				}
				parentUID = uid
			}
		}
		uid := connectionUID(profile.ID)
		index.profiles[uid] = profile
		index.children[parentUID] = append(index.children[parentUID], uid)
		index.connections++
	}
	index.sortChildren()
	return index
}

// ensureFolder adds a folder path and its ancestors, returning its uid.
func (index *treeIndex) ensureFolder(folder string) string {
	segments, err := profiles.FolderSegments(folder)
	if err != nil {
		return ""
	}
	parent := ""
	path := ""
	for _, segment := range segments {
		if path == "" {
			path = segment
		} else {
			path += "/" + segment
		}
		uid := folderUID(path)
		if _, known := index.folders[uid]; !known {
			index.folders[uid] = segment
			index.children[parent] = append(index.children[parent], uid)
		}
		parent = uid
	}
	return parent
}

// sortChildren orders folders before connections and each group by label.
func (index *treeIndex) sortChildren() {
	for parent, children := range index.children {
		sort.SliceStable(children, func(i, j int) bool {
			leftFolder, rightFolder := isFolderUID(children[i]), isFolderUID(children[j])
			if leftFolder != rightFolder {
				return leftFolder
			}
			return strings.ToLower(index.label(children[i])) < strings.ToLower(index.label(children[j]))
		})
		index.children[parent] = children
	}
}

func (index *treeIndex) childUIDs(parent string) []string {
	if index == nil {
		return nil
	}
	return index.children[parent]
}

func (index *treeIndex) label(uid string) string {
	if index == nil {
		return ""
	}
	if isFolderUID(uid) {
		return index.folders[uid]
	}
	return index.profiles[uid].Name
}

func (index *treeIndex) folderPath(uid string) string {
	return strings.TrimPrefix(uid, folderPrefix)
}

func matchesQuery(profile profiles.Profile, query string) bool {
	for _, value := range []string{profile.Name, profile.Folder, profile.Host, string(profile.Protocol), profile.Username} {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}
