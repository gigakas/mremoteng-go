package ui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

// iconSVG is the application icon: a monitor with an SSH and an RDP indicator,
// kept simple so it stays legible at taskbar sizes.
//
//go:embed assets/icon.svg
var iconSVG []byte

// AppIcon returns the embedded application icon.
func AppIcon() fyne.Resource {
	return fyne.NewStaticResource("mremoteng-go.svg", iconSVG)
}
