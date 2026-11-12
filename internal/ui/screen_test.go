package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	sshclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/ssh"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func (f *fakeStore) Load() (profiles.Document, error) {
	if f.loadErr != nil {
		return profiles.Document{}, f.loadErr
	}
	return profiles.Document{
		Profiles:  append([]profiles.Profile(nil), f.profiles...),
		Folders:   append([]string(nil), f.folders...),
		Defaults:  append([]profiles.FolderDefault(nil), f.defaults...),
		Collapsed: append([]string(nil), f.collapsed...),
	}, nil
}

func (f *fakeStore) Save(document profiles.Document) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.profiles = append([]profiles.Profile(nil), document.Profiles...)
	f.folders = append([]string(nil), document.Folders...)
	f.defaults = append([]profiles.FolderDefault(nil), document.Defaults...)
	f.collapsed = append([]string(nil), document.Collapsed...)
	f.saves++
	return nil
}

func (f *fakeConfirm) ask(title, _ string, onConfirm, onDecline func()) {
	f.asked = append(f.asked, title)
	if f.accept {
		onConfirm()
		return
	}
	f.declines++
	if onDecline != nil {
		onDecline()
	}
}

func TestNewScreen_EmptyStore_ShowsWelcomeAndGuidance(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	if s.index.connections != 0 {
		t.Fatalf("tree connections = %d, want 0", s.index.connections)
	}
	if len(s.tabs.Items) != 1 || s.tabs.Selected() != s.welcome {
		t.Errorf("tabs = %d, want only the welcome tab", len(s.tabs.Items))
	}
	if !strings.Contains(s.status.Text, "No connections yet") {
		t.Errorf("status = %q, want empty-state guidance", s.status.Text)
	}
	if s.props.proto.Selected != string(profiles.SSH) || s.props.port.Text != "22" {
		t.Errorf("defaults = %q/%q, want SSH/22", s.props.proto.Selected, s.props.port.Text)
	}
	if s.isDirty() {
		t.Error("fresh Config panel is dirty, want clean")
	}
}

func TestCloseRequested_UnsavedChanges_AsksBeforeClosing(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	guard := &fakeConfirm{accept: false}
	s.confirm = guard.ask
	s.props.name.SetText("Unsaved")

	s.closeRequested()

	if len(guard.asked) != 1 || guard.asked[0] != discardTitle {
		t.Fatalf("confirmations = %v, want one discard prompt", guard.asked)
	}
}

func (f *fakeShell) Output() io.Reader { return f.output }

func (f *fakeShell) Input(p []byte) error { f.input = append(f.input, p...); return nil }

func (f *fakeShell) Resize(size sshclient.TermSize) error {
	f.sizes = append(f.sizes, size)
	return nil
}

func (f *fakeShell) Close() error { f.closed = true; return nil }

func TestSessionContent_NarrowMinimumWidth_AvoidsHorizontalScroll(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	for _, protocol := range []string{"SSH", "RDP"} {
		profile := sampleProfile(t, "Office desktop", "Lab", protocol, "desk.example.com", 3389)
		panel := s.newSessionPanel(profile)
		minWidth := panel.content().MinSize().Width
		if minWidth > 720 {
			t.Errorf("%s content minimum width = %.0f, want <= 720 to fit the tab without horizontal scrolling", protocol, minWidth)
		}
	}
}

func TestWindowSizePreferences_RoundTripAndClamp(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	prefs := app.Preferences()
	defaultSize := fyne.NewSize(1100, 720)
	if got := RestoreWindowSize(prefs, defaultSize); got != defaultSize {
		t.Fatalf("initial size = %v, want the fallback %v", got, defaultSize)
	}
	SaveWindowSize(prefs, fyne.NewSize(1400, 900))
	if got := RestoreWindowSize(prefs, defaultSize); got != fyne.NewSize(1400, 900) {
		t.Fatalf("restored = %v, want the saved size", got)
	}
	SaveWindowSize(prefs, fyne.NewSize(10, 10))
	if got := RestoreWindowSize(prefs, defaultSize); got.Width < 720 || got.Height < 480 {
		t.Fatalf("restored = %v, want the minimum size clamp", got)
	}
}

func (f *fakeVNC) Size() (int, int) { return f.pixels.Bounds().Dx(), f.pixels.Bounds().Dy() }

func (f *fakeVNC) Name() string { return "fake" }

func (f *fakeVNC) Image() *image.RGBA { return f.pixels }

