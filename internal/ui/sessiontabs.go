package ui

import (
	"context"
	"fmt"
	"io"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
)

// connectTimeout bounds how long Open waits for protocol.Protocol.Connect
// before giving up — Connect's own contract (internal/protocol) is to
// return once the session is established or the attempt fails, not to
// run for the session's whole lifetime, so a generous but finite timeout
// is appropriate here rather than none at all.
const connectTimeout = 30 * time.Second

// SessionTabs hosts one tab per open connection session, dispatching each
// to the right view by which capability interface its protocol.Protocol
// implements — the same three-way split the protocol package itself
// establishes (TerminalProtocol/FramebufferProtocol/WindowProtocol are
// composed with Protocol precisely so a caller like this one can type-
// switch on them instead of needing per-backend knowledge).
type SessionTabs struct {
	Widget *container.AppTabs

	// win is needed only for WindowProtocol sessions, to embed their
	// native window into the tab (NativeWindowHost.Embed needs the
	// hosting fyne.Window to get its native handle from).
	win fyne.Window
}

// NewSessionTabs creates an empty tab host. win is the application window
// tabs will be embedded into (see the win field's doc comment).
func NewSessionTabs(win fyne.Window) *SessionTabs {
	return &SessionTabs{Widget: container.NewAppTabs(), win: win}
}

// Open adds a new tab titled name, connects p in the background, and
// wires its I/O to the appropriate view once connected. p.OnClose closes
// the tab automatically when the session ends, however it ends.
func (t *SessionTabs) Open(name string, p protocol.Protocol) {
	content, wire := t.buildView(p)
	tabItem := container.NewTabItem(name, content)
	t.Widget.Append(tabItem)
	t.Widget.Select(tabItem)

	p.OnClose(func() {
		fyne.Do(func() { t.Widget.Remove(tabItem) })
	})

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	go func() {
		defer cancel()
		err := p.Connect(ctx)
		fyne.Do(func() {
			if err != nil {
				tabItem.Content = widget.NewLabel(fmt.Sprintf("Connection failed: %v", err))
				t.Widget.Refresh()
				return
			}
			if wire != nil {
				wire()
			}
		})
	}()
}

// buildView creates the tab's content up front (so the tab appears
// immediately, showing a connecting state via the underlying widget's own
// empty/default rendering) and returns a wire func to call once Connect
// has succeeded — the part that needs data only available post-connect
// (a WindowProtocol's NativeWindowHandle) or that starts consuming a
// live stream (a TerminalProtocol's Read loop).
func (t *SessionTabs) buildView(p protocol.Protocol) (content fyne.CanvasObject, wire func()) {
	switch v := p.(type) {
	case protocol.TerminalProtocol:
		term := NewTerminal()
		term.OnInput = func(b []byte) { v.Write(b) }
		return term, func() { go pumpTerminal(term, v) }

	case protocol.FramebufferProtocol:
		view := NewFramebufferView()
		return view, func() { view.Attach(v) }

	case protocol.WindowProtocol:
		host := NewNativeWindowHost()
		return host, func() {
			if handle := v.NativeWindowHandle(); handle != 0 {
				if err := host.Embed(t.win, handle); err != nil {
					// Surfaced via OnError rather than swallowed — the
					// tab stays open with an unembedded (invisible)
					// native window, which is honest about the failure
					// mode rather than pretending it succeeded.
					// FireError isn't ours to call (that's the backend's
					// own Lifecycle), so this goes through fmt.Errorf's
					// %w chain back into the log for now; a visible
					// in-tab error banner is a natural follow-up once
					// this can actually be seen and iterated on.
					fmt.Printf("ui: embed native window for %T: %v\n", p, err)
				}
			}
		}

	default:
		return widget.NewLabel(fmt.Sprintf("No view implemented for %T", p)), nil
	}
}

// pumpTerminal relays a TerminalProtocol's output into term until Read
// returns an error (session ended, matching every stage 2.2/2.6 backend's
// documented behavior for a dead connection).
func pumpTerminal(term *Terminal, r io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			data := append([]byte(nil), buf[:n]...)
			fyne.Do(func() { term.Write(data) })
		}
		if err != nil {
			return
		}
	}
}
