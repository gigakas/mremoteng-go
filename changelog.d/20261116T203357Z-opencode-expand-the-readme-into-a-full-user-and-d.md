---
timestamp: 2026-11-16T20:33:57Z
agent: opencode
files:
  - README.md
---

Expand the README into a full user and developer guide

The README grew from a feature dump into a document someone can install from: badges for the two workflows, a table of contents, the three screenshots, features grouped by area (tree, details, sessions, workspace), a protocol support table stating what each backend actually does today, runtime and build requirements with apt/dnf/pacman commands, install instructions for the .deb, rpm, Flatpak, tarball and per-user prefix, the configuration and data layout (profiles.json, secrets.json, Fyne preferences, XDG handling, -profiles/-version), keyboard and mouse reference, security notes, the CI and release behaviour, an updated package layout, the development workflow and an accurate limitations list. The stale claims that HTTP opens in the system browser, that serial ports are unimplemented and that VNC ignores the keyboard were removed; the text now matches internal/ui/{webview,serial,vnc_input}.go, and the VNC clipboard gap is stated instead.
