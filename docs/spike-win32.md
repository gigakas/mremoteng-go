# Spike 0.2 findings — Win32 embedding of an external RDP client (Windows)

- **Date**: 2026-07-16 · **Agent**: claude-code · **Validated by**: human,
  on a Windows VM (VMware) connecting to the Linux host's xrdp container.
- **Verdict: the approach works.** External `mstsc.exe` embedded in a Fyne
  window via `SetParent`, content scaling on resize, keyboard focus in/out,
  and process-exit cleanup all validated visually. Full checklist in
  `internal/spike/README.md` passed.

## What works, and how

1. **Mechanism — `SetParent` + `WS_CHILD`** (`win32.go`): find the client's
   top-level window by PID (`EnumWindows` + `GetWindowThreadProcessId`),
   strip `WS_POPUP|WS_CAPTION|WS_THICKFRAME`, set `WS_CHILD`, `SetParent`
   into the Fyne window, then poll-driven resize-follow (`GetClientRect` +
   `MoveWindow`) and death detection (`IsWindow`), 200 ms cadence. This is
   the generic mechanism — works for any external process, which is what
   stages 2.5 (RDP) and 2.7 (AnyDesk) need on Windows.
2. **Zero-install test client — `mstsc.exe`**: the built-in Windows client
   embeds fine. Its credential prompt appears as a separate dialog before
   the session window exists (expected; the spike waits for the session
   window by PID).
3. **Scaling**: mstsc ignores CLI sizing options; **smart sizing only works
   via a `.rdp` file** (`smart sizing:i:1`) — the spike generates a temp
   one. Content scales preserving aspect ratio (letterboxing when the
   window ratio differs — acceptable).
4. **Fyne native access**: `driver.WindowsWindowContext.HWND` provides the
   parent handle — same `RunNative` pattern as X11.

## Notes for later phases

- **FreeRDP GitHub releases no longer ship Windows binaries** (3.29.0
  assets are source-only); prebuilt `wfreerdp.exe` lives in the nightly CI
  (ci.freerdp.com). Phase 4.4 (Windows packaging, FreeRDP binary alongside)
  must account for this: pin/mirror a nightly or build FreeRDP in our CI.
- `/parent-window` mode remains available in the spike for FreeRDP clients
  on Windows but was not validated (no wfreerdp binary at hand); SetParent
  is sufficient and client-agnostic.
- mstsc argument passing differs from FreeRDP's (`.rdp` file vs slash
  options) — the Phase 2.5 design needs a per-client argument builder, not
  a single template (already sketched as `clientArgs` in the spike).
- Polling (200 ms) was enough for resize-follow; a production version can
  hook `WM_SIZE` of the parent instead, but it is not required.

## Impact on Phase 0

With 0.1 (X11) and 0.2 (Win32) both green, the "external process + window
embedding" premise holds on both target platforms. Remaining for 0.4: weigh
the documented gaps (KDE Wayland untested, wfreerdp-on-Windows untested)
and present the go/no-go evidence.
