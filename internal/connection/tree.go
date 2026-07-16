package connection

import (
	"errors"
	"fmt"
)

var (
	ErrNilNode      = errors.New("connection: node is nil")
	ErrNilContainer = errors.New("connection: container is nil")
	ErrInvalidIndex = errors.New("connection: child index out of range")
	ErrCycle        = errors.New("connection: operation would create a tree cycle")
	ErrNodeAlias    = errors.New("connection: container base cannot be used as a leaf node")
	ErrRootChild    = errors.New("connection: root cannot be added as a child")
)

// Node is a connection or container in the homogeneous connection tree.
// The unexported marker prevents other packages from bypassing tree
// invariants with custom node implementations.
type Node interface {
	Base() *ConnectionInfo
	Kind() NodeKind
	isNode()
}

func (c *ConnectionInfo) Base() *ConnectionInfo { return c }
func (c *ConnectionInfo) Kind() NodeKind        { return NodeKindConnection }
func (c *ConnectionInfo) isNode()               {}

// ContainerInfo is a connection record that owns an ordered list of child
// connection/container nodes. Carrying the complete ConnectionInfo superset
// lets containers act as inheritance templates in stage 1.2.
type ContainerInfo struct {
	info     *ConnectionInfo
	root     bool
	expanded bool
	children []Node
}

// NewContainerInfo creates a detached container with a random ID.
func NewContainerInfo() (*ContainerInfo, error) {
	base, err := NewConnectionInfo()
	if err != nil {
		return nil, err
	}
	return newContainer(base), nil
}

// NewContainerInfoWithID creates a detached container preserving an existing
// serialized ID.
func NewContainerInfoWithID(id string) (*ContainerInfo, error) {
	base, err := NewConnectionInfoWithID(id)
	if err != nil {
		return nil, err
	}
	return newContainer(base), nil
}

// NewRootInfo creates the connection-tree root. Inheritance is disabled for
// the root and its direct children, matching mRemoteNG's RootNodeInfo rules.
func NewRootInfo() (*ContainerInfo, error) {
	base, err := NewConnectionInfo()
	if err != nil {
		return nil, err
	}
	return newRoot(base), nil
}

// NewRootInfoWithID creates a root preserving an existing ID.
func NewRootInfoWithID(id string) (*ContainerInfo, error) {
	base, err := NewConnectionInfoWithID(id)
	if err != nil {
		return nil, err
	}
	return newRoot(base), nil
}

func newContainer(base *ConnectionInfo) *ContainerInfo {
	base.Raw.Name = "New Folder"
	container := &ContainerInfo{info: base, expanded: true}
	base.containerOwner = container
	return container
}

func newRoot(base *ConnectionInfo) *ContainerInfo {
	root := newContainer(base)
	root.root = true
	root.info.Raw.Name = "Connections"
	return root
}

func (c *ContainerInfo) Base() *ConnectionInfo {
	if c == nil {
		return nil
	}
	return c.info
}

func (c *ContainerInfo) Kind() NodeKind {
	if c != nil && c.root {
		return NodeKindRoot
	}
	return NodeKindContainer
}
func (c *ContainerInfo) isNode() {}

// IsRoot reports whether c is the tree root.
func (c *ContainerInfo) IsRoot() bool { return c != nil && c.root }

// ID returns the stable ID of the container's connection record.
func (c *ContainerInfo) ID() string { return c.Base().ID() }

// Parent returns the current parent, or nil when detached.
func (c *ContainerInfo) Parent() *ContainerInfo {
	if c == nil {
		return nil
	}
	return c.info.Parent()
}

// Expanded reports whether the container is expanded in the tree UI.
func (c *ContainerInfo) Expanded() bool { return c != nil && c.expanded }

// SetExpanded changes the persisted expanded state.
func (c *ContainerInfo) SetExpanded(expanded bool) {
	if c != nil {
		c.expanded = expanded
	}
}

// Children returns a copy so callers cannot mutate tree relationships without
// going through the invariant-preserving operations below.
func (c *ContainerInfo) Children() []Node {
	if c == nil {
		return nil
	}
	return append([]Node(nil), c.children...)
}

func (c *ContainerInfo) HasChildren() bool { return c != nil && len(c.children) > 0 }

// AddChild appends child. If child belongs to another container, it is moved.
// Adding a child already present in c is an idempotent no-op, matching the C#
// model.
func (c *ContainerInfo) AddChild(child Node) error {
	if c == nil {
		return ErrNilContainer
	}
	return c.InsertChild(len(c.children), child)
}