func (f *fakeVNC) ReadMessage() error {
	if f.err != nil {
		return f.err
	}
	return io.EOF
}

func (f *fakeVNC) Pointer(x, y int, mask byte) error {
	f.pointer = append(f.pointer, [3]int{x, y, int(mask)})
	return nil
}

func (f *fakeVNC) Close() error { f.closed = true; return nil }

func (f *fakeStream) Read([]byte) (int, error) { return 0, io.EOF }

func (f *fakeStream) Write(p []byte) (int, error) {
	f.written = append(f.written, p...)
	return len(p), nil
}

func (f *fakeStream) Close() error { f.closed = true; return nil }

func TestActivityLog_RecordsAndBoundsEntries(t *testing.T) {
	var log activityLog
	for i := 0; i < logLimit+5; i++ {
		log.add("info", "entry")
	}
	if len(log.entries) != logLimit {
		t.Fatalf("entries = %d, want the limit %d", len(log.entries), logLimit)
	}
	if !strings.Contains(log.entries[0], "entry") {
		t.Errorf("entry = %q, want timestamped content", log.entries[0])
	}
	if lines := strings.Count(log.render(), "\n") + 1; lines != logVisibleLines {
		t.Errorf("rendered lines = %d, want the fixed %d", lines, logVisibleLines)
	}
}

func TestActivityLog_CollapsesMultilineMessages(t *testing.T) {
	var log activityLog
	log.add("error", "line one\nline two\n\tline three   with   spaces")
	entry := log.entries[0]
	if strings.Count(entry, "\n") != 0 {
		t.Fatalf("entry kept newlines: %q", entry)
	}
	if !strings.Contains(entry, "line one line two line three with spaces") {
		t.Errorf("entry = %q, want one collapsed line", entry)
	}
	long := strings.Repeat("x", 2000)
	log.add("error", long)
	last := log.entries[len(log.entries)-1]
	if len([]rune(last)) > 320 {
		t.Errorf("entry length = %d, want a bounded line", len([]rune(last)))
	}
}

func TestLogConsole_StaysCompactWithLongErrors(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	for i := 0; i < 20; i++ {
		s.setProblem(strings.Repeat("freeRDP output line ", 40))
	}
	if height := s.logConsole.box.MinSize().Height; height > 220 {
		t.Fatalf("log panel minimum height = %.0f, want a compact fixed panel", height)
	}
	if lines := strings.Count(s.logConsole.text.Text, "\n") + 1; lines > logVisibleLines {
		t.Errorf("rendered lines = %d, want at most %d", lines, logVisibleLines)
	}
}

func TestLogConsole_TogglesWithItsOwnIcon(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	if !s.logConsole.box.Hidden || s.logVisible {
		t.Fatal("activity console starts visible")
	}
	s.setStatus("hello")
	if !strings.Contains(s.logConsole.text.Text, "hello") {
		t.Errorf("console text = %q, want the status entry", s.logConsole.text.Text)
	}
	s.setProblem("boom")
	if !strings.Contains(s.logConsole.text.Text, "boom") {
		t.Errorf("console text = %q, want the error entry", s.logConsole.text.Text)
	}

	test.Tap(s.logButton)
	if s.logConsole.box.Hidden || !s.logVisible {
		t.Fatal("log icon did not show the console")
	}
	test.Tap(s.logButton)
	if !s.logConsole.box.Hidden {
		t.Error("log icon did not hide the console")
	}
}

func TestRepairStore_BacksUpAndStartsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	s := newTestScreen(t, profiles.NewStore(path))
	if s.ws.writable {
		t.Fatal("writes are enabled although the store is corrupt")
	}
	s.repairStore(path)

	if !s.ws.writable {
		t.Fatal("repair did not re-enable writes")
	}
	if len(s.ws.profiles) != 0 {
		t.Errorf("profiles after repair = %d, want 0", len(s.ws.profiles))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	backups := 0
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".bak-") {
			backups++
		}
	}
	if backups != 1 {
		t.Errorf("backups = %d, want the damaged file preserved", backups)
	}
}

func (f *fakeClipboard) Content() string { return f.content }

func (f *fakeClipboard) SetContent(content string) { f.content = content }

