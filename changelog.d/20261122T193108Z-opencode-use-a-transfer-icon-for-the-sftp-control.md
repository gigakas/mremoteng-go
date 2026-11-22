---
timestamp: 2026-11-22T19:31:08Z
agent: opencode
files:
  - internal/ui/assets/sftp.svg
  - internal/ui/icon.go
  - internal/ui/session.go
  - internal/ui/properties.go
---

Use a transfer icon for the SFTP control

The SSH session bar opened the SFTP window from Fyne's folder icon, which reads as 'open the file manager' rather than 'move files between this machine and the host'. internal/ui/assets/sftp.svg adds a two-arrow glyph (up and down) drawn for the 24 px toolbar, exposed as ui.SFTPIcon() with the theme's icon colour through theme.NewThemedResource, and both the session bar and the Welcome page's button card now use it. The control still appears only for SSH connections, where file transfer exists.
