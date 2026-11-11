package ui

import (
	"image"
	"net"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	vncclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/vnc"
)

// vncSession is the surface the panel needs from a VNC client.
type vncSession interface {
	Size() (int, int)
	Name() string
	Image() *image.RGBA
	ReadMessage() error
	Pointer(x, y int, mask byte) error
	Close() error
}

// vncDialer connects to a VNC server; tests substitute it.
type vncDialer func(address string, config vncclient.Config) (vncSession, error)

func defaultVNCDialer(address string, config vncclient.Config) (vncSession, error) {
	client, err := vncclient.Dial(address, config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// vncView renders the remote framebuffer and forwards pointer clicks.
type vncView struct {
	widget.BaseWidget

	image  *canvas.Image
	source vncSession
	pixels *image.RGBA
}

func newVNCView(session vncSession) *vncView {
	view := &vncView{source: session, pixels: session.Image()}
	view.image = canvas.NewImageFromImage(view.pixels)
	view.image.FillMode = canvas.ImageFillContain
	view.image.ScaleMode = canvas.ImageScaleFastest
	view.ExtendBaseWidget(view)
	return view
}

func (v *vncView) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(v.image)
}

// refresh repaints after the client applied an update.
func (v *vncView) refresh() {
	v.image.Refresh()
}

// Tapped sends a left click at the mapped remote position.
func (v *vncView) Tapped(event *fyne.PointEvent) {
	width, height := v.source.Size()
	if width == 0 || height == 0 || v.Size().Width == 0 || v.Size().Height == 0 {
		return
	}
	x := int(event.Position.X / v.Size().Width * float32(width))
	y := int(event.Position.Y / v.Size().Height * float32(height))
	_ = v.source.Pointer(x, y, 1)
	_ = v.source.Pointer(x, y, 0)
}

// MouseMoved keeps the remote pointer in sync without pressing a button.
func (v *vncView) MouseMoved(event *desktop.MouseEvent) {
	width, height := v.source.Size()
	if width == 0 || height == 0 {
		return
	}
	x := int(event.Position.X / v.Size().Width * float32(width))
	y := int(event.Position.Y / v.Size().Height * float32(height))
	_ = v.source.Pointer(x, y, 0)
}

// startVNC installs the view and starts reading framebuffer updates.
func (p *sessionPanel) startVNC(session vncSession) {
	p.vnc = session
	p.vncView = newVNCView(session)
	p.installSlot(container.NewStack(p.vncView))
	p.setConnected(true)
	p.setStatus("VNC session with " + p.profile.Host + " connected. Click the view to send a left click.")
	go p.readVNC(session)
}

func (p *sessionPanel) readVNC(session vncSession) {
	for {
		if err := session.ReadMessage(); err != nil {
			fyne.Do(func() { p.vncClosed(session) })
			return
		}
		fyne.Do(func() {
			if p.vncView != nil {
				p.vncView.refresh()
			}
		})
	}
}

func (p *sessionPanel) vncClosed(session vncSession) {
	if p.vnc != session {
		return
	}
	p.vnc = nil
	p.vncView = nil
	p.setConnected(false)
	p.setStatus("VNC session closed.")
}

// connectVNC opens the VNC session for this tab.
func (p *sessionPanel) connectVNC() {
	p.setConnecting("Connecting to " + p.profile.Host + " (VNC) ...")
	resolved := p.screen.resolve(p.profile)
	address := net.JoinHostPort(resolved.Host, strconv.Itoa(resolved.Port))
	go func() {
		session, err := p.screen.dialVNC(address, vncclient.Config{Password: p.password.Text, Timeout: 10 * time.Second})
		fyne.Do(func() {
			if err != nil {
				p.setConnected(false)
				p.setProblem("VNC connection failed: " + err.Error())
				return
			}
			p.startVNC(session)
		})
	}()
}

// vncProtocol reports whether this panel is a VNC session.
func (p *sessionPanel) vncProtocol() bool { return p.profile.Protocol == profiles.VNC }
