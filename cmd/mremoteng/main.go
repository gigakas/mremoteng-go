// Command mremoteng is the desktop application for managing connection
// profiles. It opens the Fyne window and stores profiles in the user's
// configuration directory; protocol sessions arrive with the session stage.
package main

import (
<<<<<<< HEAD
	"flag"
	"fmt"
	"os"

	"fyne.io/fyne/v2/app"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	"github.com/mRemoteNG/mremoteng-go/internal/ui"
)

// Build information is injected with -ldflags by scripts/dev.sh and the
// release build, so the window title and About dialog identify the running
// binary.
var (
	version = "devel"
	commit  = "dev"
	built   = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	profilePath := flag.String("profiles", "", "profile store path (default: user configuration directory)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("mremoteng-go %s (%s, built %s)\n", version, commit, built)
		return
	}

	path, err := resolveStorePath(*profilePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mremoteng:", err)
		os.Exit(1)
	}

	ui.SetBuildLabel(fmt.Sprintf("%s (%s)", version, commit))
	application := app.NewWithID("org.mremoteng.go")
	application.Settings().SetTheme(ui.SoberTheme())
	application.SetIcon(ui.AppIcon())
	window := ui.NewWindow(application, profiles.NewStore(path))
	window.Show()
	ui.SetWindowClass(window)
	if application.Preferences().IntWithFallback("window.x", -1) >= 0 {
		ui.RestoreWindowPosition(window, application.Preferences())
	} else {
		window.CenterOnScreen()
	}
	window.RequestFocus()
	application.Run()
}

func resolveStorePath(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	return profiles.DefaultPath()
=======
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
	ui.ApplyTheme(a, cfg.Theme)
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
			ui.ApplyTheme(a, cfg.Theme)
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
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
}
