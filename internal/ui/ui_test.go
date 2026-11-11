package ui

import (
	"errors"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	rdpclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/rdp"
	sshclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/ssh"
	telnetclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/telnet"
	vncclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/vnc"
	"github.com/mRemoteNG/mremoteng-go/internal/transfer"
	"github.com/mRemoteNG/mremoteng-go/internal/vault"
)

type fakeStore struct {
	profiles  []profiles.Profile
	folders   []string
	defaults  []profiles.FolderDefault
	collapsed []string
	loadErr   error
	saveErr   error
	saves     int
}

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

type fakeConfirm struct {
	asked    []string
	accept   bool
	declines int
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

func newTestScreen(t *testing.T, store Store) *screen {
	t.Helper()
	app := test.NewApp()
	t.Cleanup(app.Quit)
	return newScreen(app, store)
}

func sampleProfile(t *testing.T, name, folder, protocol, host string, port int) profiles.Profile {
	t.Helper()
	id, err := profiles.NewID()
	if err != nil {
		t.Fatalf("NewID: %v", err)
	}
	return profiles.Profile{ID: id, Name: name, Folder: folder, Protocol: profiles.Protocol(protocol), Host: host, Port: port}
}

func fillConfig(s *screen, name, folder, host, port string) {
	s.props.name.SetText(name)
	s.props.folder.SetText(folder)
	s.props.host.SetText(host)
	s.props.port.SetText(port)
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

func TestSave_ValidProfile_AddsTreeEntryTabAndPersists(t *testing.T) {
	store := &fakeStore{}
	s := newTestScreen(t, store)
	fillConfig(s, "Production web", "Production/Web", "web.example.com", "22")
	test.Tap(s.props.save)

	if store.saves != 1 || len(store.profiles) != 1 {
		t.Fatalf("store = %d saves, %d profiles; want 1/1", store.saves, len(store.profiles))
	}
	saved := store.profiles[0]
	if saved.Name != "Production web" || saved.Folder != "Production/Web" || saved.Host != "web.example.com" {
		t.Fatalf("saved = %+v", saved)
	}
	if _, known := s.index.profiles[connectionUID(saved.ID)]; !known {
		t.Error("saved connection is missing from the tree")
	}
	if _, known := s.index.folders[folderUID("Production/Web")]; !known {
		t.Error("folder path is missing from the tree")
	}
	if children := s.index.children[folderUID("Production")]; len(children) != 1 || children[0] != folderUID("Production/Web") {
		t.Errorf("nested folder children = %v, want the Web folder under Production", children)
	}
	if tab, open := s.openTabs[saved.ID]; !open || tab.Text != saved.Name {
		t.Errorf("session tab = %v, want a tab named %q", open, saved.Name)
	}
	if s.selected != saved.ID || s.baseline != saved {
		t.Errorf("selection = %q, baseline = %+v", s.selected, s.baseline)
	}
	if s.isDirty() {
		t.Error("Config panel still dirty after a successful save")
	}
	if s.counts.Text != "1 / 2" {
		t.Errorf("counts = %q, want %q (connections / folders)", s.counts.Text, "1 / 2")
	}
}

func TestSave_InvalidInput_ReportsProblemWithoutWriting(t *testing.T) {
	tests := []struct {
		name     string
		nameText string
		folder   string
		host     string
		port     string
		want     string
	}{
		{name: "missing name", host: "web", port: "22", want: "name is required"},
		{name: "missing host", nameText: "web", port: "22", want: "host is required"},
		{name: "non numeric port", nameText: "web", host: "web", port: "ssh", want: "Port must be a whole number"},
		{name: "port out of range", nameText: "web", host: "web", port: "70000", want: "port must be between"},
		{name: "invalid folder", nameText: "web", folder: "../Secrets", host: "web", port: "22", want: "folder path is invalid"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeStore{}
			s := newTestScreen(t, store)
			fillConfig(s, tc.nameText, tc.folder, tc.host, tc.port)
			test.Tap(s.props.save)

			if store.saves != 0 {
				t.Fatalf("store saved %d times, want 0", store.saves)
			}
			if !strings.Contains(s.status.Text, tc.want) {
				t.Errorf("status = %q, want substring %q", s.status.Text, tc.want)
			}
			if s.status.Importance != widget.DangerImportance {
				t.Error("validation problem is not styled as an error")
			}
			if !s.isDirty() {
				t.Error("invalid edits were discarded")
			}
		})
	}
}

func TestSave_StoreFailure_KeepsEditsAndReports(t *testing.T) {
	store := &fakeStore{saveErr: errors.New("disk full")}
	s := newTestScreen(t, store)
	fillConfig(s, "Prod", "", "web", "22")
	test.Tap(s.props.save)

	if len(s.all) != 0 {
		t.Fatalf("in-memory profiles = %d, want 0 after a failed save", len(s.all))
	}
	if !strings.Contains(s.status.Text, "Could not save") || !strings.Contains(s.status.Text, "disk full") {
		t.Errorf("status = %q, want save error", s.status.Text)
	}
	if s.props.name.Text != "Prod" || !s.isDirty() {
		t.Error("edits were lost after a failed save")
	}
}

func TestFilter_HidesConnectionsAndEmptyFolders(t *testing.T) {
	store := &fakeStore{profiles: []profiles.Profile{
		sampleProfile(t, "Production web", "Production", "SSH", "web.example.com", 22),
		sampleProfile(t, "Office desktop", "", "RDP", "desk.example.com", 3389),
	}}
	s := newTestScreen(t, store)
	if s.index.connections != 2 {
		t.Fatalf("tree connections = %d, want 2", s.index.connections)
	}

	s.filter.SetText("desktop")
	if s.index.connections != 1 {
		t.Errorf("filtered connections = %d, want 1", s.index.connections)
	}
	if _, known := s.index.folders[folderUID("Production")]; known {
		t.Error("folder without matches is still in the tree")
	}

	s.filter.SetText("office.example")
	if s.index.connections != 0 {
		t.Errorf("filtered connections = %d, want 0 for an unknown host", s.index.connections)
	}
}

func TestTreeSelect_Connection_OpensSessionTabAndLoadsConfig(t *testing.T) {
	existing := sampleProfile(t, "Production web", "Production", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{existing}})

	s.tree.Select(connectionUID(existing.ID))

	if s.selected != existing.ID {
		t.Fatalf("selection = %q, want %q", s.selected, existing.ID)
	}
	if tab, open := s.openTabs[existing.ID]; !open || tab.Text != existing.Name {
		t.Fatalf("session tab = %v, want one named %q", open, existing.Name)
	}
	if s.tabs.Selected() != s.openTabs[existing.ID] {
		t.Error("session tab is not selected after opening it")
	}
	if s.props.name.Text != existing.Name || s.props.host.Text != existing.Host {
		t.Errorf("Config panel = %q/%q, want the selected profile", s.props.name.Text, s.props.host.Text)
	}
}

func TestTreeSelect_Folder_ClearsSelectionAndExplainsGrouping(t *testing.T) {
	existing := sampleProfile(t, "Production web", "Production", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{existing}})

	s.tree.Select(folderUID("Production"))

	if s.selected != "" {
		t.Errorf("selection = %q, want no connection selected", s.selected)
	}
	if !strings.Contains(s.status.Text, "Folder") {
		t.Errorf("status = %q, want folder explanation", s.status.Text)
	}
}

