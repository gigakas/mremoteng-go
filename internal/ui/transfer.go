package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	"github.com/mRemoteNG/mremoteng-go/internal/transfer"
)

// requestImport opens a file picker for an mRemoteNG XML or CSV file.
func (s *screen) requestImport() {
	picker := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer func() {
			if closeErr := reader.Close(); closeErr != nil {
				s.setProblem("Could not close the imported file: " + closeErr.Error())
			}
		}()
		data, readErr := os.ReadFile(reader.URI().Path())
		if readErr != nil {
			s.setProblem("Could not read the file: " + readErr.Error())
			return
		}
		s.importBytes(reader.URI().Path(), data, "")
	}, s.win)
	picker.SetFilter(storage.NewExtensionFileFilter([]string{".xml", ".csv"}))
	picker.Show()
}

// importBytes parses a file and shows the preview. A protected XML file asks for
// its password first.
func (s *screen) importBytes(path string, data []byte, password string) {
	preview, err := importPreview(path, data, password)
	if errors.Is(err, transfer.ErrNeedPassword) {
		s.showSecretDialog("Password for "+filepath.Base(path), "Decrypt", func(secret string) {
			s.importBytes(path, data, secret)
		})
		return
	}
	if err != nil {
		s.setProblem("Import failed: " + err.Error())
		return
	}
	s.showImportPreview(path, preview)
}

// importPreview detects the format and parses it.
func importPreview(path string, data []byte, password string) (transfer.Preview, error) {
	extension := strings.ToLower(filepath.Ext(path))
	if extension == ".xml" || (extension != ".csv" && strings.Contains(string(data), "<Connections")) {
		return transfer.ReadXML(data, password)
	}
	return transfer.ReadCSV(data)
}

// showImportPreview asks how to apply the import.
func (s *screen) showImportPreview(path string, preview transfer.Preview) {
	summary := importSummary(preview, s.duplicateCount(preview))
	content := container.NewVBox(
		widget.NewLabelWithStyle(filepath.Base(path)+" ("+string(preview.Format)+")", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(summary),
	)
	var popup *widget.PopUp
	buttons := container.NewHBox(
		widget.NewButton("Merge", func() {
			popup.Hide()
			s.applyImport(preview, false)
		}),
		widget.NewButton("Replace", func() {
			popup.Hide()
			s.applyImport(preview, true)
		}),
		widget.NewButton("Cancel", func() { popup.Hide() }),
	)
	content.Add(buttons)
	popup = s.showModal(content, fyne.NewSize(520, 190))
}

// importSummary describes what the import would do.
func importSummary(preview transfer.Preview, duplicates int) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%d connection(s) and %d folder(s) found.", len(preview.Profiles), len(preview.Folders))
	if duplicates > 0 {
		fmt.Fprintf(&builder, " %d already exist and will be skipped when merging.", duplicates)
	}
	if preview.Unsupported > 0 {
		fmt.Fprintf(&builder, " %d row(s) skipped: only SSH and RDP are supported.", preview.Unsupported)
	}
	if preview.Dropped > 0 {
		fmt.Fprintf(&builder, " %d password(s) were dropped: saved passwords are never imported.", preview.Dropped)
	}
	if preview.Encrypted {
		builder.WriteString(" The file was protected.")
	}
	return builder.String()
}

// duplicateCount reports how many imported profiles already exist by name and
// folder, which is what merging skips.
func (s *screen) duplicateCount(preview transfer.Preview) int {
	existing := map[string]bool{}
	for _, profile := range s.ws.profiles {
		existing[strings.ToLower(profile.Name)+"\x00"+profile.Folder] = true
	}
	count := 0
	for _, profile := range preview.Profiles {
		if existing[strings.ToLower(profile.Name)+"\x00"+profile.Folder] {
			count++
		}
	}
	return count
}

