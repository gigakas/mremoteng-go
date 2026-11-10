package protocol

import "image"

// PointerButtons is a bitmask of pointer (mouse) buttons for
// FramebufferProtocol.SendPointer. Backends translate it to whatever their
// wire protocol expects; callers never see a backend-specific button type.
type PointerButtons uint8

// Pointer button bits. WheelUp/WheelDown represent a single wheel step,
// following the VNC convention of modeling the wheel as two extra
// momentary buttons.
const (
	PointerButtonLeft PointerButtons = 1 << iota
	PointerButtonMiddle
	PointerButtonRight
	PointerButtonWheelUp
	PointerButtonWheelDown
)

// FramebufferProtocol is implemented by backends whose session is a
// server-pushed grid of pixels plus keyboard/pointer input, rendered as an
// image rather than text (VNC — stage 2.4). Like TerminalProtocol, it is a
// separate interface composed with Protocol rather than folded into it,
// because "what a session produces and consumes" differs fundamentally
// between protocol families (RDP/AnyDesk, later stages, produce neither: a
// reparented external window owns their rendering).
type FramebufferProtocol interface {
	Protocol

	// Frames returns a channel that receives a complete framebuffer image
	// every time the server pushes an update (not just the changed
	// region — the backend composites incremental updates onto its own
	// copy before emitting). The channel is closed when the session ends
	// (OnClose fires around the same time). Each received image is an
	// independent copy the receiver may retain or mutate freely.
	Frames() <-chan image.Image

	// SendKey reports a key press (down=true) or release (down=false),
	// using an X11 keysym to identify the key.
	SendKey(keysym uint32, down bool)

	// SendPointer reports the current pointer button state and position,
	// in framebuffer pixel coordinates.
	SendPointer(buttons PointerButtons, x, y int)
}
