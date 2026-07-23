package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
	"github.com/mRemoteNG/mremoteng-go/internal/settings"
	"github.com/mRemoteNG/mremoteng-go/internal/ui"

	// Blank-imported so each protocol backend's init() registers itself
	// with internal/protocol's factory (see internal/protocol/factory.go's
	// Register doc comment). Add a line here for every protocol this
	// binary should support; a build that wants a smaller footprint can
	// drop backends by removing their import.
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/anydesk"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/raw"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/rdp"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/rlogin"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/serial"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/ssh"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/telnet"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/vnc"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/web"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/winrm"
)

func main() {
	a := app.NewWithID(ui.AppID)
	shell := ui.NewShell(a)

	settingsPath, err := settings.DefaultPath()
	if err != nil {
		log.Fatalf("mremoteng: resolve settings path: %v", err)
	}
	cfg, err := settings.Load(settingsPath)
	if err != nil {
		log.Fatalf("mremoteng: load settings: %v", err)
	}
	if cfg.WindowWidth > 0 && cfg.WindowHeight > 0 {
		shell.Window.Resize(fyne.NewSize(cfg.WindowWidth, cfg.WindowHeight))
	}
	saveSettings := func() {
		if err := cfg.Save(settingsPath); err != nil {
			log.Printf("mremoteng: save settings: %v", err)
		}
	}

	root, err := connection.NewRootInfo()
	if err != nil {
		log.Fatalf("mremoteng: create connection tree root: %v", err)
	}
	tree := ui.NewConnectionTree(root)
	shell.SetTree(tree.Widget)

	tabs := ui.NewSessionTabs(shell.Window)
	shell.SetTabs(tabs.Widget)

	properties := ui.NewPropertiesPanel()
	shell.SetProperties(properties)

	// Selecting anything (connection or folder) shows its properties —
	// both are connection.Node with the same ConnectionValues/
	// InheritanceFlags shape. Only a connection (leaf) also opens a
	// session tab; a folder selection just expands/collapses (handled by
	// the tree widget itself).
	tree.OnSelect = func(node connection.Node) {
		properties.SetTarget(node)

		conn, ok := node.(*connection.ConnectionInfo)
		if !ok {
			return
		}
		p, err := protocol.Create(conn)
		if err != nil {
			log.Printf("mremoteng: create protocol session: %v", err)
			return
		}
		tabs.Open(conn.Effective().Name, p)
	}

	shell.OnOpenConnectionsFile = func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, ferr error) {
			if ferr != nil {
				dialog.ShowError(ferr, shell.Window)
				return
			}
			if reader == nil {
				return // user cancelled
			}
			path := reader.URI().Path()
			reader.Close()

			promptConnectionsFilePassword(shell.Window, "Open Connections File", func(password []byte) {
				loaded, err := ui.LoadConnectionsFile(path, password)
				if err != nil {
					dialog.ShowError(err, shell.Window)
					return
				}
				tree.SetRoot(loaded)
				cfg.LastConnectionsFile = path
				saveSettings()
			})
		}, shell.Window)
	}

	shell.OnSaveConnectionsFile = func() {
		dialog.ShowFileSave(func(writer fyne.URIWriteCloser, ferr error) {
			if ferr != nil {
				dialog.ShowError(ferr, shell.Window)
				return
			}
			if writer == nil {
				return // user cancelled
			}
			path := writer.URI().Path()
			writer.Close()

			promptConnectionsFilePassword(shell.Window, "Save Connections File", func(password []byte) {
				if err := ui.SaveConnectionsFile(path, tree.Root(), password); err != nil {
					dialog.ShowError(err, shell.Window)
					return
				}
				cfg.LastConnectionsFile = path
				saveSettings()
			})
		}, shell.Window)
	}

	shell.OnOptions = func() {
		ui.ShowOptionsDialog(shell.Window, cfg, func(updated *settings.Settings) {
			cfg = updated
			saveSettings()
		})
	}

	shell.Window.SetOnClosed(func() {
		size := shell.Window.Canvas().Size()
		cfg.WindowWidth = size.Width
		cfg.WindowHeight = size.Height
		saveSettings()
	})

	shell.Window.ShowAndRun()
}

// promptConnectionsFilePassword asks for the password protecting a
// connections file (blank uses mRemoteNG's own default connection-file
// password, same as internal/serialize/xml.Options.Password) before
// calling onConfirm. Cancelling the form calls neither onConfirm nor the
// load/save it guards.
func promptConnectionsFilePassword(win fyne.Window, title string, onConfirm func(password []byte)) {
	entry := widget.NewPasswordEntry()
	entry.PlaceHolder = "(leave blank for the default connections-file password)"
	items := []*widget.FormItem{widget.NewFormItem("Password", entry)}

	dialog.ShowForm(title, "OK", "Cancel", items, func(confirmed bool) {
		if !confirmed {
			return
		}
		onConfirm([]byte(entry.Text))
	}, win)
}
