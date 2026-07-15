# Spike 0.1 findings — X11 embedding of external xfreerdp (Linux)

- **Date**: 2026-07-15 · **Agent**: claude-code · **Validated by**: human, on
  GNOME **Wayland** (everything below therefore ran via XWayland).
- **Verdict: the approach works.** External xfreerdp process embedded in a
  Fyne window, resize with scaling, keyboard focus in/out, and process-exit
  cleanup all validated visually. Full checklist in
  `internal/spike/README.md` passed.

## What works, and how

1. **Primary mechanism — `/parent-window:<xid>`** (xfreerdp flag): the
   session window is created directly as a child of the Fyne window. No
   window manager involvement, no races. This mirrors what the original
   mRemoteNG does on Windows (`PuTTYNG -hwndparent`, RDP ActiveX host).
   Child discovery via `xproto.QueryTree` on our window.
2. **Resize**: followed via `ConfigureNotify` on the parent +
   `ConfigureWindow` on the child (pure-Go `BurntSushi/xgb`, no cgo).
   Content scaling with `/smart-sizing` (client-side).
3. **Exit detection**: `DestroyNotify` (SubstructureNotify mask on parent)
   detects the child's death reliably; panel cleanup validated.
4. **Fyne native access**: `driver.NativeWindow.RunNative` →
   `driver.X11WindowContext.WindowHandle` gives the X11 parent id (Fyne
   2.8, default build = X11 backend, i.e. XWayland under Wayland).

## Pitfalls found (they drive the Phase 2.5 design)

- **Naive reparenting fails**: grabbing xfreerdp's top-level (via
  `_NET_CLIENT_LIST` + `_NET_WM_PID`) and calling `ReparentWindow` embeds a
  window that xfreerdp **destroys and recreates** during connection
  finalization — the final session window pops up as an independent
  toplevel. A generic reparent-based embedder (needed for AnyDesk, which has
  no `/parent-window` equivalent) must watch for re-creation and re-embed.
  Kept in the spike as `-mode reparent`, unfinished on purpose.
- **`/dynamic-resolution` kills the session against xrdp** (container test
  host): the disp channel negotiation ends in `update_recv failed` and a
  disconnect. `/smart-sizing` is the safe default; per-host opt-in to
  dynamic resolution against real Windows RDP hosts is a Phase 2.5
  decision.
- **HiDPI**: X11 geometry is physical pixels, Fyne sizes are logical
  points. Any overlay offset must be multiplied by `Canvas().Scale()`.
- **Focus**: no explicit XEmbed protocol needed for this spike — click
  focus works; a production tab host will need deliberate focus handoff
  (Phase 2.1 interface should expose Focus()).

## Test-host note

`lscr.io/linuxserver/rdesktop:ubuntu-xfce` under podman stalled its s6 init:
`svc-xrdp` stayed "down (not started yet)" (only `xrdp-sesman` came up) and
the daemon had to be started manually with
`podman exec spike-xrdp /usr/sbin/xrdp --nodaemon`. Fine for a spike; **not
reliable for CI** — Phase 4.2 should provision xrdp explicitly instead of
relying on this image.

## Impact on the rest of Phase 0

- 0.3 (Wayland assessment) already has its central datum: the whole
  validation ran under XWayland on GNOME Wayland without compositor issues.
- 0.2 (Win32) expectation unchanged: `SetParent` is the original project's
  proven mechanism.
