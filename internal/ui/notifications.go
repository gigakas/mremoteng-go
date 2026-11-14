package ui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// notificationLevel separates what needs attention from the chatter that belongs in
// the activity log.
type notificationLevel int

const (
	noticeError notificationLevel = iota
	noticeWarning
)

// String is the label the panel prints for a level.
func (l notificationLevel) String() string {
	if l == noticeWarning {
		return "warning"
	}
	return "error"
}

// notification is one message worth re-reading after the status bar has moved on.
// Nothing here is persisted: it describes this run, and a problem that outlives the
// window belongs in the activity log the user can copy.
type notification struct {
	At    time.Time
	Level notificationLevel
	Text  string
}

// notificationLimit bounds the history so a failing loop cannot grow it without end.
const notificationLimit = 100

// notifications is the bounded history behind the panel.
type notifications struct {
	entries []notification
}

// add records a message, dropping the oldest entries past the limit.
func (n *notifications) add(level notificationLevel, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	n.entries = append(n.entries, notification{At: time.Now(), Level: level, Text: oneLine(text)})
	if len(n.entries) > notificationLimit {
		n.entries = n.entries[len(n.entries)-notificationLimit:]
	}
}

// clear empties the history.
func (n *notifications) clear() { n.entries = nil }

// count is how many messages are waiting to be read.
func (n *notifications) count() int { return len(n.entries) }

// render lists the history one line per message, oldest first.
func (n *notifications) render() string {
	if len(n.entries) == 0 {
		return "No problems reported."
	}
	lines := make([]string, 0, len(n.entries))
	for _, entry := range n.entries {
		lines = append(lines, fmt.Sprintf("%s  %-7s %s", entry.At.Format("15:04:05"), entry.Level, entry.Text))
	}
	return strings.Join(lines, "\n")
}

// notificationPanel is the collapsible bottom panel with the problems the status bar
// has already replaced.
type notificationPanel struct {
	box   *fyne.Container
	title *widget.Label
	text  *widget.Label
	clear *widget.Button
}

func newNotificationPanel(s *screen) *notificationPanel {
	panel := &notificationPanel{}
	panel.title = widget.NewLabelWithStyle("Notifications", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	panel.text = widget.NewLabel("")
	panel.text.TextStyle = fyne.TextStyle{Monospace: true}
	// One message per line keeps the panel height fixed as entries arrive.
	panel.text.Wrapping = fyne.TextWrapOff
	scroll := container.NewVScroll(panel.text)
	scroll.SetMinSize(fyne.NewSize(200, 120))
	panel.clear = widget.NewButton("Clear", func() {
		s.notifications.clear()
		panel.refresh(s)
		s.refreshNotificationBadge()
		s.setStatus("Notifications cleared.")
	})
	panel.clear.Importance = widget.LowImportance
	header := container.NewBorder(nil, nil, panel.title, panel.clear)
	panel.box = container.NewBorder(container.NewVBox(header, widget.NewSeparator()), nil, nil, nil,
		container.NewPadded(scroll))
	return panel
}

// refresh repaints the panel and its title from the history.
func (p *notificationPanel) refresh(s *screen) {
	if p == nil || p.text == nil {
		return
	}
	if count := s.notifications.count(); count > 0 {
		p.title.SetText(fmt.Sprintf("Notifications (%d)", count))
	} else {
		p.title.SetText("Notifications")
	}
	p.text.SetText(s.notifications.render())
}

// notify records a problem and updates the panel and its badge. Every failure the
// window reports goes through here, so nothing it told the status bar is lost the
// moment the next message arrives.
func (s *screen) notify(level notificationLevel, text string) {
	s.notifications.add(level, text)
	s.notificationPanel.refresh(s)
	s.refreshNotificationBadge()
}

// refreshNotificationBadge shows how many problems are waiting on the status bar,
// because an icon alone would leave the panel undiscovered after the first failure.
func (s *screen) refreshNotificationBadge() {
	if s.notificationsButton == nil {
		return
	}
	count := s.notifications.count()
	if count > 0 {
		s.notificationsButton.SetText(itoa(count))
	} else {
		s.notificationsButton.SetText("")
	}
	s.notificationsButton.tip = notificationsHint(count)
}

// notificationsHint is the line the status bar shows while the pointer rests on the
// badge.
func notificationsHint(count int) string {
	switch count {
	case 0:
		return "Show the problems the status bar has already forgotten"
	case 1:
		return "Show 1 problem the status bar has already forgotten"
	}
	return fmt.Sprintf("Show %d problems the status bar has already forgotten", count)
}
