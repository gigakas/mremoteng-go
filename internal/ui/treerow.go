package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

// Protocol badge colours. They are intentionally independent from the theme so
// SSH and RDP stay distinguishable in light and dark mode.
var (
	sshBadgeColor  = color.NRGBA{R: 0x2d, G: 0x7f, B: 0xf9, A: 0xff}
	rdpBadgeColor  = color.NRGBA{R: 0xd9, G: 0x7a, B: 0x00, A: 0xff}
	vncBadgeColor  = color.NRGBA{R: 0x2e, G: 0xa0, B: 0x7a, A: 0xff}
	termBadgeColor = color.NRGBA{R: 0x6c, G: 0x7b, B: 0x93, A: 0xff}
	webBadgeColor  = color.NRGBA{R: 0x9b, G: 0x6b, B: 0xd9, A: 0xff}
)

// badgeDiameter is the circular protocol icon size; it fits inside a tree row
// without shrinking the connection name more than necessary.
const badgeDiameter = 18

// protocolBadge is the circular protocol icon shown at the start of a
// connection row.
type protocolBadge struct {
	widget.BaseWidget

	letter string
	bg     color.Color
}

func newProtocolBadge() *protocolBadge {
	badge := &protocolBadge{letter: "S", bg: sshBadgeColor}
	badge.ExtendBaseWidget(badge)
	return badge
}

// Set adapts the badge to the connection protocol.
func (b *protocolBadge) Set(protocol profiles.Protocol) {
	switch protocol {
	case profiles.RDP:
		b.letter, b.bg = "R", rdpBadgeColor
	case profiles.VNC:
		b.letter, b.bg = "V", vncBadgeColor
	case profiles.TELNET, profiles.RAW, profiles.RLOGIN:
		b.letter, b.bg = "T", termBadgeColor
	case profiles.HTTP:
		b.letter, b.bg = "W", webBadgeColor
	default:
		b.letter, b.bg = "S", sshBadgeColor
	}
	b.Refresh()
}

func (b *protocolBadge) CreateRenderer() fyne.WidgetRenderer {
	circle := canvas.NewCircle(b.bg)
	letter := canvas.NewText(b.letter, color.White)
	letter.TextSize = 10
	letter.TextStyle = fyne.TextStyle{Bold: true}
	letter.Alignment = fyne.TextAlignCenter
	return &badgeRenderer{badge: b, circle: circle, letter: letter}
}

type badgeRenderer struct {
	badge  *protocolBadge
	circle *canvas.Circle
	letter *canvas.Text
}

func (r *badgeRenderer) Layout(size fyne.Size) {
	r.circle.Resize(size)
	textSize := r.letter.MinSize()
	r.letter.Resize(fyne.NewSize(size.Width, textSize.Height))
	r.letter.Move(fyne.NewPos(0, (size.Height-textSize.Height)/2))
}

func (r *badgeRenderer) MinSize() fyne.Size {
	return fyne.NewSize(badgeDiameter, badgeDiameter)
}

func (r *badgeRenderer) Refresh() {
	r.circle.FillColor = r.badge.bg
	r.circle.Refresh()
	r.letter.Text = r.badge.letter
	r.letter.Refresh()
}

func (r *badgeRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.circle, r.letter}
}

func (r *badgeRenderer) Destroy() {}

// treeRow is one connection-tree row: a circular protocol badge plus a
// truncated name. The full name and endpoint are reported in the status bar
// while the tree reports a highlight, because Fyne's tree consumes row hover.
type treeRow struct {
	widget.BaseWidget

	screen *screen
	uid    string
	kind   treeRowKind
	name   *widget.Label
	badge  *protocolBadge
	drag   fyne.Delta
}

type treeRowKind int

const (
	rowFolder treeRowKind = iota
	rowConnection
)

func newTreeRow(s *screen) *treeRow {
	row := &treeRow{screen: s, name: widget.NewLabel("")}
	row.name.Truncation = fyne.TextTruncateEllipsis
	row.badge = newProtocolBadge()
	row.ExtendBaseWidget(row)
	return row
}

func (r *treeRow) CreateRenderer() fyne.WidgetRenderer {
	// Badge first, name in the remaining space so it truncates instead of
	// pushing the badge out.
	return widget.NewSimpleRenderer(container.NewBorder(nil, nil, r.badge, nil, r.name))
}

func (r *treeRow) setFolder(uid, name string) {
	r.uid, r.kind = uid, rowFolder
	r.name.TextStyle = fyne.TextStyle{Bold: true}
	r.name.SetText(name)
	r.badge.Hide()
	if r.screen != nil {
		r.screen.rows[uid] = r
	}
	r.Refresh()
}

func (r *treeRow) setConnection(uid string, profile profiles.Profile) {
	r.uid, r.kind = uid, rowConnection
	r.name.TextStyle = fyne.TextStyle{Bold: false}
	r.name.SetText(profile.Name)
	r.badge.Set(profile.Protocol)
	r.badge.Show()
	if r.screen != nil {
		r.screen.rows[uid] = r
	}
	r.Refresh()
}

// Tapped opens the row. The tree's own node may not receive taps once the row
// is draggable, so the row handles activation itself.
func (r *treeRow) Tapped(*fyne.PointEvent) {
	if r.screen == nil || r.uid == "" {
		return
	}
	r.screen.rowActivated(r.uid)
}

// TappedSecondary opens the row context menu.
func (r *treeRow) TappedSecondary(event *fyne.PointEvent) {
	if r.screen == nil || r.uid == "" {
		return
	}
	r.screen.showRowMenu(r.uid, event.AbsolutePosition)
}

// Dragged accumulates the pointer delta so DragEnd can resolve the drop point
// even when the driver reports a zero absolute position.
func (r *treeRow) Dragged(event *fyne.DragEvent) {
	r.drag.DX += event.Dragged.DX
	r.drag.DY += event.Dragged.DY
}

// DragEnd moves a connection onto the folder under the pointer.
func (r *treeRow) DragEnd(event *fyne.DragEvent) {
	point := event.AbsolutePosition
	if point == (fyne.Position{}) && r.screen != nil {
		origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(r)
		point = origin.Add(event.Position).Add(fyne.NewPos(r.drag.DX, r.drag.DY))
	}
	r.drag = fyne.Delta{}
	if r.screen == nil || r.kind != rowConnection {
		return
	}
	r.screen.dropConnection(r.uid, point)
}