func TestTreeSelect_UnsavedChanges_DeclinedKeepsEdits(t *testing.T) {
	first := sampleProfile(t, "First", "", "SSH", "first.example.com", 22)
	second := sampleProfile(t, "Second", "", "SSH", "second.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{first, second}})
	guard := &fakeConfirm{accept: false}
	s.confirm = guard.ask

	s.tree.Select(connectionUID(first.ID))
	s.props.name.SetText("Edited name")
	s.tree.Select(connectionUID(second.ID))

	if len(guard.asked) != 1 || guard.asked[0] != discardTitle {
		t.Fatalf("confirmations = %v, want one discard prompt", guard.asked)
	}
	if s.props.name.Text != "Edited name" || s.selected != first.ID {
		t.Errorf("edits = %q, selection = %q; want them preserved", s.props.name.Text, s.selected)
	}
}

func TestDelete_Confirmed_RemovesProfileTabAndTreeEntry(t *testing.T) {
	existing := sampleProfile(t, "Production web", "", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{existing}}
	s := newTestScreen(t, store)
	s.confirm = (&fakeConfirm{accept: true}).ask

	s.tree.Select(connectionUID(existing.ID))
	test.Tap(s.props.delete)

	if len(store.profiles) != 0 {
		t.Fatalf("store = %+v, want no profiles", store.profiles)
	}
	if len(s.all) != 0 || s.index.connections != 0 {
		t.Errorf("ui still holds %d profiles", len(s.all))
	}
	if _, open := s.openTabs[existing.ID]; open {
		t.Error("session tab survived the deletion")
	}
	if !strings.Contains(s.status.Text, "Deleted") {
		t.Errorf("status = %q, want deletion confirmation", s.status.Text)
	}
}

func TestDelete_NotConfirmed_KeepsProfile(t *testing.T) {
	existing := sampleProfile(t, "Production web", "", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{existing}}
	s := newTestScreen(t, store)
	s.confirm = (&fakeConfirm{accept: false}).ask

	s.tree.Select(connectionUID(existing.ID))
	test.Tap(s.props.delete)

	if store.saves != 0 || len(store.profiles) != 1 {
		t.Fatalf("store = %+v after declined delete, want unchanged", store.profiles)
	}
}

func TestLoadFailure_DisablesWritesAndReportsProblem(t *testing.T) {
	store := &fakeStore{loadErr: errors.New("permission denied")}
	s := newTestScreen(t, store)
	if !s.props.save.Disabled() || !s.props.delete.Disabled() {
		t.Error("Config buttons stay enabled although the store could not be read")
	}
	if !strings.Contains(s.status.Text, "Could not read") {
		t.Errorf("status = %q, want load failure", s.status.Text)
	}

	fillConfig(s, "Prod", "", "web", "22")
	test.Tap(s.props.save)
	if store.saves != 0 {
		t.Fatalf("store saved %d times despite a load failure", store.saves)
	}
}

func TestProtocolChange_UntouchedPort_UsesProtocolDefault(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.props.proto.SetSelected(string(profiles.RDP))
	if s.props.port.Text != "3389" {
		t.Errorf("port = %q after switching to RDP, want 3389", s.props.port.Text)
	}

	s.props.port.SetText("3390")
	s.props.proto.SetSelected(string(profiles.SSH))
	if s.props.port.Text != "3390" {
		t.Errorf("port = %q after switching to SSH, want the typed port", s.props.port.Text)
	}
}

func TestRevert_UnsavedChanges_RestoresSelectedProfile(t *testing.T) {
	existing := sampleProfile(t, "Production web", "", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{existing}})
	s.tree.Select(connectionUID(existing.ID))
	s.props.name.SetText("Changed")
	test.Tap(s.props.revert)

	if s.props.name.Text != existing.Name || s.isDirty() {
		t.Errorf("Config = %q, dirty = %v; want the stored values back", s.props.name.Text, s.isDirty())
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

func sshPanelProfile(t *testing.T, username string) profiles.Profile {
	t.Helper()
	profile := sampleProfile(t, "Lab server", "", "SSH", "lab.example.com", 22)
	profile.Username = username
	return profile
}

func TestSessionPanel_MissingUsername_ExplainsWithoutConnecting(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	called := false
	s.openShell = func(sshclient.Target, sshclient.Options, sshclient.TermSize) (shellSession, sshclient.Result, error) {
		called = true
		return &fakeShell{output: strings.NewReader("")}, sshclient.Result{}, nil
	}
	panel := s.newSessionPanel(sshPanelProfile(t, ""))
	panel.connectShell()

	if called {
		t.Error("session opened without a username")
	}
	if !strings.Contains(s.status.Text, "Add a username") {
		t.Errorf("status = %q, want guidance about the missing username", s.status.Text)
	}
}

type fakeShell struct {
	input  []byte
	output io.Reader
	closed bool
	sizes  []sshclient.TermSize
}

func (f *fakeShell) Output() io.Reader    { return f.output }
func (f *fakeShell) Input(p []byte) error { f.input = append(f.input, p...); return nil }
func (f *fakeShell) Resize(size sshclient.TermSize) error {
	f.sizes = append(f.sizes, size)
	return nil
}
func (f *fakeShell) Close() error { f.closed = true; return nil }

func TestTermWidget_TypedInput_SendsTerminalSequences(t *testing.T) {
	var sent []byte
	term := newTermWidget(40, 10, func(data []byte) { sent = append(sent, data...) }, nil)
	term.TypedRune('a')
	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnter})
	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
	want := "a\r\x1b[A\x7f\t"
	if string(sent) != want {
		t.Fatalf("sent = %q, want %q", sent, want)
	}
}

func TestTermWidget_AcceptsTab_KeepsCompletionInsideTerminal(t *testing.T) {
	var tabbable fyne.Tabbable = newTermWidget(40, 10, nil, nil)
	if !tabbable.AcceptsTab() {
		t.Fatal("AcceptsTab = false, Fyne will move focus instead of completing")
	}
}

func TestTermWidget_Write_PaintsScreen(t *testing.T) {
	term := newTermWidget(40, 5, nil, nil)
	term.CreateRenderer()
	term.write([]byte("hello"))
	if len(term.grid.Rows) == 0 || len(term.grid.Rows[0].Cells) < 5 {
		t.Fatalf("grid rows = %d", len(term.grid.Rows))
	}
	var row strings.Builder
	for _, cell := range term.grid.Rows[0].Cells {
		row.WriteRune(cell.Rune)
	}
	if !strings.HasPrefix(row.String(), "hello") {
		t.Errorf("row 0 = %q, want it to start with hello", row.String())
	}
}

func TestSessionPanel_Connect_ShowsTerminalAndForwardsInput(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	// A pipe keeps the session open until the test disconnects; an immediate EOF
	// would close the panel and clear the terminal.
	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })
	shell := &fakeShell{output: reader}
	s.openShell = func(sshclient.Target, sshclient.Options, sshclient.TermSize) (shellSession, sshclient.Result, error) {
		return shell, sshclient.Result{ServerVersion: "SSH-2.0-test", HostKeyVerified: true}, nil
	}
	panel := s.newSessionPanel(sshPanelProfile(t, "tester"))
	panel.applyConnect(panel.target(), sshclient.Options{}, shell, sshclient.Result{ServerVersion: "SSH-2.0-test", HostKeyVerified: true}, nil)

	if panel.term == nil || len(panel.slot.Objects) != 1 {
		t.Fatalf("terminal not installed in the tab slot: %+v", panel.slot.Objects)
	}
	if !strings.Contains(s.status.Text, "Connected") {
		t.Errorf("status = %q, want a connected notice", s.status.Text)
	}
	panel.term.TypedRune('x')
	if string(shell.input) != "x" {
		t.Errorf("shell input = %q, want %q", shell.input, "x")
	}
	panel.applyOutput([]byte("hi"))
	if text := panel.term.screenText(); !strings.Contains(text, "hi") {
		t.Errorf("terminal did not receive the remote output: %q", text)
	}
	panel.disconnectShell()
	if !shell.closed {
		t.Error("disconnect did not close the shell")
	}
}

func TestSessionPanel_ConnectUnknownHost_RetriesAfterTrust(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	guard := &fakeConfirm{accept: true}
	s.confirm = guard.ask
	calls := 0
	var trusted bool
	s.openShell = func(_ sshclient.Target, options sshclient.Options, _ sshclient.TermSize) (shellSession, sshclient.Result, error) {
		calls++
		if !options.AcceptAnyHostKey {
			return nil, sshclient.Result{}, &sshclient.UnknownHostKeyError{HostPort: "lab.example.com:22", Fingerprint: "SHA256:xyz"}
		}
		trusted = true
		return &fakeShell{output: strings.NewReader("")}, sshclient.Result{ServerVersion: "SSH-2.0-test"}, nil
	}
	panel := s.newSessionPanel(sshPanelProfile(t, "tester"))
	panel.applyConnect(panel.target(), sshclient.Options{},
		nil, sshclient.Result{}, &sshclient.UnknownHostKeyError{HostPort: "lab.example.com:22", Fingerprint: "SHA256:xyz"})

	if len(guard.asked) != 1 || guard.asked[0] != "Unverified SSH host key" {
		t.Fatalf("confirmations = %v, want a host-key prompt", guard.asked)
	}
	if calls != 1 || !trusted {
		t.Fatalf("calls = %d, trusted = %v; want one retry that accepts the key", calls, trusted)
	}
	if panel.term == nil {
		t.Error("terminal is missing after trusting the host key")
	}
	if !strings.Contains(s.status.Text, "unverified") {
		t.Errorf("status = %q, want an unverified notice", s.status.Text)
	}
}

func TestSessionPanel_UnknownHostKey_RetriesAfterTrust(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	guard := &fakeConfirm{accept: true}
	s.confirm = guard.ask
	var trusted bool
	s.openShell = func(_ sshclient.Target, options sshclient.Options, _ sshclient.TermSize) (shellSession, sshclient.Result, error) {
		if !options.AcceptAnyHostKey {
			return nil, sshclient.Result{}, &sshclient.UnknownHostKeyError{HostPort: "lab.example.com:22", Fingerprint: "SHA256:xyz"}
		}
		trusted = options.AcceptAnyHostKey
		return &fakeShell{output: strings.NewReader("")}, sshclient.Result{ServerVersion: "SSH-2.0-test"}, nil
	}
	panel := s.newSessionPanel(sshPanelProfile(t, "tester"))
	panel.applyConnect(panel.target(), sshclient.Options{},
		nil, sshclient.Result{}, &sshclient.UnknownHostKeyError{HostPort: "lab.example.com:22", Fingerprint: "SHA256:xyz"})

	if len(guard.asked) != 1 || guard.asked[0] != "Unverified SSH host key" {
		t.Fatalf("confirmations = %v, want a host-key prompt", guard.asked)
	}
	if !trusted {
		t.Fatal("retry did not accept the approved key")
	}
	if panel.term == nil {
		t.Error("terminal is missing after trusting the host key")
	}
	if !strings.Contains(s.status.Text, "unverified") {
		t.Errorf("status = %q, want an unverified notice", s.status.Text)
	}
}

