// Package ui provides the desktop window for managing connection profiles.
// The layout mirrors mRemoteNG: menu bar, toolbar, connection tree on the left,
// session tabs with a Config panel below them, and a status bar.
package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	rdpclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/rdp"
	"github.com/mRemoteNG/mremoteng-go/internal/vault"
)

const (
	// sidebarRatio keeps the connection tree at its narrowest usable width;
	// Fyne still clamps it to the pane's minimum size.
	sidebarRatio = 0.15

	baseTitle       = "mRemoteNG Go"
	welcomeTabTitle = "Welcome"
	discardTitle    = "Discard changes?"
)

// Store is the persistence surface the window needs; tests substitute it.
type Store interface {
	Load() (profiles.Document, error)
	Save(profiles.Document) error
}

// pathStore is implemented by stores that can report their location.
type pathStore interface {
	Path() string
}

// confirm asks the user to approve an action that discards data. onDecline is
// optional and restores the previous UI state.
type confirm func(title, message string, onConfirm, onDecline func())

type screen struct {
	app     fyne.App
	win     fyne.Window
	store   Store
	confirm confirm

	all               []profiles.Profile
	folders           []string
	defaults          []profiles.FolderDefault
	collapsed         map[string]bool
	suppressBranch    bool
	rows              map[string]*treeRow
	index             *treeIndex
	selected          string
	selectedUID       string
	baseline          profiles.Profile
	writable          bool
	populating        bool
	lastProtocol      profiles.Protocol
	propertiesVisible bool
	suppressTree      bool

	filter *widget.Entry
	tree   *widget.Tree
	tabs   *container.DocTabs
	props  *propertiesPanel
	right  *fyne.Container
	status *widget.Label
	counts *widget.Label

	log        activityLog
	logConsole *logConsole
	logVisible bool
	logButton  *widget.Button
	bottom     *fyne.Container

	body             *fyne.Container
	split            *container.Split
	sidebarButton    *widget.Button
	configButton     *widget.Button
	sidebarCollapsed bool

	openTabs    map[string]*container.TabItem
	panels      map[string]*sessionPanel
	openShell   shellOpener
	launchRDP   rdpLauncher
	probeRDP    rdpProber
	dialVNC     vncDialer
	dialStream  streamDialer
	openBrowser func(string) error
	clipboard   func() fyne.Clipboard
	vault       *vault.Vault
	vaultPath   string
	embedder    embedder
	welcome     *container.TabItem
}

// buildLabel identifies the running build in the window title and About
// dialog. It is empty in tests.
var buildLabel string

// SetBuildLabel sets the version/commit shown in the title and About dialog.
func SetBuildLabel(label string) {
	buildLabel = label
}

// windowTitle is the title with the build label appended when present.
func windowTitle() string {
	if buildLabel == "" {
		return baseTitle
	}
	return baseTitle + " " + buildLabel
}

// NewWindow builds the application window and loads saved profiles.
func NewWindow(app fyne.App, store Store) fyne.Window {
	return newScreen(app, store).win
}

func newScreen(app fyne.App, store Store) *screen {
	s := &screen{
		app:               app,
		store:             store,
		win:               app.NewWindow(windowTitle()),
		openTabs:          map[string]*container.TabItem{},
		rows:              map[string]*treeRow{},
		collapsed:         map[string]bool{},
		panels:            map[string]*sessionPanel{},
		openShell:         defaultShellOpener,
		launchRDP:         defaultRDPLauncher,
		probeRDP:          rdpclient.Probe,
		dialVNC:           defaultVNCDialer,
		dialStream:        defaultStreamDialer,
		openBrowser:       openExternalBrowser,
		clipboard:         app.Clipboard,
		embedder:          embedX11,
		propertiesVisible: false,
	}
	s.confirm = func(title, message string, onConfirm, onDecline func()) {
		dialog.ShowConfirm(title, message, func(ok bool) {
			if ok {
				onConfirm()
				return
			}
			if onDecline != nil {
				onDecline()
			}
		}, s.win)
	}
	s.win.SetIcon(AppIcon())
	s.win.SetContent(s.build())
	s.win.SetMainMenu(s.menu())
	s.win.Resize(RestoreWindowSize(app.Preferences(), fyne.NewSize(1100, 720)))
	s.win.SetCloseIntercept(s.closeRequested)
	s.registerControlShortcuts()
	s.reload()
	return s
}

