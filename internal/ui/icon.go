package ui

import (
	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// iconSVG is the application icon: a monitor with an SSH and an RDP indicator,
// kept simple so it stays legible at taskbar sizes.
//
//go:embed assets/icon.svg
var iconSVG []byte

// sftpSVG is the transfer icon of an SSH session's SFTP control: two arrows in
// opposite directions, because Fyne's stock icons only offer a folder, which read
// as "open the file manager" instead of "move files between this machine and the
// host".
//
//go:embed assets/sftp.svg
var sftpSVG []byte

// AppIcon returns the embedded application icon.
func AppIcon() fyne.Resource {
	return fyne.NewStaticResource("mremoteng-go.svg", iconSVG)
}

// SFTPIcon returns the SFTP control icon, tinted with the theme's icon colour.
func SFTPIcon() fyne.Resource {
	return theme.NewThemedResource(fyne.NewStaticResource("sftp.svg", sftpSVG))
}