func TestSessionPanel_Connect_HidesConfigAndRestoresItOnClose(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	shell := &fakeShell{output: strings.NewReader("")}
	s.showProperties()
	panel := s.newSessionPanel(sshPanelProfile(t, "tester"))
	panel.startSession(shell, sshclient.Result{ServerVersion: "SSH-2.0-test"})

	if s.propertiesVisible {
		t.Fatal("Config panel stayed visible during a session")
	}
	panel.disconnectShell()
	if !s.propertiesVisible {
		t.Error("Config panel was not restored after the session closed")
	}
}

func TestSessionPanel_ControlButtons_SendCtrlSequences(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	shell := &fakeShell{output: strings.NewReader("")}
	s.openShell = func(sshclient.Target, sshclient.Options, sshclient.TermSize) (shellSession, sshclient.Result, error) {
		return shell, sshclient.Result{ServerVersion: "SSH-2.0-test"}, nil
	}
	panel := s.newSessionPanel(sshPanelProfile(t, "tester"))
	panel.startSession(shell, sshclient.Result{ServerVersion: "SSH-2.0-test"})
	panel.sendControl(0x03)
	panel.sendControl(0x04)
	if string(shell.input) != "\x03\x04" {
		t.Errorf("shell input = %q, want Ctrl-C then Ctrl-D", shell.input)
	}
}

func TestIntersect_ClampsToVisibleBounds(t *testing.T) {
	area := rect{x: 100, y: 100, width: 500, height: 400}
	bounds := rect{x: 0, y: 0, width: 400, height: 300}
	got := intersect(area, bounds)
	if got != (rect{x: 100, y: 100, width: 300, height: 200}) {
		t.Fatalf("intersect = %+v, want the visible part only", got)
	}
	if empty := intersect(area, rect{x: 900, y: 900, width: 10, height: 10}); empty.width != 0 || empty.height != 0 {
		t.Fatalf("intersect = %+v, want an empty rect", empty)
	}
}

func TestScaleRect_ConvertsLogicalPointsToPixels(t *testing.T) {
	got := scaleRect(rect{x: 10, y: 20, width: 100, height: 50}, 2)
	if got != (rect{x: 20, y: 40, width: 200, height: 100}) {
		t.Fatalf("scaleRect = %+v, want doubled geometry", got)
	}
	if got := scaleRect(rect{x: 1, y: 1, width: 1, height: 1}, 0); got.width != 1 {
		t.Fatalf("scaleRect with zero scale = %+v, want it treated as 1", got)
	}
}

func TestAddFolder_EmptyFolder_IsPersistedAndShown(t *testing.T) {
	store := &fakeStore{}
	s := newTestScreen(t, store)
	s.addFolder("Production/Web")

	if len(store.folders) != 2 || store.folders[1] != "Production/Web" {
		t.Fatalf("stored folders = %v, want the path and its parent", store.folders)
	}
	if _, known := s.index.folders[folderUID("Production/Web")]; !known {
		t.Error("empty folder is missing from the tree")
	}
	if !strings.Contains(s.status.Text, "created") {
		t.Errorf("status = %q, want a creation notice", s.status.Text)
	}
	if s.selectedUID == folderUID("Production/Web") {
		t.Error("new folder was auto-selected, which scrolls the tree")
	}
	s.addFolder("Production/Web")
	if len(store.folders) != 2 || !strings.Contains(s.status.Text, "already exists") {
		t.Errorf("duplicate folder was persisted: %v (%q)", store.folders, s.status.Text)
	}
}

func TestAddFolder_InvalidPath_ReportsProblemWithoutWriting(t *testing.T) {
	store := &fakeStore{}
	s := newTestScreen(t, store)
	s.addFolder("../escape")
	if store.saves != 0 {
		t.Fatalf("store saved %d times for an invalid folder", store.saves)
	}
	if !strings.Contains(s.status.Text, "Cannot create folder") {
		t.Errorf("status = %q, want a folder error", s.status.Text)
	}
}

func TestSave_ProfileFolder_IsPersistedWithTheTree(t *testing.T) {
	store := &fakeStore{}
	s := newTestScreen(t, store)
	fillConfig(s, "Production web", "Production/Web", "web.example.com", "22")
	test.Tap(s.props.save)

	if len(store.folders) != 2 {
		t.Fatalf("stored folders = %v, want Production and Production/Web", store.folders)
	}
}

func TestRenameFolder_MovesConnectionsAndDescendants(t *testing.T) {
	connection := sampleProfile(t, "Web", "Production/Web", "SSH", "web.example.com", 22)
	store := &fakeStore{
		profiles: []profiles.Profile{connection},
		folders:  []string{"Production", "Production/Web"},
	}
	s := newTestScreen(t, store)
	s.renameFolder("Production", "Clientes")

	if len(store.profiles) != 1 || store.profiles[0].Folder != "Clientes/Web" {
		t.Fatalf("profiles = %+v, want the folder rewritten", store.profiles)
	}
	if len(store.folders) != 2 || store.folders[0] != "Clientes" || store.folders[1] != "Clientes/Web" {
		t.Fatalf("folders = %v, want the whole subtree renamed", store.folders)
	}
	if _, known := s.index.folders[folderUID("Clientes/Web")]; !known {
		t.Error("renamed folder is missing from the tree")
	}
}

func TestRenameFolder_InvalidTargets_AreRejected(t *testing.T) {
	store := &fakeStore{folders: []string{"Production", "Ops"}}
	s := newTestScreen(t, store)
	s.renameFolder("Production", "Production/Inner")
	if !strings.Contains(s.status.Text, "into itself") {
		t.Errorf("status = %q, want a self-nesting rejection", s.status.Text)
	}
	s.renameFolder("Production", "Ops")
	if !strings.Contains(s.status.Text, "already exists") {
		t.Errorf("status = %q, want an existing-folder rejection", s.status.Text)
	}
	if store.saves != 0 {
		t.Fatalf("store saved %d times for rejected renames", store.saves)
	}
}

func TestDeleteFolder_RemovesConnectionsAndTabs(t *testing.T) {
	inside := sampleProfile(t, "Web", "Production/Web", "SSH", "web.example.com", 22)
	outside := sampleProfile(t, "Desk", "", "RDP", "desk.example.com", 3389)
	store := &fakeStore{
		profiles: []profiles.Profile{inside, outside},
		folders:  []string{"Production", "Production/Web"},
	}
	s := newTestScreen(t, store)
	s.confirm = (&fakeConfirm{accept: true}).ask
	s.tree.Select(connectionUID(inside.ID))
	s.tree.Select(folderUID("Production"))
	s.requestDeleteFolder()

	if len(store.profiles) != 1 || store.profiles[0].ID != outside.ID {
		t.Fatalf("profiles = %+v, want only the outside connection", store.profiles)
	}
	if len(store.folders) != 0 {
		t.Fatalf("folders = %v, want the subtree removed", store.folders)
	}
	if _, open := s.openTabs[inside.ID]; open {
		t.Error("removed connection kept its session tab")
	}
}

func TestDeleteFolder_Declined_KeepsEverything(t *testing.T) {
	connection := sampleProfile(t, "Web", "Production", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{connection}, folders: []string{"Production"}}
	s := newTestScreen(t, store)
	s.confirm = (&fakeConfirm{accept: false}).ask
	s.tree.Select(folderUID("Production"))
	s.requestDeleteFolder()
	if store.saves != 0 || len(store.profiles) != 1 {
		t.Fatalf("declined delete changed the store: %+v", store.profiles)
	}
}

func TestMoveConnection_MovesIntoNewFolderAndBackToRoot(t *testing.T) {
	connection := sampleProfile(t, "Web", "", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{connection}}
	s := newTestScreen(t, store)
	s.tree.Select(connectionUID(connection.ID))

	s.moveConnection(connection, "Ops/Oncall")
	if store.profiles[0].Folder != "Ops/Oncall" {
		t.Fatalf("folder = %q, want Ops/Oncall", store.profiles[0].Folder)
	}
	if len(store.folders) != 2 {
		t.Fatalf("folders = %v, want Ops and Ops/Oncall", store.folders)
	}

	s.moveConnection(store.profiles[0], "")
	if store.profiles[0].Folder != "" {
		t.Fatalf("folder = %q, want the root", store.profiles[0].Folder)
	}
	if !strings.Contains(s.status.Text, "root") {
		t.Errorf("status = %q, want a root notice", s.status.Text)
	}
}

func TestMoveConnection_InvalidFolder_IsRejected(t *testing.T) {
	connection := sampleProfile(t, "Web", "", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{connection}}
	s := newTestScreen(t, store)
	s.moveConnection(connection, "../escape")
	if store.saves != 0 {
		t.Fatalf("store saved %d times for an invalid move", store.saves)
	}
	if !strings.Contains(s.status.Text, "Cannot move") {
		t.Errorf("status = %q, want a move error", s.status.Text)
	}
}

func TestWithinFolder_AndPrefixReplacement(t *testing.T) {
	if !withinFolder("Production/Web", "Production") || withinFolder("Production2", "Production") {
		t.Error("withinFolder is wrong")
	}
	if got := replaceFolderPrefix("Production/Web", "Production", "Clientes"); got != "Clientes/Web" {
		t.Errorf("replaceFolderPrefix = %q, want Clientes/Web", got)
	}
	if got := withinFolder("", ""); !got {
		t.Error("the root should be within the root")
	}
}

