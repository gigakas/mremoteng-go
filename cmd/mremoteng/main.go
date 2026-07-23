package main

import (
	"log"

	"fyne.io/fyne/v2/app"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
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

	// No persistence yet (stage 3.5) and no "open file" flow yet (stage
	// 3.4/3.5), so the tree starts empty — an empty real connection.Node
	// tree, not the placeholder label NewShell defaults to. Loading a
	// real .xml connections file into this tree is what will actually
	// satisfy Phase 2/3's shared "demo config file" exit criterion.
	root, err := connection.NewRootInfo()
	if err != nil {
		log.Fatalf("mremoteng: create connection tree root: %v", err)
	}
	shell.SetTree(ui.NewConnectionTree(root).Widget)

	shell.Window.ShowAndRun()
}
