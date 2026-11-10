package ui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// logLimit bounds the activity console history; logVisibleLines bounds what the
// panel renders so its height stays fixed as messages arrive.
const (
	logLimit        = 200
	logVisibleLines = 8
)

// activityLog keeps the timestamped history shown by the log console.
type activityLog struct {
	entries []string
}

// add appends an entry, trimming the oldest ones. Multi-line payloads (a client
// dumping its log) are collapsed to one bounded line so the console keeps a
// fixed height.
func (l *activityLog) add(kind, text string) {
	entry := fmt.Sprintf("%s  %-7s %s", time.Now().Format("15:04:05"), kind, oneLine(text))
	l.entries = append(l.entries, entry)
	if len(l.entries) > logLimit {
		l.entries = l.entries[len(l.entries)-logLimit:]
	}
}

// render joins the last entries for display.
func (l *activityLog) render() string {
	return l.renderLast(logVisibleLines)
}

// renderLast joins at most n of the newest entries.
func (l *activityLog) renderLast(n int) string {
	if n <= 0 || len(l.entries) == 0 {
		return ""
	}
	start := len(l.entries) - n
	if start < 0 {
		start = 0
	}
	return strings.Join(l.entries[start:], "\n")
}

// logConsole is the collapsible bottom panel with the activity history.
type logConsole struct {
	box   *fyne.Container
	text  *widget.Label
	clear *widget.Button
}

// oneLine collapses whitespace and caps the length.
func oneLine(text string) string {
	replaced := strings.NewReplacer("\r", " ", "\n", " ", "\t", " ")
	collapsed := strings.Join(strings.Fields(replaced.Replace(text)), " ")
	const limit = 280
	if len(collapsed) > limit {
		collapsed = collapsed[:limit] + " …"
	}
	return collapsed
}

func newLogConsole(s *screen) *logConsole {
	console := &logConsole{}
	console.text = widget.NewLabel("")
	console.text.TextStyle = fyne.TextStyle{Monospace: true}
	// No wrapping keeps one entry per line, so the panel height does not change
	// as messages arrive.
	console.text.Wrapping = fyne.TextWrapOff
	scroll := container.NewVScroll(console.text)
	scroll.SetMinSize(fyne.NewSize(200, 120))
	console.clear = widget.NewButton("Clear", func() {
		s.log.entries = nil
		console.refresh(s)
		s.setStatus("Activity log cleared.")
	})
	console.clear.Importance = widget.LowImportance
	header := container.NewBorder(nil, nil,
		widget.NewLabelWithStyle("Activity log", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		console.clear)
	console.box = container.NewBorder(container.NewVBox(header, widget.NewSeparator()), nil, nil, nil,
		container.NewPadded(scroll))
	return console
}

// refresh repaints the console from the log history.
func (c *logConsole) refresh(s *screen) {
	if c == nil || c.text == nil {
		return
	}
	c.text.SetText(s.log.render())
}
