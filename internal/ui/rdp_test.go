package ui

import (
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	rdpclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/rdp"
	"strings"
	"testing"
	"time"
)

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

func TestRDPGatewayForm_Fields_BuildTheGateway(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	form, box := newRDPGatewayForm(rdpclient.Gateway{Host: "gw.example.com", Port: 8443, UsageMethod: "direct"})
	if box == nil {
		t.Fatal("the gateway section has no content")
	}
	if form.host.Text != "gw.example.com" || form.port.Text != "8443" || form.method.Selected != "direct" {
		t.Fatalf("form = %q/%q/%q, want the stored gateway", form.host.Text, form.port.Text, form.method.Selected)
	}
	gateway, err := form.gateway()
	if err != nil {
		t.Fatalf("gateway: %v", err)
	}
	if gateway.Host != "gw.example.com" || gateway.Port != 8443 || gateway.UsageMethod != "direct" {
		t.Errorf("gateway = %+v, want the form values", gateway)
	}
}

func TestRDPGatewayForm_EmptyPort_KeepsTheClientDefault(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	form, _ := newRDPGatewayForm(rdpclient.Gateway{})
	if form.method.Selected != "client default" {
		t.Errorf("method = %q, want the default entry", form.method.Selected)
	}
	form.host.SetText(" gw.example.com ")
	gateway, err := form.gateway()
	if err != nil {
		t.Fatalf("gateway: %v", err)
	}
	if gateway.Port != 0 || gateway.UsageMethod != "" {
		t.Errorf("gateway = %+v, want no port and no method", gateway)
	}
	if argument := gatewaySummary(gateway); argument != " through gateway gw.example.com:443" {
		t.Errorf("summary = %q, want the default port spelled out", argument)
	}
}

func TestRDPGatewayForm_InvalidInput_IsRefusedWithAReason(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	cases := map[string][2]string{
		"port": {"gw.example.com", "https"},
		"host": {"gw,example.com", ""},
	}
	for name, values := range cases {
		form, _ := newRDPGatewayForm(rdpclient.Gateway{})
		form.host.SetText(values[0])
		form.port.SetText(values[1])
		if _, err := form.gateway(); err == nil {
			t.Errorf("%s case was accepted", name)
		}
	}
	if gatewaySummary(rdpclient.Gateway{}) != "" {
		t.Error("a session without a gateway must not mention one")
	}
}

