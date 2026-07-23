package ui_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/ui"
)

// testTree builds root -> [ "Servers" -> ["web1"], "standalone" ] and
// returns the root plus the leaf IDs, for tests that need to assert on
// specific nodes.
func testTree(t *testing.T) (root *connection.ContainerInfo, serversID, web1ID, standaloneID string) {
	t.Helper()

	root, err := connection.NewRootInfo()
	if err != nil {
		t.Fatalf("NewRootInfo: %v", err)
	}

	servers, err := connection.NewContainerInfo()
	if err != nil {
		t.Fatalf("NewContainerInfo: %v", err)
	}
	servers.Base().Raw.Name = "Servers"
	if err := root.AddChild(servers); err != nil {
		t.Fatalf("AddChild(servers): %v", err)
	}

	web1, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	web1.Raw.Name = "web1"
	if err := servers.AddChild(web1); err != nil {
		t.Fatalf("AddChild(web1): %v", err)
	}

	standalone, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	standalone.Raw.Name = "standalone"
	if err := root.AddChild(standalone); err != nil {
		t.Fatalf("AddChild(standalone): %v", err)
	}

	return root, servers.ID(), web1.ID(), standalone.ID()
}

func labelText(t *testing.T, o fyne.CanvasObject) string {
	t.Helper()
	row, ok := o.(*fyne.Container)
	if !ok || len(row.Objects) != 2 {
		t.Fatalf("node template = %#v, want a 2-object *fyne.Container (icon, label)", o)
	}
	label, ok := row.Objects[1].(*widget.Label)
	if !ok {
		t.Fatalf("row.Objects[1] = %T, want *widget.Label", row.Objects[1])
	}
	return label.Text
}

func TestNewConnectionTree_ChildUIDsReflectsModel(t *testing.T) {
	root, serversID, web1ID, standaloneID := testTree(t)
	tree := ui.NewConnectionTree(root)

	top := tree.Widget.ChildUIDs("")
	if len(top) != 2 || top[0] != serversID || top[1] != standaloneID {
		t.Fatalf("top-level ChildUIDs = %v, want [%s %s]", top, serversID, standaloneID)
	}

	children := tree.Widget.ChildUIDs(serversID)
	if len(children) != 1 || children[0] != web1ID {
		t.Fatalf("ChildUIDs(servers) = %v, want [%s]", children, web1ID)
	}

	if leaf := tree.Widget.ChildUIDs(web1ID); len(leaf) != 0 {
		t.Errorf("ChildUIDs(web1) = %v, want empty (it's a leaf)", leaf)
	}
}

func TestNewConnectionTree_IsBranchDistinguishesContainersFromConnections(t *testing.T) {
	root, serversID, web1ID, _ := testTree(t)
	tree := ui.NewConnectionTree(root)

	if !tree.Widget.IsBranch(serversID) {
		t.Error("IsBranch(servers) = false, want true")
	}
	if tree.Widget.IsBranch(web1ID) {
		t.Error("IsBranch(web1) = true, want false")
	}
}

func TestNewConnectionTree_UpdateNodeSetsLabelToEffectiveName(t *testing.T) {
	root, _, web1ID, _ := testTree(t)
	tree := ui.NewConnectionTree(root)

	node := tree.Widget.CreateNode(false)
	tree.Widget.UpdateNode(web1ID, false, node)

	if got := labelText(t, node); got != "web1" {
		t.Errorf("label = %q, want %q", got, "web1")
	}
}

func TestNewConnectionTree_OnSelectFiresWithTheSelectedNode(t *testing.T) {
	root, _, web1ID, _ := testTree(t)
	tree := ui.NewConnectionTree(root)

	var got connection.Node
	tree.OnSelect = func(n connection.Node) { got = n }

	tree.Widget.OnSelected(web1ID)

	if got == nil {
		t.Fatal("OnSelect was not called")
	}
	if got.Base().ID() != web1ID {
		t.Errorf("selected node ID = %q, want %q", got.Base().ID(), web1ID)
	}
}

// TestConnectionTree_Reload_IndexesNewlyReachableContainers exercises the
// case that actually needs Reload: childUIDs calls Children() live on
// whatever *connection.ContainerInfo pointer is already in the index, so
// adding a child under an *already-indexed* container is visible
// immediately with no Reload needed (not what this test checks). What
// does need Reload is a container that didn't exist in the tree at index
// time at all — until reindexed, it's simply absent from t.nodes, so
// IsBranch misreports it as a leaf and its own children are unreachable.
func TestConnectionTree_Reload_IndexesNewlyReachableContainers(t *testing.T) {
	root, _, _, _ := testTree(t)
	tree := ui.NewConnectionTree(root)

	newFolder, err := connection.NewContainerInfo()
	if err != nil {
		t.Fatalf("NewContainerInfo: %v", err)
	}
	newFolder.Base().Raw.Name = "New Folder"
	if err := root.AddChild(newFolder); err != nil {
		t.Fatalf("AddChild(newFolder): %v", err)
	}
	newLeaf, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	newLeaf.Raw.Name = "web2"
	if err := newFolder.AddChild(newLeaf); err != nil {
		t.Fatalf("AddChild(web2): %v", err)
	}

	// Before Reload: newFolder isn't in the index yet, so it's
	// misclassified as a leaf and its child is unreachable.
	if tree.Widget.IsBranch(newFolder.ID()) {
		t.Error("IsBranch(newFolder) before Reload = true, want false (not indexed yet)")
	}
	if children := tree.Widget.ChildUIDs(newFolder.ID()); len(children) != 0 {
		t.Errorf("ChildUIDs(newFolder) before Reload = %v, want empty (not indexed yet)", children)
	}

	tree.Reload()

	if !tree.Widget.IsBranch(newFolder.ID()) {
		t.Error("IsBranch(newFolder) after Reload = false, want true")
	}
	children := tree.Widget.ChildUIDs(newFolder.ID())
	if len(children) != 1 || children[0] != newLeaf.ID() {
		t.Errorf("ChildUIDs(newFolder) after Reload = %v, want [%s]", children, newLeaf.ID())
	}
}

func TestConnectionTree_SetRoot_SwapsRootAndReindexes(t *testing.T) {
	root, serversID, _, _ := testTree(t)
	tree := ui.NewConnectionTree(root)

	newRoot, err := connection.NewRootInfo()
	if err != nil {
		t.Fatalf("NewRootInfo: %v", err)
	}
	other, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	other.Raw.Name = "other"
	if err := newRoot.AddChild(other); err != nil {
		t.Fatalf("AddChild(other): %v", err)
	}

	tree.SetRoot(newRoot)

	if got := tree.Root(); got != newRoot {
		t.Fatalf("Root() = %p, want %p", got, newRoot)
	}
	top := tree.Widget.ChildUIDs("")
	if len(top) != 1 || top[0] != other.ID() {
		t.Fatalf("ChildUIDs(\"\") after SetRoot = %v, want [%s]", top, other.ID())
	}
	// The old root's nodes must no longer be reachable through the tree.
	if children := tree.Widget.ChildUIDs(serversID); len(children) != 0 {
		t.Errorf("ChildUIDs(old servers) after SetRoot = %v, want empty", children)
	}
	if tree.Widget.IsBranch(serversID) {
		t.Error("IsBranch(old servers) after SetRoot = true, want false (no longer indexed)")
	}
}