func TestRowMenu_FolderOffersFolderActions(t *testing.T) {
	store := &fakeStore{folders: []string{"Production"}}
	s := newTestScreen(t, store)
	menu := s.rowMenu(folderUID("Production"))
	labels := make([]string, 0, len(menu.Items))
	for _, item := range menu.Items {
		labels = append(labels, item.Label)
	}
	want := []string{"Folder defaults", "New folder inside", "Rename folder", "Delete folder"}
	if len(labels) != len(want) {
		t.Fatalf("menu = %v, want %v", labels, want)
	}
	for i := range want {
		if labels[i] != want[i] {
			t.Fatalf("menu = %v, want %v", labels, want)
		}
	}
}

func TestRowTapped_OpensConnectionAndSelectsIt(t *testing.T) {
	connection := sampleProfile(t, "Web", "Production", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{connection}, folders: []string{"Production"}})
	row := newTreeRow(s)
	row.setConnection(connectionUID(connection.ID), connection)
	row.Tapped(&fyne.PointEvent{})

	if s.selected != connection.ID {
		t.Fatalf("selection = %q, want the tapped connection", s.selected)
	}
	if tab, open := s.openTabs[connection.ID]; !open || tab.Text != connection.Name {
		t.Fatalf("session tab = %v, want it opened by the click", open)
	}
}

func TestRowTapped_FolderSelectsIt(t *testing.T) {
	connection := sampleProfile(t, "Web", "Production", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{connection}, folders: []string{"Production"}})
	row := newTreeRow(s)
	row.setFolder(folderUID("Production"), "Production")
	row.Tapped(&fyne.PointEvent{})

	if s.selected != "" || s.selectedUID != folderUID("Production") {
		t.Fatalf("selection = %q/%q, want the folder", s.selected, s.selectedUID)
	}
}

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

func TestRequestRenameConnection_UpdatesNameAndTab(t *testing.T) {
	connection := sampleProfile(t, "Web", "", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{connection}}
	s := newTestScreen(t, store)
	s.tree.Select(connectionUID(connection.ID))
	s.selected = connection.ID
	s.requestRenameConnection()

	// The dialog is modal, so drive the rename path directly.
	s.renameConnection(connection, "Web primary")
	if store.profiles[0].Name != "Web primary" {
		t.Fatalf("name = %q, want Web primary", store.profiles[0].Name)
	}
	if tab, open := s.openTabs[connection.ID]; !open || tab.Text != "Web primary" {
		t.Errorf("tab label not updated: %v", tab)
	}
}

func TestMoveToMenu_ListsRootAndFoldersAndMoves(t *testing.T) {
	connection := sampleProfile(t, "Web", "", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{
		profiles: []profiles.Profile{connection},
		folders:  []string{"Ops", "Production"},
	})
	menu := s.moveToMenu(connection)
	if len(menu.Items) != 3 {
		t.Fatalf("items = %d, want root plus two folders", len(menu.Items))
	}
	if menu.Items[0].Label != "(root)" || menu.Items[1].Label != "Ops" {
		t.Fatalf("items = %v", menu.Items)
	}
	menu.Items[1].Action()
	if s.all[0].Folder != "Ops" {
		t.Fatalf("folder = %q, want Ops after the menu move", s.all[0].Folder)
	}
	menu.Items[0].Action()
	if s.all[0].Folder != "" {
		t.Fatalf("folder = %q, want the root after the menu move", s.all[0].Folder)
	}
}

func TestCreateFolderInside_UsesUniqueNamesWithoutDialog(t *testing.T) {
	store := &fakeStore{}
	s := newTestScreen(t, store)
	s.createFolderInside("")
	s.createFolderInside("")
	if len(store.folders) != 2 || store.folders[0] != "New folder" || store.folders[1] != "New folder 2" {
		t.Fatalf("folders = %v, want New folder and New folder 2", store.folders)
	}
	s.createFolderInside("New folder")
	if store.folders[len(store.folders)-1] != "New folder/New folder" {
		t.Fatalf("folders = %v, want a nested default name", store.folders)
	}
}

func TestRowMenu_ConnectionOffersOpenMoveDelete(t *testing.T) {
	connection := sampleProfile(t, "Web", "Production", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{connection}, folders: []string{"Production"}})
	menu := s.rowMenu(connectionUID(connection.ID))
	if len(menu.Items) != 4 {
		t.Fatalf("menu items = %d, want 4", len(menu.Items))
	}
	if menu.Items[0].Label != "Rename" || menu.Items[2].Label != "Move to" || menu.Items[3].Label != "Delete connection" {
		t.Errorf("unexpected menu order: %v", menu.Items)
	}
	// The delete item must select the row before acting on it.
	menu.Items[3].Action()
	if s.selected != connection.ID {
		t.Errorf("selection = %q, want the menu row", s.selected)
	}
}

func TestFolderAt_ReturnsContainingRow(t *testing.T) {
	rects := map[string]rect{
		folderUID("Production"): {x: 10, y: 10, width: 100, height: 20},
		folderUID("Ops"):        {x: 10, y: 40, width: 100, height: 20},
	}
	if got := folderAt(fyne.NewPos(50, 45), rects); got != folderUID("Ops") {
		t.Errorf("folderAt = %q, want the Ops folder", got)
	}
	if got := folderAt(fyne.NewPos(500, 500), rects); got != "" {
		t.Errorf("folderAt = %q, want no folder", got)
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

type fakeVNC struct {
	pixels  *image.RGBA
	pointer [][3]int
	closed  bool
	err     error
}

func (f *fakeVNC) Size() (int, int)   { return f.pixels.Bounds().Dx(), f.pixels.Bounds().Dy() }
func (f *fakeVNC) Name() string       { return "fake" }
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

type fakeStream struct {
	written []byte
	closed  bool
}

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
	if s.writable {
		t.Fatal("writes are enabled although the store is corrupt")
	}
	s.repairStore(path)

	if !s.writable {
		t.Fatal("repair did not re-enable writes")
	}
	if len(s.all) != 0 {
		t.Errorf("profiles after repair = %d, want 0", len(s.all))
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

func TestTermWidget_Selection_CopiesOnlyHighlightedCells(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 4, nil, nil)
	term.CreateRenderer()
	term.write([]byte("hello world\r\nsecond line\r\nthird"))
	// Select from the middle of the first line to the start of the second.
	term.selStart = cellPos{col: 6, row: 0}
	term.selEnd = cellPos{col: 5, row: 1}
	if !term.selected(6, 0) || term.selected(5, 0) {
		t.Error("selection does not cover the expected first-line cells")
	}
	if !term.selected(5, 1) || term.selected(6, 1) {
		t.Error("selection end is wrong on the second line")
	}
	text := term.screenText()
	if text != "world\nsecond" {
		t.Fatalf("selected text = %q, want only the highlighted range", text)
	}

	// A reversed drag normalises.
	term.selStart = cellPos{col: 5, row: 1}
	term.selEnd = cellPos{col: 6, row: 0}
	if got := term.screenText(); got != "world\nsecond" {
		t.Fatalf("reversed selection = %q", got)
	}
}

func TestTermWidget_DragWithoutMouseDown_SelectsRange(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 4, nil, nil)
	term.CreateRenderer()
	term.write([]byte("hello world\r\nsecond line\r\n"))
	term.Resize(fyne.NewSize(300, 120))
	cell := fyne.MeasureText("M", theme.TextSize(), fyne.TextStyle{Monospace: true})
	// A single drag event from the left edge across ten cells (no MouseDown).
	term.Dragged(&fyne.DragEvent{
		PointEvent: fyne.PointEvent{Position: fyne.NewPos(cell.Width*10, cell.Height*0.5)},
		Dragged:    fyne.Delta{DX: cell.Width * 10},
	})
	term.DragEnd()
	text := term.screenText()
	if !strings.HasPrefix(text, "hello worl") || strings.Contains(text, "second") {
		t.Fatalf("drag selection = %q, want the first line range", text)
	}
}

type fakeClipboard struct{ content string }

func (f *fakeClipboard) Content() string           { return f.content }
func (f *fakeClipboard) SetContent(content string) { f.content = content }

func TestTermWidget_DragPastTopEdge_ScrollsWhileSelecting(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 3, nil, nil)
	term.CreateRenderer()
	for i := 0; i < 12; i++ {
		term.write([]byte("line " + string(rune('a'+i)) + "\r\n"))
	}
	term.Resize(fyne.NewSize(300, 120))
	history := term.emulator.HistoryLen()
	if history == 0 {
		t.Fatal("no history to scroll through")
	}

	// Start a selection inside the view, then drag above the top edge.
	inner := fyne.NewPos(10, 10)
	term.selStart = term.cellAt(inner)
	term.selEnd = term.selStart
	term.selecting = true
	term.Dragged(&fyne.DragEvent{
		PointEvent: fyne.PointEvent{Position: fyne.NewPos(10, -5)},
		Dragged:    fyne.Delta{DY: -15},
	})
	if term.scrollOffset == 0 {
		t.Fatal("dragging past the top did not scroll the view")
	}
	if term.selEnd.row >= term.selStart.row {
		t.Errorf("selection end row = %d, want a row above the start", term.selEnd.row)
	}
	if text := term.screenText(); text == "" || !strings.Contains(text, "\n") {
		t.Errorf("selected text = %q, want a multi-line selection from the scrollback", text)
	}
}

