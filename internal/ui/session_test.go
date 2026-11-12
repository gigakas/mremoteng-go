package ui

import (
	"fyne.io/fyne/v2/widget"
	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	sshclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/ssh"
	telnetclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/telnet"
	"io"
	"strings"
	"testing"
)

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
