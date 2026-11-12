package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/vault"
)

// diagnosticsText renders the support summary. It is pure so it can be tested
// and never includes secrets.
func diagnosticsText(build, storePath string, profileCount, folderCount int, vaultState string, sessions int) string {
	var builder strings.Builder
	builder.WriteString("mremoteng-go " + build + "\n")
	builder.WriteString("profile store: " + storePath + "\n")
	fmt.Fprintf(&builder, "profiles: %d, folders: %d\n", profileCount, folderCount)
	builder.WriteString("credential vault: " + vaultState + "\n")
	fmt.Fprintf(&builder, "open sessions: %d\n", sessions)
	return builder.String()
}

// vaultState reports the vault status without touching the key.
func (s *screen) vaultState() string {
	if s.vaultPath == "" {
		return "not configured"
	}
	if s.vault == nil {
		return "locked (file present: " + boolText(vault.Exists(s.vaultPath)) + ")"
	}
	if s.vault.Locked() {
		return "locked"
	}
	return "unlocked"
}

func boolText(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

// requestDiagnostics shows the support summary and copies it on request.
func (s *screen) requestDiagnostics() {
	storePath := "(store does not expose a path)"
	if aware, ok := s.store.(pathStore); ok {
		storePath = aware.Path()
	}
	text := diagnosticsText(windowTitle(), storePath, len(s.all), len(s.folders), s.vaultState(), len(s.panels))
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Monospace: true}
	var popup *widget.PopUp
	content := container.NewVBox(
		widget.NewLabelWithStyle("Diagnostics", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		label,
		container.NewHBox(
			widget.NewButton("Copy", func() {
				clipboard := s.win.Clipboard()
				if clipboard != nil {
					clipboard.SetContent(text)
				}
				s.setStatus("Diagnostics copied to the clipboard.")
			}),
			widget.NewButton("Close", func() { popup.Hide() }),
		),
	)
	popup = s.showModal(content, fyne.NewSize(560, 260))
}

// activeSessionCount counts panels holding a live session.
func (s *screen) activeSessionCount() int {
	count := 0
	for _, panel := range s.allPanels() {
		if panel.shell != nil || panel.rdp != nil || panel.vnc != nil || panel.stream != nil {
			count++
		}
	}
	return count
}

// shutdownSessions ends every live session and embedded window before the
// application exits, so no FreeRDP process or X11 child is left behind.
func (s *screen) shutdownSessions() {
	for _, panel := range s.allPanels() {
		panel.shutdown()
	}
}
