---
timestamp: 2026-11-16T20:04:19Z
agent: opencode
files:
  - README.md
  - docs/screenshots/main-window.png
  - docs/screenshots/connection-details.png
  - docs/screenshots/activity-log.png
---

Add README screenshots and refresh the project status text

The README had no images and its status paragraph still claimed that SSH and RDP sessions were unimplemented (stage L1), which stopped being true when the L2 session work landed. docs/screenshots now holds three screenshots taken from a development build driven by a throwaway demo profile store, so no personal data appears: the main window with the start page and the connection tree, a selected connection with the Connection details panel, and the activity log. The new Screenshots section shows them, the status paragraph now describes what works (profiles, folders, import/export, vault, SSH/RDP/VNC/Telnet/RAW/HTTP and the packaging scripts), the Go requirement is corrected to 1.25 to match go.mod, and the Development section documents the two GitHub Actions workflows.
