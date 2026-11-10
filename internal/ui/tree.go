package ui

import (
<<<<<<< HEAD
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
=======
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
)

// ConnectionTree adapts a connection.ContainerInfo (the tree root) into a
// fyne widget.Tree, keyed on each node's stable ID. Fyne's tree widget is
// callback-based (it asks for children/branch-ness/labels by ID on
// demand, not handed a data structure directly), so this type's job is
// entirely that adaptation — it holds no connection state of its own
// beyond the ID→Node index needed to answer those callbacks.
type ConnectionTree struct {
	Widget *widget.Tree

	// OnSelect, if set, is called with the selected node whenever the
	// user picks a tree item. Left nil by default; stage 3.3 (open a
	// session tab) and stage 3.4 (show the properties panel) are the
	// intended consumers, wired from cmd/mremoteng or a future
	// application-assembly point, not from this package.
	OnSelect func(connection.Node)

	root  *connection.ContainerInfo
	nodes map[widget.TreeNodeID]connection.Node
}

// NewConnectionTree builds a tree widget over root. root's own children
// are the visible top-level items — the root container itself (in this
// model, always a synthetic "Connections" folder, see
// connection.NewRootInfo) is not shown as a row, following the same
// invisible-root convention Fyne's tree widget itself uses for its
// Root="" default.
func NewConnectionTree(root *connection.ContainerInfo) *ConnectionTree {
	t := &ConnectionTree{root: root}
	t.Widget = widget.NewTree(t.childUIDs, t.isBranch, t.createNode, t.updateNode)
	t.Widget.OnSelected = t.onSelected
	// RefreshItem on open/close so updateNode's folder-icon choice
	// (open vs closed) reflects the new state immediately rather than
	// waiting for some unrelated refresh to happen to touch this row.
	t.Widget.OnBranchOpened = t.Widget.RefreshItem
	t.Widget.OnBranchClosed = t.Widget.RefreshItem
	t.Reload()
	return t
}

// Reload rebuilds the ID→Node index from root and refreshes the widget.
// ConnectionTree has no way to observe tree mutations on its own —
// connection.ContainerInfo has no change-notification mechanism to
// subscribe to — so call Reload after any add/remove/move that makes a
// *container* reachable for the first time (a brand-new folder, or one
// moved in from elsewhere): until reindexed, it's simply absent from the
// index, so IsBranch misreports it as a leaf and its children are
// unreachable. Adding a child under a container that's already indexed
// doesn't need Reload — childUIDs calls Children() live on the indexed
// *connection.ContainerInfo pointer, so it already reflects the current
// state.
func (t *ConnectionTree) Reload() {
	t.nodes = make(map[widget.TreeNodeID]connection.Node)
	for _, child := range t.root.Children() {
		t.indexSubtree(child)
	}
	t.Widget.Refresh()
}

func (t *ConnectionTree) indexSubtree(n connection.Node) {
	t.nodes[n.Base().ID()] = n
	if c, ok := n.(*connection.ContainerInfo); ok {
		for _, child := range c.Children() {
			t.indexSubtree(child)
		}
	}
}

// Root returns the tree's current root container, e.g. for a caller that
// needs to serialize the whole tree (SaveConnectionsFile).
func (t *ConnectionTree) Root() *connection.ContainerInfo {
	return t.root
}

// SetRoot replaces the tree's root container and reindexes, e.g. after
// loading a different connections file. Unlike an in-place mutation under
// the existing root, this always needs a Reload: the old index is for a
// different root entirely.
func (t *ConnectionTree) SetRoot(root *connection.ContainerInfo) {
	t.root = root
	t.Reload()
}

func (t *ConnectionTree) containerFor(uid widget.TreeNodeID) *connection.ContainerInfo {
	if uid == "" {
		return t.root
	}
	c, _ := t.nodes[uid].(*connection.ContainerInfo)
	return c
}

func (t *ConnectionTree) childUIDs(uid widget.TreeNodeID) []widget.TreeNodeID {
	container := t.containerFor(uid)
	if container == nil {
		return nil
	}
	children := container.Children()
	ids := make([]widget.TreeNodeID, len(children))
	for i, child := range children {
		ids[i] = child.Base().ID()
	}
	return ids
}

func (t *ConnectionTree) isBranch(uid widget.TreeNodeID) bool {
	_, ok := t.nodes[uid].(*connection.ContainerInfo)
	return ok
}

func (t *ConnectionTree) onSelected(uid widget.TreeNodeID) {
	if t.OnSelect == nil {
		return
	}
	if node, ok := t.nodes[uid]; ok {
		t.OnSelect(node)
	}
}

// createNode builds the template row: an icon plus a label, updated
// in-place by updateNode. Fyne reuses these templates across rows as the
// tree scrolls (per NewTree's doc comment: "a new template object that
// can be cached"), so nothing here may capture per-node state.
func (t *ConnectionTree) createNode(branch bool) fyne.CanvasObject {
	icon := widget.NewIcon(theme.ComputerIcon())
	label := widget.NewLabel("")
	return container.NewHBox(icon, label)
}

func (t *ConnectionTree) updateNode(uid widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
	row := o.(*fyne.Container)
	icon := row.Objects[0].(*widget.Icon)
	label := row.Objects[1].(*widget.Label)

	node, ok := t.nodes[uid]
	if !ok {
		label.SetText(uid)
		return
	}
	label.SetText(node.Base().Effective().Name)

	if branch {
		if t.Widget.IsBranchOpen(uid) {
			icon.SetResource(theme.FolderOpenIcon())
		} else {
			icon.SetResource(theme.FolderIcon())
		}
	} else {
		icon.SetResource(theme.ComputerIcon())
	}
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
}
