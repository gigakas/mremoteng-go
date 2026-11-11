---
timestamp: 2026-11-11T22:29:16Z
agent: mimocode
files:
  - internal/ui/session.go
  - internal/ui/sessionstate.go
  - internal/ui/stream.go
  - internal/ui/ui_test.go
  - internal/ui/vnc.go
---

Give session state a single owner across the four session protocols

session.go, stream.go and vnc.go each repeated the same bookkeeping around a connection: eleven button triplets that enabled Connect and Disable while disabling the password field (or the reverse), six slot assignments with a manual Refresh, four connect-disable plus status pairs, and two byte-for-byte identical read loops with a 32 KiB buffer. internal/ui/sessionstate.go now owns setConnecting, setConnected, installSlot and pumpBytes, and every protocol path calls them: SSH, RDP, Telnet/RAW/rlogin and VNC. Deliberately NOT a sessionRunner interface: the four lifecycles genuinely differ (SSH holds a pty, the stream protocols share a socket, VNC drives a framebuffer, RDP waits on an external process), so an interface would have needed three of its methods to be no-ops. The shared part is bookkeeping, so it is a set of functions, not a contract. Two behavioural consequences, both intentional: every ended session now goes through one state transition, so the RDP immediate-failure path re-enables the password field it previously left disabled, and sessionClosed's inline copy of the Config-panel restore is gone in favour of restoreConfig. Added tests for the two setConnected transitions and for installSlot. Validated with ./scripts/check.sh.
