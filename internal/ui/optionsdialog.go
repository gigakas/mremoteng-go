package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/settings"
)

// ShowOptionsDialog shows a form for editing s in place, over win. onSave
// is called with the edited settings if the user confirms; the caller is
// responsible for actually persisting it (settings.Settings.Save) and for
// applying a changed Theme value (ApplyTheme) — this function only
// collects the edit, matching the same pattern SessionTabs/ConnectionTree
// use of keeping internal/ui's widgets free of direct disk I/O or
// app-wide side effects in their own event handlers where avoidable.
func ShowOptionsDialog(win fyne.Window, s *settings.Settings, onSave func(*settings.Settings)) {
	themeSelect := widget.NewSelect(ThemeChoices, nil)
	themeSelect.SetSelected(s.Theme)

	lastFile := widget.NewEntry()
	lastFile.SetText(s.LastConnectionsFile)
	lastFile.PlaceHolder = "(none)"

	items := []*widget.FormItem{
		widget.NewFormItem("Theme", themeSelect),
		widget.NewFormItem("Last connections file", lastFile),
	}

	dialog.ShowForm("Options", "Save", "Cancel", items, func(confirmed bool) {
		if !confirmed {
			return
		}
		updated := *s
		updated.Theme = themeSelect.Selected
		updated.LastConnectionsFile = lastFile.Text
		if onSave != nil {
			onSave(&updated)
		}
	}, win)
}
