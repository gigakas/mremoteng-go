---
timestamp: 2026-11-14T18:54:20Z
agent: mimocode
files:
  - internal/ui/session.go
  - internal/ui/session_test.go
---

Put SFTP on the session bar and let the X disconnect

The session bar spent its width on a Disconnect button whose icon said nothing, while the file transfer lived three clicks away in a menu. An SSH session now shows an SFTP button in that slot, which opens the transfer window for that panel directly instead of guessing from the focus, and the X that used to close a split pane now disconnects the connection, placed after the split arrows where the hand already is. Closing a pane moved to View > Close pane, which already existed. The SFTP button appears only on SSH sessions, since the transfer needs an SSH connection, and pressing it on any other protocol is impossible rather than merely rejected. Tests: the SSH bar carries the SFTP button and the disconnect control while an RDP bar offers no SFTP, the pane bar holds exactly the two split directions and no close control, and opening SFTP on a VNC panel explains that transfer needs SSH. Validated with ./scripts/check.sh.