func TestDiagnosticsText_ReportsStateWithoutSecrets(t *testing.T) {
	text := diagnosticsText("0.4.0 (abc)", "/tmp/profiles.json", 3, 2, "unlocked", 1)
	for _, want := range []string{"mremoteng-go 0.4.0 (abc)", "/tmp/profiles.json", "profiles: 3, folders: 2", "vault: unlocked", "open sessions: 1"} {
		if !strings.Contains(text, want) {
			t.Errorf("diagnostics = %q, missing %q", text, want)
		}
	}
}

func TestShutdownSessions_ClosesLiveSessions(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	stream := &fakeStream{}
	streamPanel := s.newSessionPanel(sampleProfile(t, "T", "", "TELNET", "t.example.com", 23))
	streamPanel.stream = stream
	vncSession := &fakeVNC{pixels: image.NewRGBA(image.Rect(0, 0, 4, 4))}
	vncPanel := s.newSessionPanel(sampleProfile(t, "V", "", "VNC", "v.example.com", 5900))
	vncPanel.vnc = vncSession
	area := &fakeArea{id: 5}
	vncPanel.embedded = area
	s.panels["t"] = streamPanel
	s.panels["v"] = vncPanel

	if got := s.activeSessionCount(); got != 2 {
		t.Fatalf("active sessions = %d, want 2", got)
	}
	s.shutdownSessions()
	if !stream.closed || !vncSession.closed {
		t.Error("shutdown left a session open")
	}
	if !area.destroyed {
		t.Error("shutdown left the embedded window behind")
	}
	if got := s.activeSessionCount(); got != 0 {
		t.Errorf("active sessions after shutdown = %d", got)
	}
}

func TestWorkspace_RestoresCollapsedFolders(t *testing.T) {
	connection := sampleProfile(t, "web", "Team/Prod", "SSH", "web.example.com", 22)
	store := &fakeStore{
		profiles:  []profiles.Profile{connection},
		folders:   []string{"Team", "Team/Prod"},
		collapsed: []string{"Team"},
	}
	s := newTestScreen(t, store)
	if s.ws.collapsed["Team"] != true {
		t.Fatalf("collapsed = %v, want Team restored", s.ws.collapsed)
	}
	if s.tree.IsBranchOpen(folderUID("Team")) {
		t.Error("Team is expanded although it was collapsed last time")
	}
}

func TestBranchChanged_TracksCollapseAndPersistsOnClose(t *testing.T) {
	connection := sampleProfile(t, "web", "Team", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{connection}, folders: []string{"Team"}}
	s := newTestScreen(t, store)
	s.branchChanged(folderUID("Team"), true)
	if !s.ws.collapsed["Team"] {
		t.Fatal("collapse was not tracked")
	}
	if err := s.ws.store.Save(s.ws.document(s.ws.profiles, s.ws.folders)); err != nil {
		t.Fatalf("save: %v", err)
	}
	document, err := store.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(document.Collapsed) != 1 || document.Collapsed[0] != "Team" {
		t.Fatalf("collapsed = %v, want Team", document.Collapsed)
	}
}

func TestAboutText_ShowsFeatureStatus(t *testing.T) {
	text := aboutText()
	for _, want := range []string{"Available:", "SSH", "RDP", "VNC", "Pending:", "Serial ports", "Windows"} {
		if !strings.Contains(text, want) {
			t.Errorf("about text missing %q: %q", want, text)
		}
	}
}

func TestAboutHyperlink_OpensAMailComposer(t *testing.T) {
	link := aboutHyperlink()
	if link.Scheme != "mailto" {
		t.Fatalf("scheme = %q, want mailto", link.Scheme)
	}
	if got := link.Opaque + link.Path; got != ContactAddress {
		t.Errorf("address = %q, want %q", got, ContactAddress)
	}
}

func TestDesktopEntry_MatchesInstalledPaths(t *testing.T) {
	entry := DesktopEntry("/opt/mremoteng/bin/mremoteng", "mremoteng-go")
	if got := DesktopEntryExec(entry); got != "/opt/mremoteng/bin/mremoteng" {
		t.Errorf("Exec = %q", got)
	}
	for _, want := range []string{"Icon=mremoteng-go", "StartupWMClass=mRemoteNG Go", "Categories=Network;RemoteAccess;"} {
		if !strings.Contains(entry, want) {
			t.Errorf("entry missing %q", want)
		}
	}
}

