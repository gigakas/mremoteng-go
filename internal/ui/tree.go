package ui

import (
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
}
