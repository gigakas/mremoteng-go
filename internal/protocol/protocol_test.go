package protocol_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
)

// fakeProtocol is a minimal Protocol implementation used to exercise the
// interface contract and the factory registry without depending on any real
// backend (none exist yet — this is stage 2.1, backends land in 2.2-2.7).
type fakeProtocol struct {
	values                   connection.ConnectionValues
	connectErr               error
	connectCalls             int
	disconnectCalls          int
	focusCalls               int
	lastResizeW, lastResizeH int
	onError                  func(error)
	onClose                  func()
}

func newFakeProtocol(_ *connection.ConnectionInfo, values connection.ConnectionValues) (protocol.Protocol, error) {
	return &fakeProtocol{values: values}, nil
}

func (f *fakeProtocol) Connect(ctx context.Context) error {
	f.connectCalls++
	return f.connectErr
}

func (f *fakeProtocol) Disconnect() error {
	f.disconnectCalls++
	if f.onClose != nil {
		f.onClose()
	}
	return nil
}

func (f *fakeProtocol) Focus() { f.focusCalls++ }

func (f *fakeProtocol) Resize(width, height int) {
	f.lastResizeW, f.lastResizeH = width, height
}

func (f *fakeProtocol) OnError(cb func(error)) { f.onError = cb }
func (f *fakeProtocol) OnClose(cb func())      { f.onClose = cb }

// fireError simulates an asynchronous failure, as a real backend would after
// a successful Connect (e.g. the network connection dropped).
func (f *fakeProtocol) fireError(err error) {
	if f.onError != nil {
		f.onError(err)
	}
}

const fakeProtocolType connection.ProtocolType = "test-fake-protocol"

func newTestConnectionInfo(t *testing.T, proto connection.ProtocolType) *connection.ConnectionInfo {
	t.Helper()
	info, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	info.Raw.Protocol = proto
	info.Raw.Hostname = "example.invalid"
	return info
}

func TestRegister_DuplicateType_Panics(t *testing.T) {
	const dup connection.ProtocolType = "test-fake-protocol-duplicate"
	protocol.Register(dup, newFakeProtocol)

	defer func() {
		if recover() == nil {
			t.Fatal("expected Register to panic on a duplicate protocol type")
		}
	}()
	protocol.Register(dup, newFakeProtocol)
}

func TestRegister_NilConstructor_Panics(t *testing.T) {
	const t2 connection.ProtocolType = "test-fake-protocol-nil-ctor"

	defer func() {
		if recover() == nil {
			t.Fatal("expected Register to panic on a nil constructor")
		}
	}()
	protocol.Register(t2, nil)
}

func TestCreate_NilConnectionInfo_ReturnsError(t *testing.T) {
	if _, err := protocol.Create(nil); err == nil {
		t.Fatal("expected an error for a nil connection info")
	}
}

func TestCreate_UnregisteredProtocol_ReturnsError(t *testing.T) {
	const unregistered connection.ProtocolType = "test-fake-protocol-unregistered"
	info := newTestConnectionInfo(t, unregistered)

	if _, err := protocol.Create(info); err == nil {
		t.Fatal("expected an error for an unregistered protocol type")
	}
}

func TestCreate_RegisteredProtocol_DispatchesEffectiveValues(t *testing.T) {
	protocol.Register(fakeProtocolType, newFakeProtocol)
	info := newTestConnectionInfo(t, fakeProtocolType)

	p, err := protocol.Create(info)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	fake, ok := p.(*fakeProtocol)
	if !ok {
		t.Fatalf("Create returned %T, want *fakeProtocol", p)
	}
	if fake.values.Hostname != "example.invalid" {
		t.Errorf("hostname = %q, want %q", fake.values.Hostname, "example.invalid")
	}
}

// TestProtocol_Lifecycle exercises the full interface contract through the
// fake: Connect, Focus, Resize, the OnClose callback firing on Disconnect,
// and the OnError callback firing on an asynchronous failure. This is the
// behavior every real backend (SSH, VNC, RDP, ...) must reproduce.
func TestProtocol_Lifecycle(t *testing.T) {
	var p protocol.Protocol = &fakeProtocol{}
	fake := p.(*fakeProtocol)

	closed := false
	p.OnClose(func() { closed = true })

	var gotErr error
	p.OnError(func(err error) { gotErr = err })

	if err := p.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if fake.connectCalls != 1 {
		t.Errorf("connectCalls = %d, want 1", fake.connectCalls)
	}

	p.Focus()
	if fake.focusCalls != 1 {
		t.Errorf("focusCalls = %d, want 1", fake.focusCalls)
	}

	p.Resize(800, 600)
	if fake.lastResizeW != 800 || fake.lastResizeH != 600 {
		t.Errorf("resize = (%d, %d), want (800, 600)", fake.lastResizeW, fake.lastResizeH)
	}

	simulated := errors.New("simulated backend failure")
	fake.fireError(simulated)
	if !errors.Is(gotErr, simulated) {
		t.Errorf("OnError callback got %v, want %v", gotErr, simulated)
	}

	if err := p.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if fake.disconnectCalls != 1 {
		t.Errorf("disconnectCalls = %d, want 1", fake.disconnectCalls)
	}
	if !closed {
		t.Error("OnClose callback was not invoked by Disconnect")
	}
}

func TestProtocol_Disconnect_IsIdempotent(t *testing.T) {
	var p protocol.Protocol = &fakeProtocol{}

	if err := p.Disconnect(); err != nil {
		t.Fatalf("first Disconnect: %v", err)
	}
	if err := p.Disconnect(); err != nil {
		t.Fatalf("second Disconnect: %v", err)
	}
}
