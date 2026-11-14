package ui

import "strings"

// moveInOrder moves a tree node one place up or down among its siblings of the same
// kind. Folders and connections are two groups that never interleave, so a connection
// never jumps over a folder.
func (s *screen) moveInOrder(uid string, delta int) {
	if s.index == nil || delta == 0 {
		return
	}
	if !s.requireWritable("Reordering connections") {
		return
	}
	siblings := s.siblingGroup(uid)
	position := -1
	for index, sibling := range siblings {
		if sibling == uid {
			position = index
		}
	}
	if position < 0 {
		return
	}
	target := position + delta
	if target < 0 || target >= len(siblings) {
		s.setStatus(s.orderEdgeMessage(uid, delta))
		return
	}

	label := s.index.label(uid)
	keys := s.orderedKeysForWrite()
	from, to := indexOfKey(keys, uid), indexOfKey(keys, siblings[target])
	if from < 0 || to < 0 {
		return
	}
	// Swapping the two entries is enough: only the relative order of siblings matters,
	// so the subtrees in between stay where they are.
	keys[from], keys[to] = keys[to], keys[from]
	if err := s.ws.setOrder(keys); err != nil {
		s.setProblem("Could not reorder: " + err.Error())
		return
	}
	s.refreshTree()
	s.selectUID(uid)
	s.setStatus("Moved " + label + s.orderDirection(delta) + ".")
}

// siblingGroup lists the siblings of uid that are of its own kind, in the order the
// tree currently shows them.
func (s *screen) siblingGroup(uid string) []string {
	parent := s.parentOf(uid)
	group := make([]string, 0, 4)
	for _, child := range s.index.childUIDs(parent) {
		if isFolderUID(child) == isFolderUID(uid) {
			group = append(group, child)
		}
	}
	return group
}

// parentOf finds the node that holds uid, walking the whole index because the tree
// keeps parent-to-children only.
func (s *screen) parentOf(uid string) string {
	for parent, children := range s.index.children {
		for _, child := range children {
			if child == uid {
				return parent
			}
		}
	}
	return ""
}

// orderedKeysForWrite lists every node in display order, which is what a manual move
// writes back: a full list keeps every other arrangement in place.
//
// The walk uses the unfiltered tree when a filter hides rows. Reordering what is on
// screen must not forget where the hidden nodes were, or clearing the filter would
// shuffle them back to alphabetical order.
func (s *screen) orderedKeysForWrite() []string {
	index := s.index
	if strings.TrimSpace(s.filter.Text) != "" {
		index = buildTreeIndex(s.ws.profiles, s.ws.folders, "", orderRanks(s.ws.order))
	}
	keys := make([]string, 0, len(index.profiles)+len(index.folders))
	var walk func(parent string)
	walk = func(parent string) {
		for _, child := range index.childUIDs(parent) {
			keys = append(keys, child)
			walk(child)
		}
	}
	walk("")
	return keys
}

// orderDirection names the move for the status line.
func (s *screen) orderDirection(delta int) string {
	if delta < 0 {
		return " up"
	}
	return " down"
}

// orderEdgeMessage explains why a move did nothing, naming the group so the user is
// not left guessing why the row would not go further.
func (s *screen) orderEdgeMessage(uid string, delta int) string {
	group := "connection"
	if isFolderUID(uid) {
		group = "folder"
	}
	if delta < 0 {
		return "This is already the first " + group + " of its group."
	}
	return "This is already the last " + group + " of its group."
}

// indexOfKey reports the position of key in keys, or -1.
func indexOfKey(keys []string, key string) int {
	for index, candidate := range keys {
		if candidate == key {
			return index
		}
	}
	return -1
}

// orderUp and orderDown are the context menu entries; uid is the selected row.
func (s *screen) orderUp(uid string)   { s.moveInOrder(uid, -1) }
func (s *screen) orderDown(uid string) { s.moveInOrder(uid, 1) }

// hasRoom reports whether a node can move in that direction, so a menu entry can be
// disabled instead of failing when used.
func (s *screen) hasRoom(uid string, delta int) bool {
	group := s.siblingGroup(uid)
	for index, sibling := range group {
		if sibling != uid {
			continue
		}
		target := index + delta
		return target >= 0 && target < len(group)
	}
	return false
}
