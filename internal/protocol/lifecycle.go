package protocol

import (
	"errors"
	"io"
	"sync"
)

// Lifecycle implements the OnError/OnClose bookkeeping every Protocol
// implementation needs: single-callback storage guarded by a mutex, and a
// close-once guard so OnClose fires exactly once no matter how many times
// the session is torn down (Disconnect must be idempotent). Backends embed
// it instead of re-implementing this bookkeeping in every subpackage.
type Lifecycle struct {
	mu      sync.Mutex
	onError func(error)
	onClose func()
	closed  bool
}

// OnError implements Protocol.
func (l *Lifecycle) OnError(cb func(error)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onError = cb
}

// OnClose implements Protocol.
func (l *Lifecycle) OnClose(cb func()) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onClose = cb
}

// FireError invokes the registered error callback, if any. Safe to call
// from any goroutine.
func (l *Lifecycle) FireError(err error) {
	l.mu.Lock()
	cb := l.onError
	l.mu.Unlock()
	if cb != nil {
		cb(err)
	}
}

// FireClose marks the session closed and invokes the registered close
// callback the first time it is called; later calls are no-ops. It reports
// whether this call was the one that fired it, which callers generally
// don't need to check.
func (l *Lifecycle) FireClose() bool {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return false
	}
	l.closed = true
	cb := l.onClose
	l.mu.Unlock()
	if cb != nil {
		cb()
	}
	return true
}

// WatchedStream wraps an underlying byte stream so that any Read/Write
// error automatically drives the embedded Lifecycle: a plain io.EOF closes
// the session (a clean remote hangup, not a failure worth surfacing via
// OnError), any other error fires OnError and then closes it. This is what
// lets OnClose fire "on remote hangup" without a dedicated watcher
// goroutine per session — whichever goroutine is actively reading or
// writing when the stream dies is the one that observes it and reports it,
// which for a terminal session is always the consumer's read loop.
//
// Close closes the underlying stream, exactly once regardless of how many
// times it is called — most io.Closer implementations (e.g. net.Conn)
// return an error on a second Close, which would otherwise make every
// backend's Disconnect non-idempotent. Close deliberately does not itself
// fire OnClose — callers (a backend's Disconnect) call FireClose
// explicitly so OnClose fires even if nothing was blocked in Read at the
// time.
type WatchedStream struct {
	*Lifecycle
	io.ReadWriteCloser

	closeOnce sync.Once
	closeErr  error
}

// NewWatchedStream wraps rwc so its I/O errors drive lc.
func NewWatchedStream(lc *Lifecycle, rwc io.ReadWriteCloser) *WatchedStream {
	return &WatchedStream{Lifecycle: lc, ReadWriteCloser: rwc}
}

func (w *WatchedStream) Close() error {
	w.closeOnce.Do(func() {
		w.closeErr = w.ReadWriteCloser.Close()
	})
	return w.closeErr
}

func (w *WatchedStream) Read(p []byte) (int, error) {
	n, err := w.ReadWriteCloser.Read(p)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			w.FireError(err)
		}
		w.FireClose()
	}
	return n, err
}

func (w *WatchedStream) Write(p []byte) (int, error) {
	n, err := w.ReadWriteCloser.Write(p)
	if err != nil {
		w.FireError(err)
		w.FireClose()
	}
	return n, err
}