func TestAppIcon_IsEmbeddedSVG(t *testing.T) {
	icon := AppIcon()
	if icon == nil {
		t.Fatal("AppIcon returned nil")
	}
	if icon.Name() != "mremoteng-go.svg" {
		t.Errorf("icon name = %q", icon.Name())
	}
	if content := string(icon.Content()); !strings.Contains(content, "<svg") {
		t.Errorf("icon content is not SVG: %q", content[:min(len(content), 40)])
	}
}

func TestSoberTheme_ProvidesPalette(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	sober := SoberTheme()
	for _, name := range []fyne.ThemeColorName{
		theme.ColorNameBackground,
		theme.ColorNameInputBackground,
		theme.ColorNamePrimary,
		theme.ColorNameSeparator,
	} {
		if sober.Color(name, theme.VariantDark) == nil {
			t.Errorf("%s has no colour", name)
		}
	}
	if sober.Color(theme.ColorNameBackground, theme.VariantDark) == sober.Color(theme.ColorNameBackground, theme.VariantLight) {
		t.Error("dark and light backgrounds are identical")
	}
}

func TestSoberTheme_UsesCompactType(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	sober := SoberTheme()
	if got := sober.Size(theme.SizeNameText); got != 12 {
		t.Errorf("text size = %v, want 12", got)
	}
	if base := theme.DefaultTheme().Size(theme.SizeNameText); sober.Size(theme.SizeNameText) >= base {
		t.Errorf("sober text size %v is not smaller than the default %v", sober.Size(theme.SizeNameText), base)
	}
}

func TestToggleSidebar_CollapsesAndRestoresTheTree(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	if s.body == nil || s.split == nil {
		t.Fatal("build did not wire the sidebar containers")
	}
	if s.split.Offset != sidebarRatio {
		t.Errorf("split offset = %v, want %v", s.split.Offset, sidebarRatio)
	}
	if len(s.body.Objects) != 1 || s.body.Objects[0] != s.split {
		t.Fatalf("body starts with %d objects, want the split", len(s.body.Objects))
	}

	test.Tap(s.sidebarButton)
	if !s.sidebarCollapsed || len(s.body.Objects) != 1 || s.body.Objects[0] != s.right {
		t.Fatalf("collapsed body = %v, want only the tab area", s.body.Objects)
	}

	s.toggleSidebar()
	if s.sidebarCollapsed || s.body.Objects[0] != s.split {
		t.Fatalf("restored body = %v, want the split", s.body.Objects)
	}
}

func (f *fakeRDPSession) Wait() error {
	<-f.done
	return nil
}

func (f *fakeRDPSession) Close() error { f.closed = true; close(f.done); return nil }

func rdpProfile(t *testing.T) profiles.Profile {
	t.Helper()
	profile := sampleProfile(t, "Office desktop", "", "RDP", "desk.example.com", 3389)
	profile.Username = "kas"
	return profile
}

func (f *fakeArea) ID() uint64 { return f.id }

func (f *fakeArea) Destroy() { f.destroyed = true }

func (f *fakeArea) SetVisible(v bool) { f.visible = v }

func TestToggleProperties_HiddenByDefault_ShowsOnNewConnection(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	if !s.props.box.Hidden || s.propertiesVisible {
		t.Fatal("Config panel starts visible, want it hidden")
	}
	s.startNew()
	if s.props.box.Hidden || !s.propertiesVisible {
		t.Fatal("New connection did not reveal the Config panel")
	}
	test.Tap(s.configButton)
	if !s.props.box.Hidden {
		t.Error("settings toggle did not hide the Config panel")
	}
	test.Tap(s.configButton)
	if s.props.box.Hidden {
		t.Error("settings toggle did not restore the Config panel")
	}
}

func TestSelectConnection_KeepsConfigPanelHidden(t *testing.T) {
	first := sampleProfile(t, "Production web", "", "SSH", "web.example.com", 22)
	second := sampleProfile(t, "Office desktop", "", "RDP", "desk.example.com", 3389)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{first, second}})
	if !s.props.box.Hidden {
		t.Fatal("Config panel started visible")
	}
	s.tree.Select(connectionUID(first.ID))
	s.tree.Select(connectionUID(second.ID))
	if !s.props.box.Hidden {
		t.Error("switching connections reopened the Config panel")
	}
	s.startNew()
	if s.props.box.Hidden {
		t.Error("New connection did not reveal the Config panel")
	}
	// An explicitly shown panel also stays as the user left it.
	s.tree.Select(connectionUID(first.ID))
	if s.props.box.Hidden {
		t.Error("switching connections hid a panel the user had open")
	}
}