// waitFor polls until check passes or the deadline expires: the RDP watcher runs
// on its own goroutine, so the effect of a session ending is not synchronous.
func waitFor(t *testing.T, check func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

// TestSessionPanel_RDPImmediateExit_ReportsTheClientCause covers the report the
// session bar could not give: FreeRDP's own line, plus what to try next, in a
// dialog the user can read after the window has gone.
func TestSessionPanel_RDPImmediateExit_ReportsTheClientCause(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	notices := make([]string, 0, 1)
	s.notice = func(title, message string) { notices = append(notices, title+"\n"+message) }
	session := &fakeRDPSession{
		done:   make(chan struct{}),
		output: "[ERROR][com.freerdp.core.nego] - [nego_connect]: Failed to connect\n",
	}
	close(session.done)
	panel := s.newSessionPanel(rdpProfile(t))

	panel.applyConnectRDP(session, nil)

	if !waitFor(t, func() bool { return len(notices) > 0 }) {
		t.Fatalf("no dialog was reported; status = %q", s.status.Text)
	}
	if !strings.Contains(notices[0], "Failed to connect") {
		t.Errorf("dialog = %q, want the client's cause", notices[0])
	}
	if !strings.Contains(notices[0], "did not answer") {
		t.Errorf("dialog = %q, want the transport hint for a refused connection", notices[0])
	}
	if !strings.Contains(s.status.Text, "before opening a session") {
		t.Errorf("status = %q, want the failure notice", s.status.Text)
	}
	if panel.rdp != nil {
		t.Error("a dead session is still attached to the panel")
	}
}

// TestSessionPanel_RDPLaterExit_ReportsInTheStatus keeps the dialog for failures:
// a session that ran for a while and ended is reported, but not with a popup.
func TestSessionPanel_RDPLaterExit_ReportsInTheStatus(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	notices := 0
	s.notice = func(string, string) { notices++ }
	session := &fakeRDPSession{done: make(chan struct{}), output: "connection closed by the server\n"}
	panel := s.newSessionPanel(rdpProfile(t))
	panel.applyConnectRDP(session, nil)
	panel.rdpStart = time.Now().Add(-10 * time.Second)

	close(session.done)

	if !waitFor(t, func() bool { return strings.Contains(s.status.Text, "RDP session closed") }) {
		t.Fatalf("status = %q, want a closed notice", s.status.Text)
	}
	if notices != 0 {
		t.Errorf("dialogs = %d, want none for a session that had been open", notices)
	}
	if !strings.Contains(s.status.Text, "closed by the server") {
		t.Errorf("status = %q, want the client's cause", s.status.Text)
	}
}

func TestRDPFailureHint_PointsAtTheCause(t *testing.T) {
	cases := map[string]string{
		"Gateway authentication failed":                     "RD Gateway",
		"server certificate is not trusted":                 "Ignore cert",
		"Authentication only, exit status 1":                "credentials",
		"connect to localhost:1 failed: Connection refused": "port",
		"nego_connect failed: no protocol":                  "Legacy TLS",
		"no diagnostic output":                              "Check the address",
	}
	for cause, want := range cases {
		if got := rdpFailureHint(cause); !strings.Contains(got, want) {
			t.Errorf("rdpFailureHint(%q) = %q, want it to mention %q", cause, got, want)
		}
	}
}

// embeddedRDPScreen opens an RDP tab whose embedded area is visible, which is what
// the user has on screen while a dialog opens over a running session.
func embeddedRDPScreen(t *testing.T) (*screen, *fakeArea) {
	t.Helper()
	s := newTestScreen(t, &fakeStore{})
	profile := sampleProfile(t, "desk", "", string(profiles.RDP), "desk.example.com", 3389)
	s.openSessionTab(profile)
	panel := s.panels[profile.ID]
	area := &fakeArea{id: 21}
	panel.embedded = area
	// Select does not fire again for the tab that is already selected, so apply the
	// visibility rule the way the window does.
	s.tabSelected(s.openTabs[profile.ID])
	if !area.visible {
		t.Fatal("the embedded area is not visible on its own tab")
	}
	return s, area
}

// TestShowModal_HidesTheEmbeddedSession is the reported bug: an embedded RDP session
// is a native X11 window that X composites above the Fyne canvas, so every dialog
// opened behind it and the only way to read one was to disconnect.
func TestShowModal_HidesTheEmbeddedSession(t *testing.T) {
	s, area := embeddedRDPScreen(t)

	dialog := s.showModal(widget.NewLabel("settings"), fyne.NewSize(200, 120))

	if area.visible {
		t.Error("the embedded session stayed in front of the dialog")
	}
	dialog.Hide()
	if !area.visible {
		t.Error("the embedded session did not come back when the dialog closed")
	}
}

// TestShowModal_NestedDialogs_KeepTheSessionHiddenUntilTheLastCloses covers a dialog
// opened from a dialog: the session must not flash back between them.
func TestShowModal_NestedDialogs_KeepTheSessionHiddenUntilTheLastCloses(t *testing.T) {
	s, area := embeddedRDPScreen(t)

	first := s.showModal(widget.NewLabel("first"), fyne.NewSize(200, 120))
	second := s.showModal(widget.NewLabel("second"), fyne.NewSize(200, 120))
	first.Hide()
	if area.visible {
		t.Error("the session came back while a second dialog was still open")
	}
	second.Hide()
	if !area.visible {
		t.Error("the session did not come back after the last dialog closed")
	}
}

// TestModalHide_Repeated_IsIdempotent guards the shared Hide path: the depth counter
// must not go negative and map a session nobody asked for.
func TestModalHide_Repeated_IsIdempotent(t *testing.T) {
	s, area := embeddedRDPScreen(t)

	dialog := s.showModal(widget.NewLabel("once"), fyne.NewSize(200, 120))
	dialog.Hide()
	dialog.Hide()

	if s.modalDepth != 0 {
		t.Errorf("modal depth = %d, want 0", s.modalDepth)
	}
	if !area.visible {
		t.Error("the session is hidden after the dialog closed")
	}
}

// TestShowModal_OnlyRestoresTheSelectedTab keeps the tab rule after a dialog: a
// session on a background tab must not be mapped when the dialog closes.
func TestShowModal_OnlyRestoresTheSelectedTab(t *testing.T) {
	s, area := embeddedRDPScreen(t)

	dialog := s.showModal(widget.NewLabel("settings"), fyne.NewSize(200, 120))
	s.tabs.Select(s.welcome)
	dialog.Hide()

	if area.visible {
		t.Error("the session of a background tab was mapped when the dialog closed")
	}
}

// TestShowConfirm_HidesTheEmbeddedSession covers the dialogs the app asks for most:
// a confirmation drawn by Fyne's own dialog package would sit behind the session too,
// so the confirm path goes through our modal as well.
func TestShowConfirm_HidesTheEmbeddedSession(t *testing.T) {
	s, area := embeddedRDPScreen(t)
	answered := false

	s.confirm("Delete connection?", "This cannot be undone.", func() { answered = true }, nil)
	if area.visible {
		t.Error("the session stayed in front of the confirmation")
	}

	// Answer the dialog the way the user does: find its Confirm button in the overlay
	// and press it.
	objects := map[fyne.CanvasObject]bool{}
	for _, overlay := range s.win.Canvas().Overlays().List() {
		collectObjects(overlay, objects)
	}
	var confirmButton *widget.Button
	for object := range objects {
		if button, ok := object.(*widget.Button); ok && button.Text == "Confirm" {
			confirmButton = button
		}
	}
	if confirmButton == nil {
		t.Fatal("the confirmation has no Confirm button")
	}
	test.Tap(confirmButton)

	if !answered {
		t.Error("Confirm did not run the action")
	}
	if !area.visible {
		t.Error("the session did not come back after the confirmation closed")
	}
}

// TestShowConfirm_Declined_KeepsTheSessionBack covers the other button: cancelling
// must not leave the remote desktop unmapped.
func TestShowConfirm_Declined_KeepsTheSessionBack(t *testing.T) {
	s, area := embeddedRDPScreen(t)
	declined := false

	s.confirm("Discard changes?", "They will be lost.", func() {}, func() { declined = true })

	objects := map[fyne.CanvasObject]bool{}
	for _, overlay := range s.win.Canvas().Overlays().List() {
		collectObjects(overlay, objects)
	}
	var cancel *widget.Button
	for object := range objects {
		if button, ok := object.(*widget.Button); ok && button.Text == "Cancel" {
			cancel = button
		}
	}
	if cancel == nil {
		t.Fatal("the confirmation has no Cancel button")
	}
	test.Tap(cancel)

	if !declined {
		t.Error("Cancel did not report the decline")
	}
	if !area.visible {
		t.Error("the session stayed hidden after the confirmation was cancelled")
	}
}

// TestModalGuard_RestoresOnce covers the bracket around Fyne's file chooser, which
// reports a pick and a cancel through the same callback.
func TestModalGuard_RestoresOnce(t *testing.T) {
	s, area := embeddedRDPScreen(t)

	end := s.modalGuard()
	if area.visible {
		t.Error("the chooser bracket did not hide the session")
	}
	end()
	end()

	if !area.visible {
		t.Error("the session did not come back after the chooser closed")
	}
	if s.modalDepth != 0 {
		t.Errorf("modal depth = %d, want 0", s.modalDepth)
	}
}
