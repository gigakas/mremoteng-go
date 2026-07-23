package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/settings"
)

// themeChoices lists the values the Theme select offers. Stage 3.6 (not
// built yet) is what will actually read Settings.Theme and apply a
// palette — this dialog only lets the value be set and persisted ahead
// of that, per the blueprint's stage split (3.5 owns settings
// persistence; 3.6 owns theming itself).
var themeChoices = []string{"system", "light", "dark"}

// ShowOptionsDialog shows a form for editing s in place, over win. onSave
// is called with the edited settings if the user confirms; the caller is
// responsible for actually persisting it (settings.Settings.Save) — this
// function only collects the edit, matching the same pattern
// SessionTabs/ConnectionTree use of keeping internal/ui's widgets free of
// direct disk I/O in their own event handlers where avoidable.
func ShowOptionsDialog(win fyne.Window, s *settings.Settings, onSave func(*settings.Settings)) {
	theme := widget.NewSelect(themeChoices, nil)
	theme.SetSelected(s.Theme)

	lastFile := widget.NewEntry()
	lastFile.SetText(s.LastConnectionsFile)
	lastFile.PlaceHolder = "(none)"

	items := []*widget.FormItem{
		widget.NewFormItem("Theme", theme),
		widget.NewFormItem("Last connections file", lastFile),
	}

	dialog.ShowForm("Options", "Save", "Cancel", items, func(confirmed bool) {
		if !confirmed {
			return
		}
		updated := *s
		updated.Theme = theme.Selected
		updated.LastConnectionsFile = lastFile.Text
		if onSave != nil {
			onSave(&updated)
		}
	}, win)
}