// InsertChild inserts child at index, moving it from its former parent. All
// validation happens before detachment, so failed moves leave the tree intact.
func (c *ContainerInfo) InsertChild(index int, child Node) error {
	if c == nil {
		return ErrNilContainer
	}
	if index < 0 || index > len(c.children) {
		return fmt.Errorf("%w: %d", ErrInvalidIndex, index)
	}
	base, err := validatedNodeBase(child)
	if err != nil {
		return err
	}
	if base.parent == c {
		return nil
	}
	if c.wouldCreateCycle(base) {
		return ErrCycle
	}
	if base.parent != nil {
		base.parent.removeChild(base)
	}
	base.parent = c
	c.children = append(c.children, nil)
	copy(c.children[index+1:], c.children[index:])
	c.children[index] = child
	return nil
}

// AddChildAbove inserts child immediately before reference, or appends it if
// reference is not a direct child.
func (c *ContainerInfo) AddChildAbove(child, reference Node) error {
	if c == nil {
		return ErrNilContainer
	}
	index := c.indexOf(nodeBase(reference))
	if index < 0 {
		index = len(c.children)
	}
	return c.InsertChild(index, child)
}

// AddChildBelow inserts child immediately after reference, or appends it if
// reference is not a direct child.
func (c *ContainerInfo) AddChildBelow(child, reference Node) error {
	if c == nil {
		return ErrNilContainer
	}
	index := c.indexOf(nodeBase(reference))
	if index < 0 {
		index = len(c.children)
	} else {
		index++
	}
	return c.InsertChild(index, child)
}

// RemoveChild detaches child when it is a direct child. It reports whether a
// node was removed; absent and nil nodes are no-ops.
func (c *ContainerInfo) RemoveChild(child Node) bool {
	if c == nil {
		return false
	}
	return c.removeChild(nodeBase(child))
}

// SetChildPosition reorders a direct child. Negative indices and absent
// children are no-ops; indices beyond the end are clamped, matching mRemoteNG.
func (c *ContainerInfo) SetChildPosition(child Node, newIndex int) bool {
	if c == nil || newIndex < 0 {
		return false
	}
	base := nodeBase(child)
	original := c.indexOf(base)
	if original < 0 || original == newIndex {
		return false
	}
	node := c.children[original]
	c.children = append(c.children[:original], c.children[original+1:]...)
	if newIndex > len(c.children) {
		newIndex = len(c.children)
	}
	c.children = append(c.children, nil)
	copy(c.children[newIndex+1:], c.children[newIndex:])
	c.children[newIndex] = node
	return true
}

// Descendants returns depth-first preorder descendants, excluding c itself.
func (c *ContainerInfo) Descendants() []Node {
	if c == nil {
		return nil
	}
	var out []Node
	for _, child := range c.children {
		out = append(out, child)
		if container, ok := child.(*ContainerInfo); ok {
			out = append(out, container.Descendants()...)
		}
	}
	return out
}

// Walk visits root and then every descendant in depth-first preorder.
func Walk(root Node, visit func(Node) error) error {
	if nodeBase(root) == nil {
		return ErrNilNode
	}
	if visit == nil {
		return errors.New("connection: visit function is nil")
	}
	if err := visit(root); err != nil {
		return err
	}
	container, ok := root.(*ContainerInfo)
	if !ok {
		return nil
	}
	for _, child := range container.children {
		if err := Walk(child, visit); err != nil {
			return err
		}
	}
	return nil
}

func (c *ContainerInfo) wouldCreateCycle(child *ConnectionInfo) bool {
	for current := c; current != nil; current = current.Parent() {
		if current.Base() == child {
			return true
		}
	}
	return false
}

func (c *ContainerInfo) removeChild(child *ConnectionInfo) bool {
	index := c.indexOf(child)
	if index < 0 {
		return false
	}
	c.children = append(c.children[:index], c.children[index+1:]...)
	child.parent = nil
	return true
}

func (c *ContainerInfo) indexOf(child *ConnectionInfo) int {
	if c == nil || child == nil {
		return -1
	}
	for i, candidate := range c.children {
		if nodeBase(candidate) == child {
			return i
		}
	}
	return -1
}

func nodeBase(node Node) *ConnectionInfo {
	if node == nil {
		return nil
	}
	return node.Base()
}

func validatedNodeBase(node Node) (*ConnectionInfo, error) {
	base := nodeBase(node)
	if base == nil {
		return nil, ErrNilNode
	}
	if base.containerOwner != nil {
		container, ok := node.(*ContainerInfo)
		if !ok || container != base.containerOwner {
			return nil, ErrNodeAlias
		}
		if container.root {
			return nil, ErrRootChild
		}
	}
	return base, nil
}
