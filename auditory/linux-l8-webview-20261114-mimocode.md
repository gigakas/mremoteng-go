# Audit — Linux track, Stage L8 (embedded web view)

- **Date (UTC)**: 2026-11-14
- **Agent**: mimocode
- **Audited stage**: L8 — additional protocols, "HTTP/HTTPS via native webview (the only
  allowed cgo wrapper)", the last item of the stage
- **Commits covered**: this commit

## 1. Code quality

An HTTP profile used to hand the address to the system browser. It now embeds a WebKit
view in the tab, using the same window machinery the RDP session already had.

**Why a helper process.** Fyne draws everything into its own window and cannot host a
foreign toolkit, and WebKitGTK needs a GTK main loop that would have to share the
application's. So the split is:

- `cmd/mremoteng-webview` — a small cgo binary (GTK 3 + WebKitGTK 4.1) that takes a window
  id and a URL, creates a `GtkPlug` with a `WebKitWebView` and runs `gtk_main`.
- `internal/protocol/webview` — finds that binary (next to the application first, then on
  the `PATH`), launches it with the window id and the URL, keeps its stderr for a failure
  report, and kills it on `Close`.
- `internal/ui/webview.go` — creates the native area with the existing embedder, starts
  the helper, keeps the session on the panel, and reports the view closing.

Because the view lives in the same kind of area FreeRDP uses, three behaviours came for
free and needed no new code: the tab switch maps and unmaps it (`setEmbeddedVisible`), a
dialog unmaps it (`beginModal`) so the dialog is not hidden behind it, and the 250 ms
geometry poll resizes it — `fitChildren` already sizes the children of the area.

**A finding worth recording: `GDK_BACKEND=x11` is mandatory.** The area is an X11 window
(through XWayland on a Wayland desktop). With GTK's default backend, `gtk_plug_new`
refuses (`GtkPlug only works under X11`) and the helper then segfaults inside
`gtk_plug_new`. `Launch` therefore adds `GDK_BACKEND=x11` to the child environment, which
is the only place that belongs: the variable changes how the helper talks to the display,
not how the application does.

**Dialogs and the view.** Nothing to do: the modal machinery hides every embedded window,
and the web view is an embedded window. `TestShowModal_HidesTheEmbeddedSession` covers the
mechanism for RDP; the web view goes through the same code.

**Fallbacks, all of them visible.** No X11 → the system browser, with the reason in the
status line. Helper missing or failing to start → the area is destroyed, the browser opens
and the status line says why. The tab's own note (`webViewNote`) reads
`webview.FindExecutable` so it promises the embedded view only when it can deliver it.

## 2. Performance

The helper is a separate process with its own GTK loop; the application's memory does not
grow by a WebKit engine, and closing the tab ends the helper (checked on
`disconnectShell`, on application shutdown and by the watcher). WebKit's own memory is
WebKit's: a page holding a video is the page's cost, not the window's.

## 3. Architecture

**New dependency — justification.** This is the cgo wrapper the blueprint allows for
HTTP/HTTPS, and the only one in the repository. Measured before choosing:

- `webkit2gtk-4.1` and `gtk+-3.0` development files are present on this machine
  (2.52.6 / 3.24.52), and the runtime libraries are a Debian package
  (`libwebkit2gtk-4.1-0`) that `scripts/package-deb.sh` now lists in `Depends`, together
  with `libgtk-3-0`.
- `github.com/webview/webview_go` was considered and rejected: its Linux backend takes a
  `GtkWidget*` and starts its own `gtk_main`, so it can only be used inside a GTK process —
  the helper would have been needed anyway, and the wrapper adds a layer without removing
  one.
- The alternative "GTK inside the application" was rejected outright: two event loops in
  one process, and GTK's `gtk_init` alongside Fyne's GLFW/X11 setup.

Nothing GTK or WebKit is linked into `cmd/mremoteng`; the boundary is one binary and one
`exec`. `mremoteng-webview` is built by `scripts/dev.sh`, packaged by `scripts/release.sh`,
installed next to the application (which is where `FindExecutable` looks first) and started
by `scripts/smoke.sh -version`.

The protocol package keeps no X dependency: it takes a window id, which is what
`embeddedArea.ID()` already returns.

## 4. Evidence

- `./scripts/check.sh`: OK (gofmt, vet, all tests, including this package's cgo build).
- `./scripts/smoke.sh`: OK, and it now runs `mremoteng-webview -version`.
- The helper embedding into a plain X11 window, checked with `xwininfo` on this machine:

```
socket 0x1800001 (created with xgb)
 1 child:
 0x2a0000c "mRemoteNG Go": ("mremoteng-webview" "Mremoteng-webview")  400x400+0+0
   1 child:
   0x2a0000d (no name): ()  1x1+-1+-1
```

  i.e. the helper creates its plug inside the window the application gives it, which is the
  whole mechanism. The first attempt without `GDK_BACKEND=x11` produced
  `Gtk-WARNING: GtkPlug only works under X11` followed by a SIGSEGV, which is why the
  variable is set explicitly and covered by a test.
- New tests: `TestLaunch_PassesTheWindowTheURLAndTheX11Backend`,
  `TestLaunch_RejectsWhatItCannotDo`, `TestSession_Close_EndsTheHelperAndWaitReportsIt`,
  `TestFindExecutable_FindsTheHelperBesideTheApplication`,
  `TestSummarize_PicksTheLastLineAndTrimsTheGtkPrefix`, `TestHTTPURL_KeepsTheSchemePortsOutOfTheAddress`,
  `TestConnectHTTP_EmbedsTheViewInTheTab`, `TestConnectHTTP_WithoutAWindow_OpensTheBrowser`,
  `TestConnectHTTP_HelperFailure_FallsBackToTheBrowser`,
  `TestWatchWebView_ReportsTheHelperCause`, `TestDisconnectHTTP_WithoutAView_DoesNothing`,
  `TestWebViewNote_DependsOnTheHelperBeingAvailable`.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Human: an HTTP profile with a real page, then tab switching, a dialog on top of it and a
  window resize — the page must follow all three.
- The view has **no navigation controls** (no back/forward/reload/address bar): the profile
  fixes the address. mRemoteNG's HTTP panel has none either, but this is the obvious next
  step if the embedded view is used for real browsing.
- There is no per-profile setting to choose browser versus embedded view; the fallback
  happens only when embedding is impossible.
- WebKit keeps its own cookies and cache under the user's data directory, outside the
  profile store; that is intentional (same behaviour as a browser) and documented here so
  it is not mistaken for a leak.