// applyImport merges or replaces the store with the imported connections.
func (s *screen) applyImport(preview transfer.Preview, replace bool) {
	if !s.requireWritable("Importing") {
		return
	}
	profilesOut := make([]profiles.Profile, 0, len(s.ws.profiles)+len(preview.Profiles))
	foldersOut := make([]string, 0, len(s.ws.folders)+len(preview.Folders))
	seen := map[string]bool{}
	if !replace {
		profilesOut = append(profilesOut, s.ws.profiles...)
		foldersOut = append(foldersOut, s.ws.folders...)
		for _, profile := range s.ws.profiles {
			seen[strings.ToLower(profile.Name)+"\x00"+profile.Folder] = true
		}
	}
	added, skipped := 0, 0
	for _, profile := range preview.Profiles {
		key := strings.ToLower(profile.Name) + "\x00" + profile.Folder
		if seen[key] {
			skipped++
			continue
		}
		seen[key] = true
		profilesOut = append(profilesOut, profile)
		added++
	}
	foldersOut = unionFolders(foldersOut, preview.Folders)
	if err := s.ws.store.Save(s.ws.document(profilesOut, foldersOut)); err != nil {
		s.setProblem("Could not save the imported connections: " + err.Error())
		return
	}
	s.ws.profiles = profilesOut
	s.ws.folders = foldersOut
	s.refreshTree()
	s.tree.OpenAllBranches()
	mode := "merged"
	if replace {
		mode = "replaced"
	}
	s.setStatus(fmt.Sprintf("Import %s: %d connection(s) added, %d skipped.", mode, added, skipped))
}

// unionFolders appends folders that are not present yet, parents first.
func unionFolders(existing []string, incoming []string) []string {
	known := map[string]bool{}
	for _, folder := range existing {
		known[folder] = true
	}
	result := append([]string(nil), existing...)
	for _, folder := range incoming {
		if !known[folder] {
			known[folder] = true
			result = append(result, folder)
		}
	}
	return result
}

// requestExport writes the current connections as XML.
func (s *screen) requestExportXML() {
	s.saveExport(".xml", func(password string) ([]byte, error) {
		if password == "" {
			return transfer.WriteXML(s.ws.profiles, s.ws.folders, "")
		}
		return transfer.WriteXML(s.ws.profiles, s.ws.folders, password)
	})
}

// requestExportCSV writes the current connections as CSV.
func (s *screen) requestExportCSV() {
	s.saveExport(".csv", func(string) ([]byte, error) {
		return transfer.WriteCSV(s.ws.profiles, s.ws.folders)
	})
}

// saveExport asks for the destination and writes the export.
func (s *screen) saveExport(extension string, build func(password string) ([]byte, error)) {
	saver := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		path := writer.URI().Path()
		if closeErr := writer.Close(); closeErr != nil {
			s.setProblem("Could not prepare the export file: " + closeErr.Error())
			return
		}
		write := func(password string) {
			data, buildErr := build(password)
			if buildErr != nil {
				s.setProblem("Export failed: " + buildErr.Error())
				return
			}
			if writeErr := os.WriteFile(path, data, 0o600); writeErr != nil {
				s.setProblem("Could not write the export: " + writeErr.Error())
				return
			}
			s.setStatus("Exported " + fmt.Sprintf("%d connection(s) to %s.", len(s.ws.profiles), filepath.Base(path)))
		}
		if extension == ".xml" {
			s.showSecretDialog("Password for the export (optional)", "Export", write)
			return
		}
		write("")
	}, s.win)
	saver.SetFileName("mremoteng-connections" + extension)
	saver.Show()
}

// showSecretDialog asks for a password in a compact popup.
func (s *screen) showSecretDialog(title, confirm string, onConfirm func(string)) {
	entry := widget.NewEntry()
	entry.Password = true
	entry.SetPlaceHolder("Password")
	var popup *widget.PopUp
	content := container.NewVBox(
		widget.NewLabel(title),
		entry,
		container.NewHBox(
			widget.NewButton(confirm, func() {
				popup.Hide()
				onConfirm(entry.Text)
			}),
			widget.NewButton("Cancel", func() { popup.Hide() }),
		),
	)
	popup = s.showModal(content, fyne.NewSize(textDialogWidth, textDialogHeight))
}