// build assembles the mRemoteNG-style frame around the panels.
func (s *screen) build() fyne.CanvasObject {
	s.buildTree()
	s.buildTabs()
	s.props = newPropertiesPanel(s)
	s.buildStatus()
	s.logConsole = newLogConsole(s)

	// Both panel toggles live in the status bar as flat icons: no toolbar row,
	// so the tabs and terminal own the full height. The View menu keeps the
	// labelled entries for discoverability.
	s.sidebarButton = widget.NewButtonWithIcon("", theme.MenuIcon(), s.toggleSidebar)
	s.sidebarButton.Importance = widget.LowImportance
	s.configButton = widget.NewButtonWithIcon("", theme.SettingsIcon(), s.toggleProperties)
	s.configButton.Importance = widget.LowImportance
	s.logButton = widget.NewButtonWithIcon("", theme.InfoIcon(), s.toggleLog)
	s.logButton.Importance = widget.LowImportance

	sidebarTitle := widget.NewLabelWithStyle("Connections", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	quick := widget.NewButtonWithIcon("", theme.ContentAddIcon(), s.requestQuickConnect)
	quick.Importance = widget.LowImportance
	sidebarHeader := container.NewVBox(container.NewBorder(nil, nil, sidebarTitle, quick), s.filter)
	left := container.NewBorder(container.NewPadded(sidebarHeader), container.NewPadded(s.counts), nil, nil, s.tree)
	s.right = container.NewBorder(nil, s.props.box, nil, nil, s.tabs)
	if !s.propertiesVisible {
		s.props.box.Hide()
	}
	s.split = container.NewHSplit(left, s.right)
	s.split.SetOffset(sidebarRatio)
	s.body = container.NewStack(s.split)

	toggles := container.NewHBox(s.sidebarButton, s.configButton, s.logButton)
	// Counts stay in the tree footer: a Border right-hand label was clipped by
	// long status messages.
	statusBar := container.NewBorder(nil, nil, toggles, nil, container.NewPadded(s.status))
	s.logConsole.box.Hide()
	s.bottom = container.NewVBox(s.logConsole.box, widget.NewSeparator(), statusBar)
	return container.NewBorder(nil, s.bottom, nil, nil, s.body)
}

func (s *screen) buildTree() {
	s.filter = widget.NewEntry()
	s.filter.SetPlaceHolder("Filter")
	s.filter.OnChanged = func(string) { s.refreshTree() }

	s.tree = widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID { return s.index.childUIDs(uid) },
		// The empty uid is Fyne's invisible root and must be a branch for its
		// children (the top-level folders and connections) to be walked.
		func(uid widget.TreeNodeID) bool { return uid == "" || isFolderUID(uid) },
		func(branch bool) fyne.CanvasObject { return newTreeRow(s) },
		func(uid widget.TreeNodeID, branch bool, object fyne.CanvasObject) {
			row := object.(*treeRow)
			if branch {
				row.setFolder(uid, s.index.label(uid))
				return
			}
			row.setConnection(uid, s.index.profiles[uid])
		},
	)
	s.tree.OnSelected = s.treeSelected
	s.tree.OnBranchOpened = func(uid string) { s.branchChanged(uid, false) }
	s.tree.OnBranchClosed = func(uid string) { s.branchChanged(uid, true) }
	// Fyne's tree rows consume hover, so the full name/endpoint is reported in
	// the status bar when a row is highlighted.
	s.tree.OnHighlighted = s.treeHighlighted
}

func (s *screen) buildTabs() {
	s.welcome = container.NewTabItem(welcomeTabTitle, welcomeContent())
	s.tabs = container.NewDocTabs(s.welcome)
	s.tabs.OnClosed = s.tabClosed
	s.tabs.OnSelected = s.tabSelected
}

func (s *screen) buildStatus() {
	s.status = widget.NewLabel("")
	s.status.Wrapping = fyne.TextWrapWord
	s.counts = widget.NewLabel("")
}

