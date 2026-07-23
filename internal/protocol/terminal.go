package protocol

import "io"

// TerminalProtocol is implemented by backends whose session is a raw
// bidirectional byte stream meant to be rendered by a terminal emulator:
// SSH, Telnet, rlogin, raw socket and serial (stage 2.2). It is a separate,
// composed interface rather than part of Protocol itself because
// window-embedded backends (RDP, VNC, AnyDesk — later stages) have no
// byte-stream concept on the caller's side; reparenting handles their
// rendering instead.
//
// Read/Write are only valid between a successful Connect and the session
// ending (OnClose firing); calling them outside that window returns an
// error. The terminal emulator widget that renders this stream is Phase 3
// UI work and does not exist yet — this interface is what it will consume.
type TerminalProtocol interface {
	Protocol
	io.ReadWriter
}
