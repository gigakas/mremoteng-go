package connection

import (
	"errors"
	"reflect"
	"testing"
)

func TestContainerInfo_AddChild_SetsParentAndPreservesOrder(t *testing.T) {
	root := testContainer(t, "root")
	first := testConnection(t, "first")
	second := testConnection(t, "second")
	if err := root.AddChild(first); err != nil {
		t.Fatal(err)
	}
	if err := root.AddChild(second); err != nil {
		t.Fatal(err)
	}
	if first.Parent() != root || second.Parent() != root {
		t.Fatal("child parent was not set")
	}
	assertNodeIDs(t, root.Children(), "first", "second")
}

func TestContainerInfo_AddChild_ExistingChild_IsIdempotent(t *testing.T) {
	root := testContainer(t, "root")
	child := testConnection(t, "child")
	if err := root.AddChild(child); err != nil {
		t.Fatal(err)
	}
	if err := root.InsertChild(0, child); err != nil {
		t.Fatal(err)
	}
	assertNodeIDs(t, root.Children(), "child")
}

func TestContainerInfo_AddChild_AttachedNode_MovesBetweenParents(t *testing.T) {
	oldParent := testContainer(t, "old")
	newParent := testContainer(t, "new")
	child := testConnection(t, "child")
	if err := oldParent.AddChild(child); err != nil {
		t.Fatal(err)
	}
	if err := newParent.AddChild(child); err != nil {
		t.Fatal(err)
	}
	if oldParent.HasChildren() {
		t.Error("old parent still contains moved child")
	}
	if child.Parent() != newParent {
		t.Error("new parent was not assigned")
	}
}

func TestContainerInfo_InsertAndSetPosition_ReordersChildren(t *testing.T) {
	root := testContainer(t, "root")
	a := testConnection(t, "a")
	b := testConnection(t, "b")
	c := testConnection(t, "c")
	_ = root.AddChild(a)
	_ = root.AddChild(c)
	if err := root.InsertChild(1, b); err != nil {
		t.Fatal(err)
	}
	assertNodeIDs(t, root.Children(), "a", "b", "c")
	if !root.SetChildPosition(a, 99) {
		t.Error("expected reorder")
	}
	assertNodeIDs(t, root.Children(), "b", "c", "a")
}

func TestContainerInfo_RemoveChild_ClearsParent(t *testing.T) {
	root := testContainer(t, "root")
	child := testConnection(t, "child")
	_ = root.AddChild(child)
	if !root.RemoveChild(child) {
		t.Fatal("expected child to be removed")
	}
	if child.Parent() != nil || root.HasChildren() {
		t.Error("remove did not clear both sides of the relationship")
	}
	if root.RemoveChild(child) {
		t.Error("removing an absent child must be a no-op")
	}
}

func TestContainerInfo_Descendants_ReturnsDepthFirstPreorder(t *testing.T) {
	root := testContainer(t, "root")
	left := testContainer(t, "left")
	leftLeaf := testConnection(t, "left-leaf")
	right := testConnection(t, "right")
	_ = root.AddChild(left)
	_ = left.AddChild(leftLeaf)
	_ = root.AddChild(right)
	assertNodeIDs(t, root.Descendants(), "left", "left-leaf", "right")
}

func TestWalk_NestedTree_VisitsRootAndDescendants(t *testing.T) {
	root := testContainer(t, "root")
	child := testContainer(t, "child")
	leaf := testConnection(t, "leaf")
	_ = root.AddChild(child)
	_ = child.AddChild(leaf)
	var got []string
	if err := Walk(root, func(node Node) error {
		got = append(got, node.Base().ID())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"root", "child", "leaf"}; !reflect.DeepEqual(got, want) {
		t.Errorf("walk = %v, want %v", got, want)
	}
}

func TestContainerInfo_AddChild_Ancestor_ReturnsCycleErrorAtomically(t *testing.T) {
	root := testContainer(t, "root")
	child := testContainer(t, "child")
	_ = root.AddChild(child)
	if err := child.AddChild(root); !errors.Is(err, ErrCycle) {
		t.Fatalf("error = %v, want ErrCycle", err)
	}
	if child.Parent() != root || root.Parent() != nil {
		t.Error("failed cycle operation changed the tree")
	}
	assertNodeIDs(t, root.Children(), "child")
}

func TestContainerInfo_InsertChild_InvalidIndex_LeavesOldParentIntact(t *testing.T) {
	oldParent := testContainer(t, "old")
	newParent := testContainer(t, "new")
	child := testConnection(t, "child")
	_ = oldParent.AddChild(child)
	if err := newParent.InsertChild(1, child); !errors.Is(err, ErrInvalidIndex) {
		t.Fatalf("error = %v, want ErrInvalidIndex", err)
	}
	if child.Parent() != oldParent {
		t.Error("failed move detached child from old parent")
	}
	assertNodeIDs(t, oldParent.Children(), "child")
}

func TestContainerInfo_Children_ReturnsCopy(t *testing.T) {
	root := testContainer(t, "root")
	child := testConnection(t, "child")
	_ = root.AddChild(child)
	children := root.Children()
	children[0] = testConnection(t, "replacement")
	assertNodeIDs(t, root.Children(), "child")
}

func TestContainerInfo_AddChild_ContainerBaseAlias_ReturnsError(t *testing.T) {
	root := testContainer(t, "root")
	child := testContainer(t, "child")
	if err := root.AddChild(child.Base()); !errors.Is(err, ErrNodeAlias) {
		t.Fatalf("error = %v, want ErrNodeAlias", err)
	}
	if root.HasChildren() || child.Parent() != nil {
		t.Error("rejected alias changed the tree")
	}
}

func TestContainerInfo_RelativeAdd_NilReceiver_ReturnsError(t *testing.T) {
	var container *ContainerInfo
	child := testConnection(t, "child")
	if err := container.AddChildAbove(child, nil); !errors.Is(err, ErrNilContainer) {
		t.Errorf("AddChildAbove error = %v, want ErrNilContainer", err)
	}
	if err := container.AddChildBelow(child, nil); !errors.Is(err, ErrNilContainer) {
		t.Errorf("AddChildBelow error = %v, want ErrNilContainer", err)
	}
}

func testConnection(t *testing.T, id string) *ConnectionInfo {
	t.Helper()
	connection, err := NewConnectionInfoWithID(id)
	if err != nil {
		t.Fatal(err)
	}
	return connection
}

func testContainer(t *testing.T, id string) *ContainerInfo {
	t.Helper()
	container, err := NewContainerInfoWithID(id)
	if err != nil {
		t.Fatal(err)
	}
	return container
}

func assertNodeIDs(t *testing.T, nodes []Node, want ...string) {
	t.Helper()
	got := make([]string, len(nodes))
	for i, node := range nodes {
		got[i] = node.Base().ID()
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("node IDs = %v, want %v", got, want)
	}
}