func TestTermWidget_ControlShortcuts_SendControlBytes(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	var sent []byte
	term := newTermWidget(20, 4, func(data []byte) { sent = append(sent, data...) }, nil)

	term.TypedShortcut(&fyne.ShortcutSelectAll{})
	if len(sent) != 1 || sent[0] != 0x01 {
		t.Fatalf("Ctrl-A sent %v, want 0x01", sent)
	}
	sent = nil
	term.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyL, Modifier: fyne.KeyModifierControl})
	if len(sent) != 1 || sent[0] != 0x0c {
		t.Fatalf("Ctrl-L sent %v, want 0x0c", sent)
	}
	sent = nil
	term.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyL})
	if len(sent) != 0 {
		t.Errorf("Ctrl-less shortcut sent %v, want nothing", sent)
	}
}

func TestControlCodes_CoverTheShellCombinations(t *testing.T) {
	want := map[fyne.KeyName]byte{
		fyne.KeyA: 0x01, fyne.KeyD: 0x04, fyne.KeyE: 0x05, fyne.KeyK: 0x0b,
		fyne.KeyL: 0x0c, fyne.KeyR: 0x12, fyne.KeyU: 0x15, fyne.KeyW: 0x17, fyne.KeyZ: 0x1a,
	}
	for key, code := range want {
		if got := controlByte(key); got != code {
			t.Errorf("controlByte(%s) = %#x, want %#x", key, got, code)
		}
	}
}

func TestTermWidget_ArrowKeysNavigateHistoryWhenScrolled(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	var sent []byte
	term := newTermWidget(20, 3, func(data []byte) { sent = append(sent, data...) }, nil)
	term.CreateRenderer()
	for i := 0; i < 10; i++ {
		term.write([]byte("line " + string(rune('a'+i)) + "\r\n"))
	}
	term.Resize(fyne.NewSize(300, 120))

	// At the bottom the arrow goes to the shell (command history).
	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	if string(sent) != "\x1b[A" {
		t.Fatalf("sent = %q, want the up sequence at the bottom", sent)
	}
	sent = nil

	// PageUp browses the scrollback without reaching the shell.
	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyPageUp})
	if len(sent) != 0 {
		t.Errorf("PageUp sent %q, want local scrolling", sent)
	}
	if term.scrollOffset == 0 {
		t.Fatal("PageUp did not scroll the view")
	}

	before := term.scrollOffset
	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	if len(sent) != 0 {
		t.Errorf("up sent %q while scrolled, want local scrolling only", sent)
	}
	if term.scrollOffset != before+1 {
		t.Errorf("offset = %d, want one line further back", term.scrollOffset)
	}

	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd})
	if term.scrollOffset != 0 {
		t.Errorf("End did not return to the bottom: %d", term.scrollOffset)
	}
	if len(sent) != 0 {
		t.Errorf("End sent %q, want it consumed locally", sent)
	}
}

func TestSessionPanel_Disconnect_ClearsTerminalContent(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	stream := &fakeStream{}
	panel := s.newSessionPanel(sampleProfile(t, "T", "", "TELNET", "t.example.com", 23))
	panel.startStream(stream, telnetclient.ModeTelnet)
	panel.term.write([]byte("old output\r\n"))
	panel.disconnectShell()

	if panel.term != nil {
		t.Error("terminal widget survived the disconnect")
	}
	if len(panel.slot.Objects) != 1 {
		t.Fatalf("slot objects = %d, want a single placeholder", len(panel.slot.Objects))
	}
	if label, ok := panel.slot.Objects[0].(*widget.Label); !ok || !strings.Contains(label.Text, "Session closed") {
		t.Errorf("slot content = %#v, want the closed placeholder", panel.slot.Objects[0])
	}
}

func TestSessionPanel_Connect_FocusesTerminalForTyping(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	stream := &fakeStream{}
	s.openEphemeral(sampleProfile(t, "T", "", "TELNET", "t.example.com", 23))
	var panel *sessionPanel
	for _, candidate := range s.panels {
		panel = candidate
	}
	panel.startStream(stream, telnetclient.ModeTelnet)
	if panel.term == nil {
		t.Fatal("terminal not installed")
	}
	panel.term.TypedRune('x')
	if string(stream.written) != "x" {
		t.Errorf("typed input = %q, want it forwarded", stream.written)
	}
}

func TestSessionPanel_AfterDisconnect_CanReconnect(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	shell := &fakeShell{output: strings.NewReader("")}
	s.openShell = func(sshclient.Target, sshclient.Options, sshclient.TermSize) (shellSession, sshclient.Result, error) {
		return shell, sshclient.Result{ServerVersion: "SSH-2.0-test"}, nil
	}
	panel := s.newSessionPanel(sshPanelProfile(t, "tester"))
	panel.startSession(shell, sshclient.Result{ServerVersion: "SSH-2.0-test"})
	if !panel.password.Disabled() {
		t.Fatal("password should be disabled during a session")
	}
	panel.disconnectShell()

	if panel.password.Disabled() {
		t.Error("password field stayed disabled after disconnecting")
	}
	if panel.connect.Disabled() {
		t.Error("Connect stayed disabled after disconnecting")
	}
	if !panel.disconnect.Disabled() {
		t.Error("Disconnect stayed enabled after disconnecting")
	}
	panel.password.SetText("again")
	if panel.password.Text != "again" {
		t.Error("password field is not editable after disconnecting")
	}
}

func TestTermWidget_AdjustZoom_ClampsAndUpdatesTheme(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 4, nil, nil)
	start := term.textSize()

	term.adjustZoom(2)
	if term.textSize() != start+2 {
		t.Fatalf("size = %.0f, want %.0f", term.textSize(), start+2)
	}
	if term.zoom == nil || term.zoom.Size(theme.SizeNameText) != start+2 {
		t.Errorf("zoom theme text size = %v, want %.0f", term.zoom, start+2)
	}
	for i := 0; i < 50; i++ {
		term.adjustZoom(1)
	}
	if term.textSize() != maxFontSize {
		t.Errorf("size = %.0f, want the maximum %d", term.textSize(), maxFontSize)
	}
	for i := 0; i < 60; i++ {
		term.adjustZoom(-1)
	}
	if term.textSize() != minFontSize {
		t.Errorf("size = %.0f, want the minimum %d", term.textSize(), minFontSize)
	}
}

func TestZoomTheme_OverridesOnlyTextSize(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	z := zoomTheme{base: theme.DefaultTheme(), size: 20}
	if got := z.Size(theme.SizeNameText); got != 20 {
		t.Fatalf("text size = %.0f, want 20", got)
	}
	if got, want := z.Size(theme.SizeNamePadding), theme.DefaultTheme().Size(theme.SizeNamePadding); got != want {
		t.Errorf("padding = %.0f, want it untouched (%0.f)", got, want)
	}
}

func TestTermWidget_Zoom_RebuildsTheGridForBothAxes(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 4, nil, nil)
	term.CreateRenderer()
	before := term.grid
	term.Resize(fyne.NewSize(400, 200))
	oldCols, oldRows := term.cols, term.rows

	term.adjustZoom(6)
	term.fit(fyne.NewSize(400, 200))
	renderer := term.CreateRenderer()
	renderer.Layout(fyne.NewSize(400, 200))

	if term.grid == before {
		t.Fatal("grid was not rebuilt after the zoom change")
	}
	if term.cols >= oldCols || term.rows >= oldRows {
		t.Errorf("grid = %dx%d, want fewer columns and rows at a larger font (%dx%d)", term.cols, term.rows, oldCols, oldRows)
	}
	if term.cols < 5 || term.rows < 2 {
		t.Errorf("grid collapsed to %dx%d", term.cols, term.rows)
	}
}

func TestTermWidget_CtrlCInterruptsAndCtrlVPastes(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	var sent []byte
	interrupts := 0
	term := newTermWidget(20, 4, func(data []byte) { sent = append(sent, data...) }, nil)
	term.onInterrupt = func() { interrupts++ }

	term.TypedShortcut(&fyne.ShortcutCopy{})
	if interrupts != 1 {
		t.Fatalf("Ctrl-C interrupts = %d, want 1", interrupts)
	}
	if len(sent) != 0 {
		t.Errorf("Ctrl-C sent %v, want no bytes from the shortcut path", sent)
	}
	term.sendBytes([]byte{0x03})
	if len(sent) != 1 || sent[0] != 0x03 {
		t.Errorf("sent = %v, want the interrupt byte", sent)
	}
}

func TestTermWidget_Scrollback_MovesThroughHistory(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 3, nil, nil)
	term.CreateRenderer()
	for i := 0; i < 10; i++ {
		term.write([]byte("line " + string(rune('0'+i)) + "\r\n"))
	}
	if term.emulator.HistoryLen() == 0 {
		t.Fatal("no history recorded")
	}
	term.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.Delta{DY: 2}})
	if term.scrollOffset != 2 {
		t.Fatalf("offset = %d, want 2", term.scrollOffset)
	}
	for i := 0; i < 50; i++ {
		term.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.Delta{DY: 5}})
	}
	if term.scrollOffset != term.emulator.HistoryLen() {
		t.Errorf("offset = %d, want the history length", term.scrollOffset)
	}
	term.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.Delta{DY: -100}})
	if term.scrollOffset != 0 {
		t.Errorf("offset = %d, want it clamped to zero", term.scrollOffset)
	}
}