func (s *screen) menu() *fyne.MainMenu {
	file := fyne.NewMenu("File",
		fyne.NewMenuItem("Credentials", s.requestVault),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Import connections", s.requestImport),
		fyne.NewMenuItem("Export as XML", s.requestExportXML),
		fyne.NewMenuItem("Export as CSV", s.requestExportCSV),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quick connect", s.requestQuickConnect),
		fyne.NewMenuItem("New folder", s.requestNewFolder),
		fyne.NewMenuItem("New connection", s.requestNew),
		fyne.NewMenuItem("Save connection", s.save),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Delete connection", s.requestDelete),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", s.closeRequested),
	)
	view := fyne.NewMenu("View",
		fyne.NewMenuItem("Toggle connections panel", s.toggleSidebar),
		fyne.NewMenuItem("Toggle Config panel", s.toggleProperties),
	)
	tools := fyne.NewMenu("Tools",
		fyne.NewMenuItem("Connection file location", s.showStoreLocation),
		fyne.NewMenuItem("Diagnostics", s.requestDiagnostics),
		fyne.NewMenuItem("Repair connection file", s.requestRepairStore),
	)
	help := fyne.NewMenu("Help",
		fyne.NewMenuItem("About mRemoteNG Go", s.showAbout),
	)
	return fyne.NewMainMenu(file, view, tools, help)
}

// tabSelected keeps an embedded RDP window visible only on its own tab; Fyne
// tabs share one X11 window, so an orphaned child would cover the whole app.
func (s *screen) tabSelected(item *container.TabItem) {
	for id, panel := range s.panels {
		panel.setEmbeddedVisible(s.openTabs[id] == item)
	}
}

func (s *screen) tabClosed(item *container.TabItem) {
	for id, tab := range s.openTabs {
		if tab == item {
			// Closing a session tab ends its session, which removes the embedded
			// window instead of leaving it behind.
			if panel, known := s.panels[id]; known {
				panel.disconnectShell()
			}
			delete(s.openTabs, id)
			delete(s.panels, id)
		}
	}
	if len(s.tabs.Items) == 0 {
		s.tabs.Append(s.welcome)
		s.tabs.Select(s.welcome)
	}
}

func (s *screen) openSessionTab(profile profiles.Profile) {
	if tab, ok := s.openTabs[profile.ID]; ok {
		tab.Text = profile.Name
		if panel, known := s.panels[profile.ID]; known {
			panel.profile = profile
			// Rebuild the bar with the new name/endpoint; the slot keeps the live
			// terminal widget, so an open session survives a rename.
			tab.Content = panel.content()
		}
		s.tabs.Refresh()
		s.tabs.Select(tab)
		return
	}
	panel := s.newSessionPanel(profile)
	tab := container.NewTabItem(profile.Name, panel.content())
	s.openTabs[profile.ID] = tab
	s.panels[profile.ID] = panel
	s.tabs.Append(tab)
	s.tabs.Select(tab)
}

// toggleSidebar collapses or restores the connection tree pane. The toolbar
// button stays visible while collapsed, so the panel can always come back.
func (s *screen) toggleSidebar() {
	s.sidebarCollapsed = !s.sidebarCollapsed
	if s.sidebarCollapsed {
		s.body.Objects = []fyne.CanvasObject{s.right}
		s.setStatus("Connections panel hidden. Use the menu icon in the status bar, or View > Toggle connections panel, to show it again.")
	} else {
		s.body.Objects = []fyne.CanvasObject{s.split}
		s.setStatus("Connections panel shown.")
	}
	s.body.Refresh()
}

// toggleLog shows or hides the activity console with its own status-bar icon.
// registerControlShortcuts registers Ctrl+<key> so Fyne delivers the combination
// to the focused terminal instead of dropping it: key events carry no modifier
// information, and only a few control combinations are built in.
func (s *screen) registerControlShortcuts() {
	canvas := s.win.Canvas()
	if canvas == nil {
		return
	}
	for key := range controlCodes {
		if handledByFyne[key] {
			continue
		}
		key := key
		canvas.AddShortcut(&desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierControl},
			func(fyne.Shortcut) { s.sendControlToFocused(key) })
	}
}

// handledByFyne lists combinations Fyne already turns into built-in shortcuts;
// the terminal handles those in TypedShortcut.
var handledByFyne = map[fyne.KeyName]bool{
	fyne.KeyC: true, fyne.KeyV: true, fyne.KeyX: true,
	fyne.KeyA: true, fyne.KeyZ: true, fyne.KeyY: true,
}

