//go:build !windows

package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
)

// Embed is not implemented on this platform yet — the same honest gap as
// internal/protocol/rdp and internal/protocol/anydesk's own Linux paths
// (no X server available in the session that wrote this to validate an
// X11 reparenting implementation against). See docs/spike-x11.md for the
// validated Linux mechanism (xfreerdp's /parent-window flag; a generic
// XReparentWindow-based approach for AnyDesk, left "unfinished on
// purpose" by the spike) that a future implementation should follow.
func (h *NativeWindowHost) Embed(win fyne.Window, handle uintptr) error {
	return fmt.Errorf("ui: NativeWindowHost.Embed is not implemented on this platform")
}