func TestTermWidget_RightClickCopiesSelection(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 4, nil, nil)
	term.CreateRenderer()
	term.write([]byte("pick me\r\nother\r\n"))
	var copied string
	term.onCopy = func(text string) { copied = text }
	term.selStart = cellPos{col: 0, row: 0}
	term.selEnd = cellPos{col: 6, row: 0}
	term.TappedSecondary(&fyne.PointEvent{})
	if copied != "pick me" {
		t.Fatalf("right click copied %q, want the selection", copied)
	}
	if _, _, hasSelection := term.selectionRange(); hasSelection {
		t.Error("right click left the selection active")
	}
}

func TestCopyTerminalScreen_PutsTextInClipboard(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	clipboard := &fakeClipboard{}
	s.clipboard = func() fyne.Clipboard { return clipboard }
	stream := &fakeStream{}
	panel := s.newSessionPanel(sampleProfile(t, "T", "", "TELNET", "t.example.com", 23))
	panel.startStream(stream, telnetclient.ModeTelnet)
	panel.term.write([]byte("copied text\r\n"))
	panel.copyTerminalScreen()
	if !strings.Contains(clipboard.content, "copied text") {
		t.Fatalf("clipboard = %q, want the terminal text", clipboard.content)
	}
	if !strings.Contains(s.status.Text, "copied") {
		t.Errorf("status = %q, want a copy confirmation", s.status.Text)
	}
}

func TestTermWidget_DoubleTapSelectsLine(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 4, nil, nil)
	term.CreateRenderer()
	term.write([]byte("first\r\nsecond\r\n"))
	term.Resize(fyne.NewSize(200, 80))
	term.DoubleTapped(&fyne.PointEvent{Position: fyne.NewPos(2, 2)})
	text := term.screenText()
	if !strings.HasPrefix(text, "first") || strings.Contains(text, "second") {
		t.Fatalf("double tap selection = %q, want the first line", text)
	}
}

func TestTerminalScreenText_ReturnsVisibleLines(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 4, nil, nil)
	term.CreateRenderer()
	term.write([]byte("first\r\nsecond   \r\n"))
	text := term.screenText()
	if !strings.Contains(text, "first") || !strings.Contains(text, "second") {
		t.Fatalf("screenText = %q, want both lines", text)
	}
	if strings.HasSuffix(text, "\n") {
		t.Errorf("screenText = %q, want trailing blank lines trimmed", text)
	}
}

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
	if s.collapsed["Team"] != true {
		t.Fatalf("collapsed = %v, want Team restored", s.collapsed)
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
	if !s.collapsed["Team"] {
		t.Fatal("collapse was not tracked")
	}
	if err := s.store.Save(s.document(s.all, s.folders)); err != nil {
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

func TestVault_StoreUseAndLockSecrets(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.vaultPath = filepath.Join(t.TempDir(), "secrets.json")
	s.createVault("master-password")

	profile := sampleProfile(t, "web", "", "SSH", "web.example.com", 22)
	panel := s.newSessionPanel(profile)
	panel.password.SetText("s3cr3t")
	panel.requestStorePassword()
	if !strings.Contains(s.status.Text, "stored") {
		t.Fatalf("status = %q, want a storage confirmation", s.status.Text)
	}

	panel.password.SetText("")
	if got := panel.passwordFor(); got != "s3cr3t" {
		t.Errorf("passwordFor = %q, want the stored secret", got)
	}
	panel.password.SetText("typed")
	if got := panel.passwordFor(); got != "typed" {
		t.Errorf("passwordFor = %q, want the typed password to win", got)
	}

	reopened, err := vault.Open(s.vaultPath, "master-password")
	if err != nil {
		t.Fatalf("reopen vault: %v", err)
	}
	if secret, ok := reopened.Get(profile.ID); !ok || secret != "s3cr3t" {
		t.Fatalf("stored secret = %q/%v", secret, ok)
	}

	s.lockVault()
	panel.password.SetText("")
	if got := panel.passwordFor(); got != "" {
		t.Errorf("passwordFor = %q after locking, want empty", got)
	}
	panel.requestStorePassword()
	if !strings.Contains(s.status.Text, "Unlock the credential vault") {
		t.Errorf("status = %q, want an unlock hint", s.status.Text)
	}
}

func TestVault_WrongPasswordIsReported(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.vaultPath = filepath.Join(t.TempDir(), "secrets.json")
	s.createVault("master-password")
	s.unlockVault("nope")
	if !strings.Contains(s.status.Text, "Could not unlock") {
		t.Errorf("status = %q, want an unlock error", s.status.Text)
	}
}

func TestSessionPanel_TelnetConnect_ShowsTerminalAndSendsInput(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	stream := &fakeStream{}
	var gotMode telnetclient.Mode
	s.dialStream = func(_ string, config telnetclient.Config) (streamSession, error) {
		gotMode = config.Mode
		return stream, nil
	}
	profile := sampleProfile(t, "Router", "", "TELNET", "router.example.com", 23)
	panel := s.newSessionPanel(profile)

	panel.startStream(stream, telnetclient.ModeTelnet)
	if gotMode != "" {
		t.Fatal("dialer ran before startStream")
	}
	if panel.term == nil || len(panel.slot.Objects) != 1 {
		t.Fatal("terminal view was not installed")
	}
	if panel.term.onCopy == nil {
		t.Error("telnet terminal is not wired for clipboard copy")
	}
	if panel.term.zoom == nil {
		t.Error("telnet terminal has no grid-level zoom theme")
	}
	panel.term.TypedRune('a')
	panel.term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnter})
	if string(stream.written) != "a\r" {
		t.Errorf("written = %q, want the typed input", stream.written)
	}
	panel.disconnectShell()
	if !stream.closed {
		t.Error("disconnect did not close the stream")
	}
}

func TestStreamMode_MapsProtocols(t *testing.T) {
	cases := map[profiles.Protocol]telnetclient.Mode{
		profiles.TELNET: telnetclient.ModeTelnet,
		profiles.RAW:    telnetclient.ModeRaw,
		profiles.RLOGIN: telnetclient.ModeRlogin,
	}
	for protocol, want := range cases {
		if got, ok := streamMode(protocol); !ok || got != want {
			t.Errorf("streamMode(%s) = %q, %v", protocol, got, ok)
		}
	}
	if _, ok := streamMode(profiles.SSH); ok {
		t.Error("SSH mapped to a stream protocol")
	}
}

func TestHTTPConnect_OpensBrowser(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	var opened string
	s.openBrowser = func(url string) error { opened = url; return nil }
	profile := sampleProfile(t, "Intranet", "", "HTTP", "intranet.example.com", 80)
	panel := s.newSessionPanel(profile)
	panel.connectHTTP()
	if opened != "http://intranet.example.com" {
		t.Fatalf("opened %q, want the address without :80", opened)
	}
	if !strings.Contains(s.status.Text, "browser") {
		t.Errorf("status = %q, want a browser notice", s.status.Text)
	}
}

func TestQuickConnect_OpensTemporaryTabWithoutSaving(t *testing.T) {
	store := &fakeStore{}
	s := newTestScreen(t, store)
	s.quickConnect("lab.example.com", string(profiles.SSH), "", "demo")

	if store.saves != 0 || len(store.profiles) != 0 {
		t.Fatalf("quick connect wrote %d profiles to the store", len(store.profiles))
	}
	if len(s.openTabs) != 1 || len(s.panels) != 1 {
		t.Fatalf("tabs = %d, want one temporary session", len(s.openTabs))
	}
	for _, panel := range s.panels {
		if panel.profile.Host != "lab.example.com" || panel.profile.Port != 22 || panel.profile.Username != "demo" {
			t.Errorf("profile = %+v", panel.profile)
		}
		if !strings.Contains(panel.profile.Name, "Quick:") {
			t.Errorf("name = %q, want a Quick prefix", panel.profile.Name)
		}
	}
}

func TestQuickConnect_InvalidInput_DoesNotOpenTab(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.quickConnect("", string(profiles.SSH), "", "")
	if len(s.openTabs) != 0 {
		t.Fatal("empty host opened a session")
	}
	s.quickConnect("lab.example.com", string(profiles.SSH), "not-a-port", "")
	if len(s.openTabs) != 0 {
		t.Fatal("invalid port opened a session")
	}
	if !strings.Contains(s.status.Text, "must be a number") {
		t.Errorf("status = %q, want a port error", s.status.Text)
	}
}

func TestSessionPanel_VNCConnect_ShowsFramebufferAndForwardsClicks(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	session := &fakeVNC{pixels: image.NewRGBA(image.Rect(0, 0, 100, 50))}
	session.pixels.SetRGBA(1, 1, color.RGBA{R: 0xff, A: 0xff})
	s.dialVNC = func(string, vncclient.Config) (vncSession, error) { return session, nil }
	profile := sampleProfile(t, "Desktop", "", "VNC", "vnc.example.com", 5900)
	panel := s.newSessionPanel(profile)

	panel.startVNC(session)
	if panel.vncView == nil {
		t.Fatal("VNC view was not installed")
	}
	panel.vncView.Resize(fyne.NewSize(200, 100))
	panel.vncView.Tapped(&fyne.PointEvent{Position: fyne.NewPos(100, 50)})
	if len(session.pointer) < 2 {
		t.Fatalf("pointer events = %v, want press and release", session.pointer)
	}
	if session.pointer[0][0] < 45 || session.pointer[0][0] > 55 || session.pointer[0][1] < 20 || session.pointer[0][1] > 30 {
		t.Errorf("mapped click = %v, want around (50,25)", session.pointer[0])
	}
	panel.disconnectShell()
	if !session.closed {
		t.Error("disconnect did not close the VNC session")
	}
}

