# Audit — Phase 3, Stage 3.3

- **Date (UTC)**: 2026-07-23
- **Agent**: claude-code
- **Audited stage**: Session tabs hosting protocol views
- **Commits covered**: uncommitted at audit time; see the stage 3.3
  implementation and closing-audit changelog fragments created alongside
  this file

## 1. Code quality

This stage is the largest and most heterogeneous of the phase — it has to
host three structurally different kinds of session (byte-stream terminal,
pushed-framebuffer image, reparented native window) plus the dispatcher
that ties them to `internal/protocol`'s three composed interfaces. Broken
down by piece:

- **`internal/ui/ansi.go` — `ansiState`**: a real, scoped ANSI/VT100
  interpreter over `widget.TextGrid` (which Fyne's own doc comment names
  as intended for "a text editor, code preview or terminal emulator").
  Covers cursor movement (CUU/CUD/CUF/CUB/CUP), erase in display/line
  (ED/EL), basic SGR (reset, bold, 16-color fg/bg), CR/LF/backspace/tab,
  and safely discards OSC payloads (window-title sequences) instead of
  leaking their bytes into the display. Explicitly does not attempt
  256-color/truecolor, the alternate screen buffer, mouse reporting,
  scroll regions, or save/restore cursor — this is the stage 2.2/3.3 "real
  cost driver" the blueprint asks to estimate separately, scoped down
  deliberately rather than either attempting a full xterm clone or falling
  back to external PuTTY processes (which would have left stage 2.2's
  tested native Go protocol implementations unused for the UI path). This
  is the single best-tested piece in the whole phase: 12 tests
  (`ansi_test.go`) feed exact byte sequences and assert exact grid-cell
  contents — pure logic, no rendering involved, so headless testing here
  is not a compromise the way it is for the rest of this phase.
- **`internal/ui/terminal.go` — `Terminal`**: wraps `ansiState` as a
  focusable, typeable Fyne widget. Keyboard mapping
  (`terminalKeySequences`) covers Enter/Backspace/Tab/Escape/arrows;
  unmapped keys are silently ignored (documented, not a bug).
- **`internal/ui/framebuffer.go` — `FramebufferView`**: renders a
  `protocol.FramebufferProtocol`'s `Frames()` channel via `canvas.Image`
  and forwards pointer/keyboard input back through `SendPointer`/
  `SendKey`. Two things worth calling out:
  - **X11 keysym mapping needed no lookup table for printable
    characters** — X11's `keysymdef.h` convention makes printable
    ASCII/Latin-1 keysyms numerically identical to the character code, so
    `TypedRune` just casts `uint32(r)`. Only the non-printable keys
    (`framebufferKeysyms`) needed an explicit table.
  - **v1 simplification, stated in the type's own doc comment**: the image
    renders at native resolution (`canvas.ImageFillOriginal`), not scaled
    to fit the tab, specifically so widget-space pointer coordinates equal
    framebuffer pixel coordinates with no scale-factor math — avoiding a
    real class of off-by-scale bugs at the cost of not fitting arbitrary
    window sizes. A defensible v1 trade, not an oversight.
- **`internal/protocol/winembed` gains `SetWindowPosition`**: the one
  addition to an already-closed Phase 2 package this stage needed — moving
  an *already-embedded* child window as its host widget's on-screen
  geometry changes is a distinct operation from `EmbedChild`'s one-time
  placement, and belongs in the same package as the mechanism it extends
  rather than duplicated in `internal/ui`.
- **`internal/ui/nativewindow*.go` — `NativeWindowHost`**: platform-split
  the same way `internal/protocol/rdp` is (`nativewindow_windows.go` /
  `nativewindow_other.go`). This is `winembed.EmbedChild`'s **first
  production use with a real Fyne-owned window** rather than a hand-built
  test window — stage 2.5/2.7's own tests already validated the mechanism
  itself; what's new here is wiring it to `driver.NativeWindow`/
  `driver.WindowsWindowContext` to get a genuine Fyne window's HWND.
  **A precisely-stated, not-glossed-over limitation**: `Move`'s position
  is relative to the widget's immediate parent container, not necessarily
  the window's absolute client-area origin for deeply nested layouts —
  correct for a host placed directly in window content, unverified beyond
  that, and Fyne has no simple public API to query cumulative container
  offsets. Said plainly in the doc comment rather than assumed correct.
- **`internal/ui/sessiontabs.go` — `SessionTabs`**: the assembly point.
  `Open` builds the right view via a type switch on
  `TerminalProtocol`/`FramebufferProtocol`/`WindowProtocol` (exactly the
  three-way split `internal/protocol` established for this purpose across
  stages 2.1/2.2/2.4), shows the tab immediately, connects in the
  background with a 30s timeout, and wires the connect-dependent parts
  (starting the terminal's read pump; attaching the framebuffer; embedding
  the native window, now that `NativeWindowHandle()` is populated) only
  after `Connect` succeeds. `OnClose` removes the tab automatically.
  Connect failure replaces the tab's content with a visible error message
  rather than leaving a dead, silent tab.

