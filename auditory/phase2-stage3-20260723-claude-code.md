# Audit — Phase 2, Stage 2.3

- **Date (UTC)**: 2026-07-23
- **Agent**: claude-code
- **Audited stage**: HTTP/HTTPS (native webview)
- **Commits covered**: uncommitted at audit time; see the stage 2.3
  implementation and closing-audit changelog fragments created alongside
  this file

## 1. Code quality

- `internal/protocol/window.go`: new `WindowProtocol` interface
  (`Protocol` + `NativeWindowHandle() uintptr`), composed like
  `TerminalProtocol`/`FramebufferProtocol` from the previous two stages.
  Deliberately shaped to serve *both* in-process windows (this stage's
  webview, created via cgo) and external-process windows (RDP/AnyDesk,
  stages 2.5/2.7, found via window-search the way Phase 0's spike did) —
  from the eventual Phase 3 UI's perspective, both reduce to "here is a
  native handle, reparent it into a tab." Establishing this now, on the
  first stage that needs it, means 2.5/2.7 reuse it instead of inventing
  their own variant.
- `internal/protocol/web/web.go`: `New` is registered for *both*
  `connection.ProtocolHTTP` and `connection.ProtocolHTTPS` — the only
  difference between the original C# `ProtocolHTTP`/`ProtocolHTTPS`
  classes is the URL scheme, so one constructor branching on
  `values.Protocol` avoids duplicating the session type.
- **Real bug found and fixed via testing, not by inspection alone**: the
  first version of `Disconnect` called `w.Terminate()` directly from the
  caller's goroutine. The first attempt at
  `TestSession_ConnectAndDisconnect_CreatesAndClosesANativeWindow` failed
  — `OnClose` never fired within the test's timeout. Traced it by reading
  the vendored C++ source
  (`libs/webview/include/webview.h`) rather than guessing: the Windows
  backend's `terminate_impl` is `PostQuitMessage(0)`, which posts `WM_QUIT`
  to the *calling* thread's message queue — a Win32-level footgun, since
  `PostQuitMessage` is documented by Microsoft as thread-local, not
  window-targeted. Called from a different goroutine (a different OS
  thread, since the webview's message loop runs on a thread locked via
  `runtime.LockOSThread`), it posts to nobody's queue that matters and is
  silently a no-op. This contradicts webview_go's own Go-level doc comment
  ("safe to call this function from a background thread") — true for the
  library's GTK backend (`terminate_impl` there calls `dispatch_impl`
  internally, which *does* cross threads safely via GLib), false for its
  Windows backend. Fixed by routing `Terminate` through `Dispatch` (which
  uses `PostMessageW` targeted at a specific message-only window — safe
  to call cross-thread by Win32 design) instead of calling it directly.
  Documented inline in both `web.go` and the blueprint's 2.3 stage notes
  so this doesn't get silently reintroduced or rediscovered from scratch.
- `run()`'s goroutine locks the OS thread for the whole webview lifetime
  (`runtime.LockOSThread`/`defer runtime.UnlockOSThread`), required by
  both Win32 and Cocoa (a window's message loop must run on the thread
  that created the window) — GTK doesn't strictly require this but it's
  harmless there.
- No duplication, no function over ~50 lines, no discarded errors.

## 2. Performance

Not applicable: one native window per session, no hot path. `Resize`
dispatches onto the webview's own thread rather than touching Win32/GTK
state directly from an arbitrary caller thread, which is a correctness
requirement here, not a performance one.

## 3. Architecture

- New dependency: `github.com/webview/webview_go` — the blueprint names
  this exact approach explicitly ("OS-native webview... via a thin
  wrapper. This is the only place where cgo is acceptable, and only
  through the wrapper library"), so the dependency itself needs no further
  justification beyond what's already in the blueprint.
- **Environment note, important for reproducibility**: this stage was
  blocked for most of this session on a missing C compiler (see the
  blueprint's 2.3 notes and this session's earlier changelog fragment
  recording the block). Unblocked by downloading a portable, no-installer
  mingw-w64 build to a session-scratch directory — **not committed to the
  repo, not a durable machine change**. Anyone continuing this work needs
  their own C toolchain; this is called out three times now (blueprint,
  this audit, changelog) specifically so it isn't missed.
- `internal/protocol/web/` stays inside its own package, consistent with
  every other stage-2.x backend; `cmd/mremoteng/main.go` blank-imports it.
- No impact on closed Phase 1 packages or on the `Protocol`/
  `TerminalProtocol`/`FramebufferProtocol` contracts from prior stages;
  `WindowProtocol` is purely additive.

## 4. Evidence

- `./scripts/check.sh`: **green** (run with `CGO_ENABLED=1` and the
  session's portable mingw-w64 on `PATH` — this matters for this specific
  package; every other package in the module still builds fine with
  `CGO_ENABLED=0`).
- `./scripts/smoke.sh`: **green** — binary builds (with `web` now
  blank-imported), `mremoteng` starts, changelog reproducible.
- New tests (`internal/protocol/web/web_test.go`, 3 tests): two are
  construction/validation only (missing hostname, both HTTP and HTTPS
  register); the third,
  `TestSession_ConnectAndDisconnect_CreatesAndClosesANativeWindow`, is a
  **real integration test** — it creates an actual OS-native webview
  window (confirmed working via a disposable probe program before writing
  the real test, run under a hard `timeout` as a safety net) and tears it
  down, asserting `NativeWindowHandle()` is non-zero and `OnClose` fires
  after `Disconnect` (this is the test that caught the `Terminate`
  cross-thread bug above). It's bounded by a `context.WithTimeout` so that
  if a future environment lacks an interactive window station, the test
  fails cleanly within the timeout rather than hanging the test binary —
  though a genuine hang inside the C `webview_create` call itself would
  still leak the one goroutine, since `Connect`'s `ctx` has no way to
  abort a stuck native call already in flight (same category of
  limitation as the SSH/VNC handshake goroutines from earlier stages,
  which have the same shape).
- `go test -race`: still not run module-wide (pre-existing environment
  gap), though notably this stage's own bug was a genuine data race in
  spirit (a cross-thread call with no synchronization) that the race
  detector likely would have flagged immediately if it had been
  available — worth prioritizing getting `-race` working in some
  environment before Phase 2 ships, given two stages now (this one, and
  the concurrent code from 2.2/2.4) have real concurrency to get wrong.

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- **The C toolchain used to build/test this stage is session-local**, not
  part of any committed or durable environment state. Flagged repeatedly
  (blueprint, here, changelog) so it isn't lost. Whoever next builds this
  package needs mingw-w64 (Windows) or WebKitGTK dev headers (Linux).
- **Race detector still unavailable** in this environment; this stage is
  a good candidate to re-verify under `-race` first, given it just proved
  out a real cross-thread bug in a dependency.
- **No visible reparenting yet**: `NativeWindowHandle()` exists and is
  tested for non-zero-ness, but nothing actually reparents the window
  into a tab yet — that's Phase 3 UI work, same as the terminal widget
  noted as out of scope in stage 2.2's audit.
- Commit the working tree — not done without explicit request.
