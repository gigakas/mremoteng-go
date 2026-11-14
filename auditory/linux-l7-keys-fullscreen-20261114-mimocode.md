# Audit — Linux track, Stage L7 (special keys, full screen, multi-monitor)

- **Date (UTC)**: 2026-11-14
- **Agent**: mimocode
- **Audited stage**: L7 — RDP parity, the last items of `blueprint/backlog.md`'s L7
  scope: "send special keys", full screen, and multi-monitor as an explicit option
- **Commits covered**: this commit

## 1. Code quality

**Special keys (`internal/ui/specialkeys.go`, `embed.go`).** A chord cannot be typed
into an embedded session: `Ctrl+Alt+Del` never leaves the local X server, and `Alt+Tab`
is grabbed by the window manager, so the session never sees either. The feature types
them instead:

1. focus the session window (`SetInputFocus`, parent-relative),
2. fake the key events with **XTEST**, modifiers pressed first and released in reverse,
3. sync.

XTEST rather than `XSendEvent` because most toolkits ignore synthetic events marked
`send_event`, and the whole point is a chord the desktop would otherwise swallow.
Keycodes come from the **X keyboard mapping** (`keycodeForKeysym`), never from a
hardcoded US table, so the feature works on any layout. The chord table is one list
(`specialChords`) shared by the session-bar button and the View > Send keys submenu.

The bar carries **one** icon-only button (it sends `Ctrl+Alt+Del`, the chord users
actually need) and the rest live in the menu: the session bar already overflowed once
when a text label was added to it, and five chord labels would push the session into
horizontal scrolling. The button is documented on the start page's **Buttons** card.

`embeddedArea` gained `SendKeys`, so the panel asks its area rather than reaching into
X itself; the fake area records what it was asked to send, which is how the headless
tests assert the chord without an X server.

**Full screen.** `View > Full screen` toggles `Window.SetFullScreen` and reports the
state, and the menu item carries a check mark — without a labelled way back, a
full-screen session on a desktop that hides its own window controls would be a trap.
The embedded session follows through the existing 250 ms geometry poll, so no new
resize path was added.

**Multi-monitor was broken, not pending.** `/multimon` was passed even when the session
was embedded, where it means nothing: an embedded window is clamped to the tab, which is
the opposite of spanning displays. The flag now suppresses embedding, so FreeRDP opens
its own window and the option does what its label says ("All monitors (separate
window)"), and the status line reports the session as a separate window.

## 2. Performance

One X round trip per chord plus one before it for the keyboard mapping, only when a
button is pressed. Full screen costs a window property change, and the embed poll was
already running.

## 3. Architecture

No new dependency: `github.com/BurntSushi/xgb` was already required for embedding and
its `xtest` subpackage ships in the same module. Sending keys stays inside
`internal/ui` — FreeRDP is never linked, and the feature uses the same X11 boundary the
embedder already documents.

**Limits, stated rather than hidden:**

- A chord needs the session **embedded**. A session in its own FreeRDP window has no
  window this package can focus without searching the window tree by PID, and the
  panel says so instead of failing silently.
- The session window must be **mapped**: a chord sent while its tab is in the
  background unmaps nothing but focuses a hidden window, so the remote side may ignore
  it. The button lives on the bar of the selected tab, which is the visible case.
- Multi-monitor still has no automated end-to-end check: it needs a second monitor. The
  unit test pins the decision that matters (no `/parent-window`), not the visual result.

## 4. Evidence

- `./scripts/check.sh`: OK.
- XTEST and the keysym mapping probed on this machine's display (`DISPLAY=:0`), with a
  temporary test that was removed afterwards:

```
PROBE XTEST ok, keys per keycode 10, range 8-255
Ctrl+Alt+Del     Control_L 0xffe3 -> 37, Alt_L 0xffe9 -> 64, Delete 0xffff -> 119
Ctrl+Alt+End     End 0xff57 -> 115
Ctrl+Alt+Insert  Insert 0xff63 -> 118
Alt+Tab          Tab 0xff09 -> 23
Alt+Shift+Tab    Shift_L 0xffe1 -> 50
```

  Every keysym of every chord resolves on this keyboard, so the feature is not a
  US-layout assumption.
- New tests: `TestSpecialChords_PutModifiersFirstAndEndWithTheKey`,
  `TestSendSpecialKey_TypesTheChordIntoTheSession`,
  `TestSendSpecialKey_WithoutAnEmbeddedSession_ExplainsWhy`,
  `TestSendSpecialKey_ClientFailure_IsReported`,
  `TestSpecialKeysButton_IsOnTheRDPBar`, `TestSendChordToSelected_ActsOnTheOpenTab`,
  `TestToggleFullScreen_TogglesTheWindowAndReportsIt`,
  `TestFullScreenItem_IsInTheViewMenu`,
  `TestConnectRDP_MultiMonitor_DoesNotEmbed`,
  `TestKeycodeForKeysym_FindsTheKeyInTheMapping`.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Human, with the session embedded: press the special-keys button and confirm the
  Windows/Linux login screen of the remote host reacts, then `View > Full screen` and
  back.
- Multi-monitor needs a machine with two screens: confirm FreeRDP opens its own window
  spanning them and that `View > Full screen` does not fight it.
- `Alt+Tab` and friends are only delivered while the session window is mapped; a hidden
  tab needs the button pressed after selecting it.