No duplication across the four pieces beyond what's structurally
unavoidable (each view type genuinely needs its own widget); no function
over ~50 lines; no discarded errors — the one place an error is only
logged rather than propagated (`NativeWindowHost.Embed` failure inside
`buildView`'s wire func) is because there's no UI mechanism yet to show
an in-tab error banner post-connect, noted as a pending action rather
than silently swallowed.

## 2. Performance

Not applicable at this stage's scale: one goroutine per open session
(terminal read pump or framebuffer frame relay), each blocked on I/O
between events, not polling.

## 3. Architecture

- All four pieces stay inside `internal/ui/`, this phase's owned package,
  plus the one justified `winembed` addition described above.
- No new dependencies — everything is stdlib, `fyne.io/fyne/v2` (already
  present since stage 3.1), or `golang.org/x/sys/windows` (already present
  since stage 2.5, via `winembed`).
- `cmd/mremoteng/main.go` now creates a real `SessionTabs`, wires it into
  the shell via `SetTabs`, and wires the connection tree's `OnSelect` to
  `protocol.Create` + `SessionTabs.Open` — a connection leaf selected in
  the tree now genuinely attempts to open a session. Nothing populates the
  tree with real connections yet (persistence is stage 3.5), so this is
  currently unreachable in practice, but the wiring itself is real, not a
  stub.
- No impact on already-closed Phase 1/2 packages beyond the additive
  `winembed.SetWindowPosition`.

## 4. Evidence — same visual-verification limitation as 3.1/3.2, now with a
  narrower carve-out

- `./scripts/check.sh`: **green**.
- `./scripts/smoke.sh`: **green** (first run after this stage's changes
  needed several minutes for a fresh cgo/Fyne compile pass, same category
  noted in stage 3.1's audit — not a failure, just slow).
- **No visual verification was possible or attempted**, same phase-wide
  limitation as 3.1/3.2 (see `blueprint/phase-3-ui.md`'s note). This
  applies fully to `Terminal`'s rendered appearance, `FramebufferView`'s
  image display, and — most importantly — `NativeWindowHost`'s actual
  on-screen alignment, which is the piece most likely to need visual
  iteration to get exactly right (see the `Move` limitation in section 1).
- New tests: 12 in `ansi_test.go` (byte-exact grid assertions — the
  strongest coverage in this phase), 5 in `terminal_test.go`, 4 in
  `framebuffer_test.go` (including a real render-a-pushed-frame check via
  `fyne.Do`, confirmed to execute promptly even without a running app
  event loop), 3 in `nativewindow_test.go` (the boundary that's actually
  testable headlessly: `driver.NativeWindow` type-assertion failure —
  confirmed empirically that `test.NewWindow()` does *not* implement it,
  before writing the test, not assumed), 6 in `sessiontabs_test.go`
  (dispatch to all three view types, the unsupported-protocol fallback,
  connect-failure error display, and `OnClose` tab removal). 30 new tests
  total for this stage.
- **What's explicitly not tested**: `NativeWindowHost.Embed`'s actual
  reparenting against a real, live Fyne window's real HWND. Unlike stage
  2.3's webview probe (spun up standalone via `runtime.LockOSThread` +
  `w.Run()`), Fyne's desktop driver was judged impractical to stand up
  from inside a `go test` run without a full `ShowAndRun()` event loop —
  attempting it was considered a real risk of a flaky, platform-fragile
  test for uncertain benefit, given `winembed.EmbedChild` (the mechanism
  `Embed` calls) already has its own genuine integration test from stage
  2.5 against a hand-built Win32 window. What's untested here is
  specifically the last step: wiring that already-proven mechanism to a
  genuine Fyne-owned window instead of a synthetic one.

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- **No visual confirmation of any of the four views** — repeating the
  phase-wide note, with `NativeWindowHost`'s on-screen alignment
  specifically flagged as the piece most likely to need real-display
  iteration (see the `Move` limitation in section 1).
- **`NativeWindowHost.Embed` against a real Fyne window is untested** —
  see section 4's last paragraph. Recommend a manual smoke check (connect
  to a real RDP/AnyDesk/webview target and look at the tab) before relying
  on this in production, and consider whether a live-window test is worth
  the fragility once someone can actually watch it run.
- **`Embed` failures are only printed, not shown in the tab** — a visible
  in-tab error banner is the natural fix, deferred because there's no
  established pattern yet for post-connect error UI in a tab (the
  pre-connect failure path in `Open` does have one, via replacing tab
  content — extending that same idea to post-connect failures is
  reasonable future work).
- **ANSI terminal is a real subset, not xterm** — 256-color/truecolor,
  alternate screen, mouse reporting, and scroll regions are all
  unsupported; programs relying on them (full-screen `vim`/`tmux` in some
  configurations, for instance) will render incorrectly. Documented in
  `ansiState`'s doc comment, not hidden.
- Commit the working tree — not done without explicit request.
