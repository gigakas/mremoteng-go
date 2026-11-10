// Command mremoteng is the desktop application for managing connection
// profiles. It opens the Fyne window and stores profiles in the user's
// configuration directory; protocol sessions arrive with the session stage.
package main

import (
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
}