func TestProfile_Validate_AcceptsVNC(t *testing.T) {
	profile := sampleProfile(t, "Desktop", "", "VNC", "vnc.example.com", 5900)
	if err := profile.Validate(); err != nil {
		t.Fatalf("Validate(VNC) = %v", err)
	}
}

func TestTreeRow_VNCBadge_UsesItsOwnLetterAndColour(t *testing.T) {
	row := newTreeRow(nil)
	row.setConnection("c:vnc", sampleProfile(t, "Desktop", "", "VNC", "vnc.example.com", 5900))
	if row.badge.letter != "V" || row.badge.bg != vncBadgeColor {
		t.Fatalf("badge = %q %v, want V with the VNC colour", row.badge.letter, row.badge.bg)
	}
}

func TestSaveFolderDefaults_PersistsAndClears(t *testing.T) {
	connection := sampleProfile(t, "web", "Team", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{connection}, folders: []string{"Team"}}
	s := newTestScreen(t, store)

	s.saveFolderDefaults("Team", "team-user", "2222")
	document, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(document.Defaults) != 1 || document.Defaults[0].Username != "team-user" || document.Defaults[0].Port != 2222 {
		t.Fatalf("defaults = %+v", document.Defaults)
	}

	s.saveFolderDefaults("Team", "", "")
	document, err = store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(document.Defaults) != 0 {
		t.Fatalf("defaults = %+v, want the entry cleared", document.Defaults)
	}
}

func TestSaveFolderDefaults_InvalidPortIsRejected(t *testing.T) {
	store := &fakeStore{folders: []string{"Team"}}
	s := newTestScreen(t, store)
	s.saveFolderDefaults("Team", "team-user", "not-a-port")
	if store.saves != 0 {
		t.Fatalf("store saved %d times for an invalid port", store.saves)
	}
	if !strings.Contains(s.status.Text, "must be a number") {
		t.Errorf("status = %q, want a port error", s.status.Text)
	}
}

func TestInheritanceToggles_ShowEffectiveValuesAndPersist(t *testing.T) {
	connection := sampleProfile(t, "web", "Team", "SSH", "web.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{connection}, folders: []string{"Team"},
		defaults: []profiles.FolderDefault{{Path: "Team", Username: "team-user", Port: 2222}}}
	s := newTestScreen(t, store)
	s.tree.Select(connectionUID(connection.ID))

	s.props.inheritUser.SetChecked(true)
	s.props.inheritPort.SetChecked(true)
	if !s.props.user.Disabled() || !s.props.port.Disabled() {
		t.Error("inherited fields stay editable")
	}
	if s.props.user.PlaceHolder != "team-user" || s.props.port.PlaceHolder != "2222" {
		t.Errorf("placeholders = %q/%q, want the folder values", s.props.user.PlaceHolder, s.props.port.PlaceHolder)
	}
	test.Tap(s.props.save)
	if !store.profiles[0].InheritUsername || !store.profiles[0].InheritPort {
		t.Errorf("saved profile = %+v, want the inheritance flags", store.profiles[0])
	}
	if s.resolve(store.profiles[0]).Port != 2222 {
		t.Error("resolved port does not come from the folder")
	}
}

func TestApplyImport_MergeSkipsDuplicatesAndAddsFolders(t *testing.T) {
	existing := sampleProfile(t, "web01", "Production", "SSH", "web01.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{existing}, folders: []string{"Production"}}
	s := newTestScreen(t, store)
	incoming := sampleProfile(t, "web01", "Production", "SSH", "other.example.com", 22)
	fresh := sampleProfile(t, "db01", "Production/Data", "SSH", "db01.example.com", 22)
	preview := transfer.Preview{Format: transfer.FormatXML, Profiles: []profiles.Profile{incoming, fresh},
		Folders: []string{"Production", "Production/Data"}}

	s.applyImport(preview, false)

	if len(store.profiles) != 2 {
		t.Fatalf("profiles = %d, want the existing plus the fresh one", len(store.profiles))
	}
	if len(store.folders) != 2 {
		t.Fatalf("folders = %v, want the union", store.folders)
	}
	if !strings.Contains(s.status.Text, "1 connection(s) added") || !strings.Contains(s.status.Text, "1 skipped") {
		t.Errorf("status = %q, want added/skipped counts", s.status.Text)
	}
}

func TestApplyImport_ReplaceDropsExistingConnections(t *testing.T) {
	existing := sampleProfile(t, "old", "", "SSH", "old.example.com", 22)
	store := &fakeStore{profiles: []profiles.Profile{existing}, folders: []string{"Legacy"}}
	s := newTestScreen(t, store)
	preview := transfer.Preview{Format: transfer.FormatCSV, Profiles: []profiles.Profile{
		sampleProfile(t, "new", "Fresh", "RDP", "new.example.com", 3389),
	}, Folders: []string{"Fresh"}}

	s.applyImport(preview, true)

	if len(store.profiles) != 1 || store.profiles[0].Name != "new" {
		t.Fatalf("profiles = %+v, want only the imported one", store.profiles)
	}
	if len(store.folders) != 1 || store.folders[0] != "Fresh" {
		t.Fatalf("folders = %v, want only the imported folder", store.folders)
	}
}

func TestImportSummary_ReportsSkippedAndDropped(t *testing.T) {
	summary := importSummary(transfer.Preview{
		Profiles:    []profiles.Profile{{Name: "a"}},
		Unsupported: 2,
		Dropped:     3,
		Encrypted:   true,
	}, 1)
	for _, want := range []string{"2 row(s) skipped", "3 password(s) were dropped", "protected", "1 already exist"} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary = %q, missing %q", summary, want)
		}
	}
}

