package ui

import (
	"fmt"
	"strings"
)

// DesktopEntry renders the .desktop file for the current build. execPath must be
// absolute: desktop shells do not resolve relative paths.
func DesktopEntry(execPath, iconPath string) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Version=1.0
Name=mRemoteNG Go
GenericName=Connection manager
Comment=Manage SSH and RDP connections, mRemoteNG style
Exec=%s
Icon=%s
Terminal=false
Categories=Network;RemoteAccess;
Keywords=ssh;rdp;remote;mremoteng;
StartupWMClass=%s
`, execPath, iconPath, WindowClass)
}

// DesktopEntryExec extracts the Exec line, used by tests and installers.
func DesktopEntryExec(entry string) string {
	for _, line := range strings.Split(entry, "\n") {
		if strings.HasPrefix(line, "Exec=") {
			return strings.TrimPrefix(line, "Exec=")
		}
	}
	return ""
}
