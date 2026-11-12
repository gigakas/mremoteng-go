package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

// paneNode is one cell of a tab's tmux-style layout: a leaf shows one session
// panel, a split shows two child nodes side by side or stacked. view is the
// stack that displays the node, so a leaf can turn into a split (and back) in
// place without its parent noticing.
type paneNode struct {
	parent *paneNode
	panel  *sessionPanel
	a, b   *paneNode
	view   *fyne.Container
}

// newRootPane wraps the panel of a freshly opened tab.
func newRootPane(panel *sessionPanel) *paneNode {
	node := &paneNode{panel: panel, view: container.NewStack(panel.content())}
	panel.pane = node
	return node
}

// leaves lists the session panels under this node, in layout order.
func (n *paneNode) leaves() []*sessionPanel {
	if n.panel != nil {
		return []*sessionPanel{n.panel}
	}
	return append(n.a.leaves(), n.b.leaves()...)
}

func (n *paneNode) root() *paneNode {
	for n.parent != nil {
		n = n.parent
	}
	return n
}

// splittable reports whether a protocol can be duplicated into another pane.
// RDP and VNC embed a native view that cannot be shared or tiled safely.
func splittable(protocol profiles.Protocol) bool {
	switch protocol {
	case profiles.SSH, profiles.TELNET, profiles.RAW, profiles.RLOGIN:
		return true
	}
	return false
}

// splitPane duplicates the connection of p into a new pane next to it
// (sideBySide) or below it, and connects the copy.
func (s *screen) splitPane(p *sessionPanel, sideBySide bool) {
	if p == nil || p.pane == nil {
		s.setStatus("Open a terminal session first, then split its pane.")
		return
	}
	if !splittable(p.profile.Protocol) {
		s.setStatus("Only SSH, Telnet, RAW and rlogin panes can be split.")
		return
	}
	leaf := p.pane
	dup := s.newSessionPanel(p.profile)
	dup.password.SetText(p.password.Text)

	mine := &paneNode{parent: leaf, panel: p, view: container.NewStack(leaf.view.Objects...)}
	other := &paneNode{parent: leaf, panel: dup, view: container.NewStack(dup.content())}
	p.pane, dup.pane = mine, other
	leaf.panel, leaf.a, leaf.b = nil, mine, other

	if sideBySide {
		leaf.view.Objects = []fyne.CanvasObject{container.NewHSplit(mine.view, other.view)}
	} else {
		leaf.view.Objects = []fyne.CanvasObject{container.NewVSplit(mine.view, other.view)}
	}
	leaf.view.Refresh()
	dup.connectShell()
}

// closePane ends the session of p and gives its space to the sibling pane. The
// last pane of a tab stays: closing the tab ends the session.
func (s *screen) closePane(p *sessionPanel) {
	if p == nil || p.pane == nil || p.pane.parent == nil {
		s.setStatus("This is the only pane of the tab; close the tab to end the session.")
		return
	}
	node := p.pane
	parent := node.parent
	sibling := parent.a
	if sibling == node {
		sibling = parent.b
	}
	p.disconnectShell()

	parent.panel, parent.a, parent.b = sibling.panel, sibling.a, sibling.b
	if parent.panel != nil {
		parent.panel.pane = parent
	} else {
		parent.a.parent, parent.b.parent = parent, parent
	}
	parent.view.Objects = sibling.view.Objects
	parent.view.Refresh()

	// The tab keeps pointing at a live panel if the one it knew was closed.
	for id, known := range s.panels {
		if known == p {
			s.panels[id] = parent.leaves()[0]
		}
	}
	if survivor := parent.leaves()[0]; survivor.term != nil {
		survivor.term.focus()
	}
}

// tabPanels lists every session panel of the tab opened for a connection.
func (s *screen) tabPanels(id string) []*sessionPanel {
	main, ok := s.panels[id]
	if !ok {
		return nil
	}
	if main.pane == nil {
		return []*sessionPanel{main}
	}
	return main.pane.root().leaves()
}

// allPanels lists every session panel in every tab.
func (s *screen) allPanels() []*sessionPanel {
	var all []*sessionPanel
	for id := range s.panels {
		all = append(all, s.tabPanels(id)...)
	}
	return all
}

// focusedPanel returns the session panel whose terminal owns the keyboard.
func (s *screen) focusedPanel() *sessionPanel {
	term, ok := s.focusedTerminal()
	if !ok {
		return nil
	}
	for _, panel := range s.allPanels() {
		if panel.term == term {
			return panel
		}
	}
	return nil
}

func (s *screen) splitFocusedPane(sideBySide bool) { s.splitPane(s.focusedPanel(), sideBySide) }
func (s *screen) closeFocusedPane()                { s.closePane(s.focusedPanel()) }