// sendControlToFocused delivers a control byte to the focused terminal, if any.
func (s *screen) sendControlToFocused(key fyne.KeyName) {
	canvas := s.win.Canvas()
	if canvas == nil {
		return
	}
	term, ok := canvas.Focused().(*termWidget)
	if !ok {
		return
	}
	if code, known := controlCodes[key]; known {
		term.sendBytes([]byte{code})
	}
}

func (s *screen) toggleLog() {
	s.logVisible = !s.logVisible
	if s.logVisible {
		s.logConsole.box.Show()
		s.logConsole.refresh(s)
	} else {
		s.logConsole.box.Hide()
	}
	s.bottomRefresh()
	s.logConsole.box.Refresh()
}

// bottomRefresh repaints the area that hosts the console and the status line.
func (s *screen) bottomRefresh() {
	if s.bottom != nil {
		s.bottom.Refresh()
	}
}

func (s *screen) toggleProperties() {
	if s.propertiesVisible {
		s.hideProperties()
		return
	}
	s.showProperties()
}

// showProperties reveals the Config panel; New/selection use it so the editor
// appears exactly when it is needed.
func (s *screen) showProperties() {
	if s.propertiesVisible {
		return
	}
	s.propertiesVisible = true
	s.props.box.Show()
	s.right.Refresh()
}

func (s *screen) hideProperties() {
	if !s.propertiesVisible {
		return
	}
	s.propertiesVisible = false
	s.props.box.Hide()
	s.setStatus("Config panel hidden. Use the settings icon in the status bar to restore it.")
	s.right.Refresh()
}

// treeHighlighted reports the hovered row in the status bar.
func (s *screen) treeHighlighted(uid widget.TreeNodeID) {
	if uid == "" || s.index == nil {
		return
	}
	if isFolderUID(uid) {
		s.setStatus("Folder " + s.index.folderPath(uid))
		return
	}
	profile, known := s.index.profiles[uid]
	if !known {
		return
	}
	s.setStatus(s.endpointSummary(profile) + "  -  " + endpointText(profile))
}

func (s *screen) showStoreLocation() {
	path := "(store does not expose a path)"
	if aware, ok := s.store.(pathStore); ok {
		path = aware.Path()
	}
	dialog.ShowInformation("Connection file", path, s.win)
}

func (s *screen) showAbout() {
	title := "mRemoteNG Go"
	if buildLabel != "" {
		title += " " + buildLabel
	}
	dialog.ShowInformation("About mRemoteNG Go",
		title+"\n\n"+
			"A Go implementation of the mRemoteNG connection manager (menu, connection tree, "+
			"session tabs and Config panel).\n\n"+
			"Connection profiles are stored locally. SSH and RDP sessions are not implemented yet.",
		s.win)
}

func (s *screen) setCounts() {
	folders := map[string]bool{}
	for _, profile := range s.all {
		segments, err := profiles.FolderSegments(profile.Folder)
		if err != nil {
			continue
		}
		path := ""
		for _, segment := range segments {
			if path == "" {
				path = segment
			} else {
				path += "/" + segment
			}
			folders[path] = true
		}
	}
	s.counts.SetText(fmt.Sprintf("%d / %d", len(s.all), len(folders)))
}

func (s *screen) indexHas(uid string) bool {
	if s.index == nil {
		return false
	}
	if isFolderUID(uid) {
		_, ok := s.index.folders[uid]
		return ok
	}
	_, ok := s.index.profiles[uid]
	return ok
}

func (s *screen) selectUID(uid string) {
	s.suppressTree = true
	if s.indexHas(uid) {
		s.tree.Select(uid)
	} else {
		s.tree.UnselectAll()
	}
	s.suppressTree = false
}

func (s *screen) endpointSummary(profile profiles.Profile) string {
	return fmt.Sprintf("%s (%s %s:%d)", profile.Name, profile.Protocol, profile.Host, profile.Port)
}

func (s *screen) protocolChanged(choice string) {
	if s.populating {
		return
	}
	previous := profiles.DefaultPort(s.lastProtocol)
	s.lastProtocol = profiles.Protocol(choice)
	current := strings.TrimSpace(s.props.port.Text)
	if current == "" || current == strconv.Itoa(previous) {
		s.props.port.SetText(strconv.Itoa(profiles.DefaultPort(profiles.Protocol(choice))))
	}
}
