# Spike 0.2 findings — Win32 embedding of an external RDP client (Windows)

- **Date**: 2026-07-16 (supersedes the same-day first version of this doc)
- **Agent**: claude-code · **Validated by**: human, on a Windows 11 VM
  (VMware) connecting to the Linux host's xrdp container.
- **Verdict: the approach works.** `sdl-freerdp.exe` (FreeRDP nightly)
  embedded in a Fyne window via `SetParent`, interactive session (apps
  opened inside the remote desktop), client-side scaling on resize.
  Notepad used as the control target for the mechanism.

## Validation matrix

| Client | Result |
|---|---|
| Notepad (control) | ✅ embeds, re-embeds on reconnect |
| `sdl-freerdp.exe` (FreeRDP nightly) | ✅ embeds; needs find-and-adopt retry (window re-created during renderer init) |
| `mstsc.exe` | ⚠️ unreliable — embedded on first attempt, subsequent attempts refused (`SetParent` returns NULL, no error, no effect) |

## The hard-won knowledge (Phase 2.5 field manual)

1. **DPI awareness is the gatekeeper of cross-process `SetParent` on
   Win10+.** If parent and child have different `DPI_AWARENESS_CONTEXT`s
   (even per-monitor **v1 vs v2**), `SetParent` refuses **silently**
   (returns NULL, `GetLastError()` = 0, window stays top-level). Required
   recipe, all pieces mandatory:
   - `SetProcessDpiAwarenessContext(PER_MONITOR_AWARE_V2)` at startup;
   - `SetThreadDpiHostingBehavior(MIXED)` on the **main thread before the
     Fyne window is created** — the hosting behavior is captured per-window
     at creation time; setting it later does nothing;
   - the enum is `INVALID=-1, DEFAULT=0, MIXED=1` — passing 2 fails
     silently (returns -1) and cost this spike several rounds;
   - **verify with `GetAncestor(child, GA_PARENT)` after `SetParent`** —
     never trust the return value alone.
2. **`SetParent` first, restyle after** (strip
   `WS_POPUP|WS_CAPTION|WS_THICKFRAME`, add `WS_CHILD`, then
   `SetWindowPos(FRAMECHANGED)`) — the original mRemoteNG order for PuTTY.
3. **Find-and-adopt must be a retry loop.** sdl-freerdp creates a
   provisional `SDL_app` window and re-creates it during Direct3D renderer
   init; the first handle dies between discovery and adoption
   (`SetParent` → `ERROR_INVALID_PARAMETER`). Retrying the whole
   find→adopt cycle until a deadline fixes it. The same loop re-embeds
   when a client re-creates its window later (AnyDesk prep, stage 2.7).
4. **Skip `#32770` dialogs when hunting the session window by PID** —
   credential/trust prompts are visible top-levels of the same process.
5. **mstsc is not a dependable embedding target.** It embedded once, then
   refused (NULL return, no error, no effect — the silent-refusal
   signature). Prime suspect: its manifest sets `uiAccess=true`, so UIPI
   can block adoption by normal-integrity processes. Not worth chasing:
   the original mRemoteNG never embeds mstsc either (it uses the RDP
   ActiveX control). mstsc quirks kept for reference: args via a temp
   `.rdp` file (`smart sizing:i:1`), no CLI password.
6. **Resize strategies** (`-resize` flag in the spike):
   - `smart` (`/smart-sizing`): client-side scaling, works everywhere,
     blurry at non-native sizes (visible when maximizing an app inside
     the session) — the universal fallback;
   - `dynamic` (`/dynamic-resolution`): native quality, but **drops the
     connection against the xrdp test container on both platforms**
     (Linux/xfreerdp and Windows/sdl-freerdp) — server-dependent, per-host
     opt-in in 2.5, retest against real Windows RDP hosts.

## Packaging note (Phase 4.4)

FreeRDP **no longer ships `wfreerdp.exe`**: GitHub releases are
source-only, and the nightly CI publishes `sdl-freerdp.exe` (SDL client,
window class `SDL_app`) plus proxy/server/tools. Phase 4.4 must pin/mirror
the nightly or build FreeRDP in our CI, and Phase 2.5's Windows client is
sdl-freerdp via generic reparenting (no `/parent-window`: SDL creates its
own window).

## Impact on Phase 0

Both platforms validated with a real FreeRDP client end-to-end. Remaining
gaps for 0.4 to weigh: KDE Wayland untested (0.3), mstsc unreliability
(fallback only), dynamic resolution untested against a real Windows RDP
server.
