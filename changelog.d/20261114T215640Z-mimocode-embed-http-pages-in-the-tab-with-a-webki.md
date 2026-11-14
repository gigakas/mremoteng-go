---
timestamp: 2026-11-14T21:56:40Z
agent: mimocode
files:
  - cmd/mremoteng-webview/main.go
  - cmd/mremoteng-webview/main_nocgo.go
  - internal/protocol/webview/webview.go
  - internal/protocol/webview/webview_test.go
  - internal/ui/webview.go
  - internal/ui/webview_test.go
  - internal/ui/session.go
  - internal/ui/stream.go
  - internal/ui/shell.go
  - scripts/dev.sh
  - scripts/release.sh
  - scripts/smoke.sh
  - scripts/package-deb.sh
  - auditory/linux-l8-webview-20261114-mimocode.md
  - blueprint/backlog.md
  - blueprint/linux-preview.md
---

Embed HTTP pages in the tab with a WebKit helper

An HTTP profile used to hand the address to the system browser. It now embeds a WebKit view in the tab, in the same native window FreeRDP uses, so the tab switch, the dialog-hiding work and the geometry poll all apply without new code. Fyne cannot host a foreign toolkit and WebKitGTK needs its own GTK main loop, so the view runs in a helper process: cmd/mremoteng-webview creates a GtkPlug with a WebKitWebView in the window the application gives it, and internal/protocol/webview finds, launches and stops it. Nothing GTK or WebKit is linked into the application, the helper ships next to it and scripts/package-deb.sh lists libwebkit2gtk-4.1-0 and libgtk-3-0. One finding decided the design: with GTK's default backend GtkPlug refuses GtkPlug only works under X11 and the helper segfaults, so Launch sets GDK_BACKEND=x11 for the child. Every fallback is visible: no window, a missing helper or one that fails all end in the system browser with the reason in the status line, and the tab note reads FindExecutable so it promises only what it can deliver. The helper was verified embedding into a plain X11 window with xwininfo. Validated with ./scripts/check.sh and ./scripts/smoke.sh.
