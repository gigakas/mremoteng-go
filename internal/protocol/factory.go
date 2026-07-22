package protocol

import (
	"fmt"
	"sync"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
)

// Constructor builds a Protocol for a connection. It receives the
// connection's effective (inheritance-resolved) values, already selected by
// Create — implementations do not need to call Effective() themselves.
type Constructor func(info *connection.ConnectionInfo, values connection.ConnectionValues) (Protocol, error)

var (
	registryMu sync.RWMutex
	registry   = map[connection.ProtocolType]Constructor{}
)

// Register associates a protocol type with its constructor. Backend
// subpackages call this from an init() func:
//
//	func init() {
//	    protocol.Register(connection.ProtocolSSH2, New)
//	}
//
// This package deliberately never imports backend subpackages — that would
// create an import cycle, since every backend imports this package for the
// Protocol interface. Wiring instead happens the other way around: the
// binary that wants a given protocol available blank-imports its package
// (see cmd/mremoteng), which runs that init() and populates the registry.
//
// Register panics on a nil constructor or a duplicate registration for the
// same type: both are programmer errors caught at init time, mirroring
// database/sql.Register and image.RegisterFormat.
func Register(t connection.ProtocolType, ctor Constructor) {
	if ctor == nil {
		panic(fmt.Sprintf("protocol: Register called with a nil constructor for %q", t))
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registry[t]; exists {
		panic(fmt.Sprintf("protocol: Register called twice for %q", t))
	}
	registry[t] = ctor
}

// Create builds the Protocol for info's configured protocol type, mirroring
// ProtocolFactory.CreateProtocol. It fails if no backend registered itself
// for that type — e.g. the binary was built without that protocol, or the
// protocol is not implemented yet.
func Create(info *connection.ConnectionInfo) (Protocol, error) {
	if info == nil {
		return nil, fmt.Errorf("protocol: connection info is nil")
	}

	values := info.Effective()

	registryMu.RLock()
	ctor, ok := registry[values.Protocol]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("protocol: no backend registered for %q", values.Protocol)
	}
	return ctor(info, values)
}