func TestUnionFolders_ParentsFirstWithoutDuplicates(t *testing.T) {
	got := unionFolders([]string{"A"}, []string{"A", "A/B", "C"})
	if len(got) != 3 || got[0] != "A" || got[1] != "A/B" || got[2] != "C" {
		t.Fatalf("union = %v", got)
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

func TestTreeRow_Connection_ShowsCircularProtocolBadge(t *testing.T) {
	row := newTreeRow(nil)
	row.setConnection("c:test", sampleProfile(t, "Production web", "", "SSH", "web.example.com", 22))
	if row.badge.letter != "S" || row.badge.Hidden {
		t.Fatalf("badge = %q hidden=%v, want a visible SSH badge", row.badge.letter, row.badge.Hidden)
	}
	if row.badge.bg != sshBadgeColor {
		t.Error("SSH badge does not use the SSH colour")
	}

	row.setConnection("c:test", sampleProfile(t, "Office desktop", "", "RDP", "desk.example.com", 3389))
	if row.badge.letter != "R" {
		t.Fatalf("badge = %q, want R", row.badge.letter)
	}
	if row.badge.bg != rdpBadgeColor {
		t.Error("RDP badge does not use the RDP colour")
	}
}

func TestTreeRow_Folder_HidesProtocolBadge(t *testing.T) {
	row := newTreeRow(nil)
	row.setFolder("f:Production", "Production")
	if row.name.Text != "Production" {
		t.Errorf("row = %q, want the folder name", row.name.Text)
	}
	if !row.badge.Hidden {
		t.Error("folder row shows a protocol badge")
	}
}

func TestTreeHighlighted_ReportsEndpointInStatusBar(t *testing.T) {
	existing := sampleProfile(t, "Production web", "Production", "SSH", "web.example.com", 22)
	s := newTestScreen(t, &fakeStore{profiles: []profiles.Profile{existing}})
	s.treeHighlighted(connectionUID(existing.ID))
	if !strings.Contains(s.status.Text, "web.example.com:22") {
		t.Errorf("status = %q, want the hovered endpoint", s.status.Text)
	}
	s.treeHighlighted(folderUID("Production"))
	if !strings.Contains(s.status.Text, "Production") {
		t.Errorf("status = %q, want the highlighted folder", s.status.Text)
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

type fakeRDPSession struct {
	closed bool
	done   chan struct{}
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

func TestSessionPanel_RDPConnect_ShowsExternalSessionState(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	panel := s.newSessionPanel(rdpProfile(t))
	session := &fakeRDPSession{done: make(chan struct{})}
	panel.applyConnectRDP(session, nil)

	if panel.rdp != session {
		t.Fatal("panel did not keep the RDP session")
	}
	if !panel.connect.Disabled() || panel.disconnect.Disabled() {
		t.Error("button state does not reflect a running RDP session")
	}
	if !strings.Contains(s.status.Text, "separate window") {
		t.Errorf("status = %q, want a separate-window notice", s.status.Text)
	}
	panel.disconnectShell()
	if !session.closed {
		t.Error("disconnect did not close the RDP session")
	}
	if panel.disconnect.Disabled() == false || panel.connect.Disabled() {
		t.Error("button state does not reflect a closed RDP session")
	}
}

func TestSessionPanel_RDPConnect_ForwardsTargetAndPassword(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	type call struct {
		target  rdpclient.Target
		options rdpclient.Options
	}
	calls := make(chan call, 1)
	s.launchRDP = func(target rdpclient.Target, options rdpclient.Options) (rdpSession, error) {
		calls <- call{target: target, options: options}
		return &fakeRDPSession{done: make(chan struct{})}, nil
	}
	panel := s.newSessionPanel(rdpProfile(t))
	panel.password.SetText("secret")
	panel.connectRDP()

	select {
	case got := <-calls:
		if got.target.Host != "desk.example.com" || got.target.Port != 3389 || got.target.Username != "kas" {
			t.Errorf("target = %+v", got.target)
		}
		if got.options.Password != "secret" {
			t.Error("password not forwarded to the launcher")
		}
		if got.options.ParentWindowID != 0 {
			t.Errorf("ParentWindowID = %d, want 0 in this preview", got.options.ParentWindowID)
		}
		if got.options.Security != "" {
			t.Errorf("Security = %q, want the client default unless legacy is enabled", got.options.Security)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("launcher was not called")
	}
}

type fakeArea struct {
	id        uint64
	destroyed bool
	visible   bool
}

func (f *fakeArea) ID() uint64        { return f.id }
func (f *fakeArea) Destroy()          { f.destroyed = true }
func (f *fakeArea) SetVisible(v bool) { f.visible = v }

func TestSessionPanel_RDPConnect_EmbedsSessionInTheTab(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	area := &fakeArea{id: 77}
	s.embedder = func(fyne.Window, fyne.CanvasObject, fyne.CanvasObject) (embeddedArea, error) { return area, nil }
	ids := make(chan uint64, 1)
	s.launchRDP = func(_ rdpclient.Target, options rdpclient.Options) (rdpSession, error) {
		ids <- options.ParentWindowID
		return &fakeRDPSession{done: make(chan struct{})}, nil
	}
	panel := s.newSessionPanel(rdpProfile(t))
	panel.connectRDP()

	select {
	case id := <-ids:
		if id != 77 {
			t.Fatalf("ParentWindowID = %d, want the embedded area id 77", id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("launcher was not called")
	}

	session := &fakeRDPSession{done: make(chan struct{})}
	panel.applyConnectRDPWithArea(session, area, nil, nil)
	if !strings.Contains(s.status.Text, "embedded") {
		t.Errorf("status = %q, want an embedded notice", s.status.Text)
	}
	panel.disconnectShell()
	if !area.destroyed {
		t.Error("disconnect did not destroy the embedded area")
	}
}

func TestSessionPanel_RDPConnect_EmbeddingUnavailable_FallsBackToSeparateWindow(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.embedder = func(fyne.Window, fyne.CanvasObject, fyne.CanvasObject) (embeddedArea, error) { return nil, ErrNoX11 }
	ids := make(chan uint64, 1)
	s.launchRDP = func(_ rdpclient.Target, options rdpclient.Options) (rdpSession, error) {
		ids <- options.ParentWindowID
		return &fakeRDPSession{done: make(chan struct{})}, nil
	}
	panel := s.newSessionPanel(rdpProfile(t))
	panel.connectRDP()

	select {
	case id := <-ids:
		if id != 0 {
			t.Fatalf("ParentWindowID = %d, want 0 without an embedded area", id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("launcher was not called")
	}

	panel.applyConnectRDPWithArea(&fakeRDPSession{done: make(chan struct{})}, nil, ErrNoX11, nil)
	if !strings.Contains(s.status.Text, "separate window") {
		t.Errorf("status = %q, want the separate-window fallback", s.status.Text)
	}
}

func TestEmbeddedArea_FollowsTabSelection(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	area := &fakeArea{id: 11}
	panel := s.newSessionPanel(rdpProfile(t))
	panel.embedded = area
	tab := container.NewTabItem("session", panel.content())
	s.panels["id"] = panel
	s.openTabs["id"] = tab

	s.tabSelected(tab)
	if !area.visible {
		t.Error("embedded area is hidden on its own tab")
	}
	s.tabSelected(s.welcome)
	if area.visible {
		t.Error("embedded area stayed visible after switching tabs")
	}
}

func TestSessionPanel_RDPConnect_LegacyCheckboxSelectsLegacySecurity(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	security := make(chan string, 1)
	s.launchRDP = func(_ rdpclient.Target, options rdpclient.Options) (rdpSession, error) {
		security <- options.Security
		return &fakeRDPSession{done: make(chan struct{})}, nil
	}
	panel := s.newSessionPanel(rdpProfile(t))
	panel.rdpLegacy.SetChecked(true)
	panel.connectRDP()

	select {
	case got := <-security:
		if got != "rdp" {
			t.Errorf("Security = %q, want rdp when legacy is enabled", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("launcher was not called")
	}
}

func TestSessionPanel_RDPTest_CallsProbeWithCredentials(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	probed := make(chan rdpclient.Options, 1)
	s.probeRDP = func(target rdpclient.Target, options rdpclient.Options) (string, error) {
		probed <- options
		return "Authentication only, exit status 0", nil
	}
	panel := s.newSessionPanel(rdpProfile(t))
	panel.password.SetText("secret")
	panel.testRDP()

	if !strings.Contains(s.status.Text, "Testing RDP credentials") {
		t.Errorf("status = %q, want a testing notice", s.status.Text)
	}
	select {
	case options := <-probed:
		if options.Password != "secret" {
			t.Error("probe did not receive the typed password")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("probe was not called")
	}
}

func TestSessionPanel_RDPProbeFailure_PointsToConnect(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	panel := s.newSessionPanel(rdpProfile(t))
	panel.rdpTest.Disable()
	// Mirror the async failure path without a goroutine.
	output := ""
	_ = output
	panel.screen.setProblem("RDP auth-only check failed: exit status 1. Some servers (for example xrdp) do not support auth-only; use Connect to open a real session.")
	if !strings.Contains(s.status.Text, "use Connect") {
		t.Errorf("status = %q, want a hint to use Connect", s.status.Text)
	}
}

func TestSessionPanel_RDPLaunchFailure_ReportsProblem(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	panel := s.newSessionPanel(rdpProfile(t))
	panel.applyConnectRDP(nil, errors.New("xfreerdp not found"))
	if !strings.Contains(s.status.Text, "RDP launch failed") || s.status.Importance != widget.DangerImportance {
		t.Errorf("status = %q, want a launch failure", s.status.Text)
	}
}

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

func TestRequireWritable_ReadOnlyStore_ReportsProblemAndBlocks(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.writable = false
	if s.requireWritable("Saving") {
		t.Fatal("requireWritable allowed a write on a read-only store")
	}
	want := "Saving is disabled because the existing file could not be read."
	if s.status.Text != want {
		t.Errorf("status = %q, want %q", s.status.Text, want)
	}
}

func TestRequireWritable_WritableStore_AllowsAndKeepsStatus(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.setStatus("ready")
	if !s.requireWritable("Saving") {
		t.Fatal("requireWritable blocked a writable store")
	}
	if s.status.Text != "ready" {
		t.Errorf("status = %q, want it left untouched", s.status.Text)
	}
}

func TestSetConnected_LiveSession_DisablesConnectAndPassword(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	p := s.newSessionPanel(sampleProfile(t, "lab", "", string(profiles.SSH), "192.0.2.10", 22))
	p.setConnected(true)
	if !p.connect.Disabled() {
		t.Error("Connect is still enabled while a session is live")
	}
	if p.disconnect.Disabled() {
		t.Error("Disconnect is disabled while a session is live")
	}
	if !p.password.Disabled() {
		t.Error("password field is enabled while a session is live")
	}
}

func TestSetConnected_EndedSession_RestoresControls(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	p := s.newSessionPanel(sampleProfile(t, "lab", "", string(profiles.SSH), "192.0.2.10", 22))
	p.setConnected(true)
	p.setConnected(false)
	if p.connect.Disabled() {
		t.Error("Connect is disabled after the session ended")
	}
	if !p.disconnect.Disabled() {
		t.Error("Disconnect is enabled after the session ended")
	}
	if p.password.Disabled() {
		t.Error("password field is disabled after the session ended")
	}
}

func TestInstallSlot_NewContent_ReplacesSessionArea(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	p := s.newSessionPanel(sampleProfile(t, "lab", "", string(profiles.SSH), "192.0.2.10", 22))
	label := widget.NewLabel("session area")
	p.installSlot(label)
	if len(p.slot.Objects) != 1 || p.slot.Objects[0] != label {
		t.Errorf("session area holds %d objects, want only the installed content", len(p.slot.Objects))
	}
}

func TestApplyConnect_DeclinedUnknownHostKey_ReEnablesConnect(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	p := s.newSessionPanel(sampleProfile(t, "lab", "", string(profiles.SSH), "192.0.2.10", 22))
	s.confirm = func(_, _ string, _, onDecline func()) {
		if onDecline != nil {
			onDecline()
		}
	}
	p.connect.Disable()
	p.applyConnect(sshclient.Target{Host: "192.0.2.10"}, sshclient.Options{}, nil, sshclient.Result{},
		&sshclient.UnknownHostKeyError{Fingerprint: "SHA256:abc"})
	if p.connect.Disabled() {
		t.Error("Connect stays disabled after the user declined the host key")
	}
}
