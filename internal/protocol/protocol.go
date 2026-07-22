// Package protocol defines the lifecycle contract every remote-access
// backend (SSH, VNC, RDP, ...) implements, and the factory that constructs
// one from a connection.ConnectionInfo. It mirrors the original
// Connection/Protocol/ProtocolBase.cs and ProtocolFactory.cs, adapted to
// Go idiom: events become callback setters instead of multicast delegates,
// and the factory becomes a self-registering constructor registry instead
// of a type switch, so each backend subpackage (internal/protocol/ssh,
// internal/protocol/vnc, ...) can wire itself in without this package
// importing any of them.
package protocol

import "context"

// Protocol is the lifecycle every backend implements: a single connection
// attempt, explicit teardown, focus for keyboard routing when the UI
// activates this session's tab, and resize for backends that support live
// geometry changes. Implementations are not required to be safe for
// concurrent use from multiple goroutines beyond what is documented here.
type Protocol interface {
	// Connect starts the session and returns once the underlying transport
	// is established, ctx is done, or the attempt fails outright (bad
	// credentials, host unreachable, protocol handshake error). Once
	// Connect has returned nil, further asynchronous failures (network
	// drop, remote-initiated close) are reported through OnError/OnClose,
	// not through a second return value.
	Connect(ctx context.Context) error

	// Disconnect tears down an active session and blocks until teardown
	// completes. It is idempotent: calling it on a session that is already
	// closed, or was never connected, is not an error. It triggers the
	// OnClose callback, if one is registered.
	Disconnect() error

	// Focus requests keyboard/input focus for the session's view. Called by
	// the UI when the user activates this session's tab.
	Focus()

	// Resize notifies the backend that its allotted view size changed, in
	// pixels. Backends that cannot resize a live session (most
	// external-process backends embedded by window reparenting) may treat
	// this as a no-op.
	Resize(width, height int)

	// OnError registers the callback invoked when the session fails
	// asynchronously after a successful Connect (network error, backend
	// process crash, protocol-level rejection). Registering a new callback
	// replaces the previous one; passing nil clears it. Only one callback
	// is held at a time — the caller (a session tab controller) is the
	// single owner of a Protocol instance, so multicast delegates as in the
	// original ProtocolBase are unnecessary here.
	OnError(func(error))

	// OnClose registers the callback invoked exactly once when the session
	// ends, however it ends: explicit Disconnect, remote hangup, or a
	// fatal error. Registering a new callback replaces the previous one;
	// passing nil clears it.
	OnClose(func())
}
