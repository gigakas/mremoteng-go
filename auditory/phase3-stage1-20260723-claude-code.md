# Audit — Phase 3, Stage 3.1

- **Date (UTC)**: 2026-07-23
- **Agent**: claude-code
- **Audited stage**: Application shell (window, menu, layout)
- **Commits covered**: uncommitted at audit time; see the stage 3.1
  implementation and closing-audit changelog fragments created alongside
  this file

## 1. Code quality

- `internal/ui/shell.go`: `Shell` struct wraps `fyne.App`/`fyne.Window`
  plus two exported mutator methods (`SetTree`/`SetTabs`) that stages 3.2
  and 3.3 will call to replace the current placeholder
  `*widget.Label`s — designed so those stages don't need to touch
  `shell.go` at all, just call the setters from their own package.
- Menu (`buildMenu`): File (New Connection / New Connections File / Quit),
  View (Connections), Help (About) — a minimal, honest v1 set; items whose
  real behavior depends on later stages (New Connection needs 3.2/3.4, the
  View toggle needs 3.2) have empty `func(){}` actions with a comment
  pointing at the stage that wires them, rather than a `TODO` with no
  further information.
- No duplication, no function over ~50 lines, no discarded errors.

## 2. Performance

Not applicable: one window, built once at startup.

## 3. Architecture

- New package `internal/ui`, matching the blueprint's declared ownership
  for this phase.
- New dependency: `fyne.io/fyne/v2` (and its substantial transitive
  closure — GL/GLFW bindings, font shaping, SVG, image codecs; all
  standard for a cross-platform native GUI toolkit, and this is the
  toolkit the blueprint names explicitly for Phase 3, so no alternative
  was evaluated). Requires cgo, same category of environment requirement
  stage 2.3's webview backend already introduced — this session's
  portable mingw toolchain (fetched for 2.3, still on `PATH`) covers it
  again; still session-local, not a durable environment fixture (repeating
  the note from 2.3's audit since it applies again here).
- `cmd/mremoteng/main.go` now launches the real application
  (`app.NewWithID` + `ui.NewShell` + `Window.ShowAndRun`) instead of
  printing a skeleton message — this is the actual, intended content of
  stage 3.1 ("Application shell"), not scope creep.
- **`scripts/smoke.sh` updated**, necessarily: it used to capture the
  binary's stdout and grep for a marker string, which made sense for a CLI
  skeleton but not for a GUI app whose `ShowAndRun()` blocks until the
  window closes and prints nothing to a terminal. Replaced with the same
  launch-wait-verify-alive-then-kill technique used throughout this
  session to validate GUI processes without being able to see them (`kill
  -0` to check the process didn't crash within 2s of starting, then
  terminate it). This is a project-tooling change, not owned by any single
  phase, but directly necessitated by this stage's own deliverable — the
  app fundamentally changed from "prints and exits" to "runs until
  closed."
- **`AppID` constant** (`go.mremoteng.mremoteng`) is set now, ahead of
  actually needing it, specifically because stage 3.5 (settings
  persistence) will need `Preferences()`, which Fyne silently disables
  without a unique app ID (confirmed directly: the very first probe run in
  this session, before `NewWithID` was used, logged exactly this error).
- No impact on `internal/protocol` or `internal/connection`; this stage
  doesn't touch either yet (3.2/3.3 will).

## 4. Evidence — and an important limitation

- `./scripts/check.sh`: **green**.
- `./scripts/smoke.sh`: **green**, after the necessary update described
  above; confirms `mremoteng` starts and stays running for at least 2s
  without crashing.
- **No visual verification was possible or attempted.** Before writing
  any stage 3.1 code, a disposable Fyne probe was built and run to check
  whether this development environment could show and screenshot a real
  window. It could not: the probe's window is real by every Win32
  measure available — `Get-Process` reports a valid `MainWindowHandle`,
  `IsWindowVisible` returns true, `GetWindowRect` reports plausible
  on-screen coordinates within the primary screen's bounds, and the
  process runs in the same Windows `SessionId` as the interactive
  session driving this tool — but the window never appears in a
  screenshot taken via `Graphics.CopyFromScreen` from the same session,
  with or without `FYNE_RENDERER=software`. The cause was not fully
  diagnosed (candidate explanation: a desktop-object association
  mismatch between how this session's shell launches child processes and
  the interactive desktop being captured, but this is a guess, not a
  confirmed root cause). This was raised with the user, who chose to
  proceed on headless-tests-only verification for the rest of this phase
  rather than pause to chase the screenshot issue further — recorded here
  so the decision and its basis are traceable.
- New tests (`internal/ui/shell_test.go`, 4 tests), using
  `fyne.io/fyne/v2/test` (Fyne's own headless driver, rendering to an
  in-memory software canvas — not a screenshot substitute, a genuinely
  different and standard verification method Fyne itself uses for CI):
  menu structure and placeholder types (`TestNewShell_...`), the Quit
  item's `IsQuit`/`Action` wiring, and both `SetTree`/`SetTabs` mutators.
  These verify structure and wiring, not visual appearance — stated
  plainly rather than implied to be more than they are.

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- **No visual confirmation this stage looks acceptable** — layout,
  spacing, menu ordering, and general appearance are all unverified
  beyond "Fyne's headless renderer doesn't error building this widget
  tree." The user should run the binary themselves and look at it
  before this is considered visually final; this applies to every
  remaining Phase 3 UI stage, not just this one.
- **`fyne.Do` threading model**: a Fyne-internal warning
  ("This application has not been migrated to the fyne.Do threading
  model") appears in `smoke.sh`'s output. Not traced to any code in this
  stage (no goroutines touch UI state here), but worth watching as later
  stages add background work (protocol connections, credential fetches)
  that *will* need to marshal UI updates through `fyne.Do` correctly.
- **The mingw C toolchain is session-local**, repeating the note from
  stage 2.3 — now load-bearing for the main binary itself, not just one
  backend.
- Commit the working tree — not done without explicit request.
