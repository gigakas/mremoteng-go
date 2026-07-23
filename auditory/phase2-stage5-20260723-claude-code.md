# Audit — Phase 2, Stage 2.5

- **Date (UTC)**: 2026-07-23
- **Agent**: claude-code
- **Audited stage**: RDP — external process only
- **Commits covered**: uncommitted at audit time; see the stage 2.5
  implementation and closing-audit changelog fragments created alongside
  this file

## 1. Code quality

- `internal/protocol/rdp/rdp.go`: platform-neutral `Session` (lifecycle,
  CLI arg building) with a `connectPlatform` seam implemented per-OS in
  `embed_windows.go`/`embed_linux.go` — the same shape as the deleted
  Phase 0 spike's `internal/spike/reparent/{win32,x11}.go` split, rebuilt
  from the documented findings since that code was deleted at Phase 0's
  close ("spike code deleted per charter" — stage 0.4's audit).
- `embed_windows.go`: the highest-risk code in this stage —
  hand-written LazyDLL bindings to seven `user32.dll` functions not
  wrapped by `golang.org/x/sys/windows`
  (SetParent/GetWindowLongPtrW/SetWindowLongPtrW/SetWindowPos/
  GetAncestor/SetThreadDpiHostingBehavior/SetProcessDpiAwarenessContext).
  Every numeric constant that matters (`GWL_STYLE=-16` as its two's-
  complement `uintptr` form, `DPI_HOSTING_BEHAVIOR_MIXED=1`,
  `DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2=((void*)-4)`, `WS_*`/`SWP_*`
  flags) is taken directly from the spike's own hard-won documentation
  (`docs/spike-win32.md`, which explicitly records that getting the DPI
  hosting-behavior enum wrong — passing 2 instead of 1 — cost that spike
  "several rounds"), not re-derived from scratch.
- **`EmbedChild`'s doc comment is explicit about a real limitation**: it
  cannot fix a parent window created without `DPI_HOSTING_BEHAVIOR_MIXED`
  already set on the thread that created it, because that behavior "is
  captured per-window at creation time" (spike finding). This is
  correctly the caller's (Phase 3's) responsibility, not something
  `EmbedChild` can paper over after the fact — documented rather than
  silently assumed.
- Struct layout risk was handled by verification, not memory: before
  writing `wndClassExW` (the Go mirror of `WNDCLASSEXW`, needed only for
  the *test*, to create a throwaway parent window), a small throwaway C
  program was compiled with the session's mingw toolchain to print
  `sizeof`/`offsetof` for every field directly from the real Windows SDK
  headers. The Go struct's field order/types were chosen to match those
  offsets exactly (confirmed: 80 bytes total, matching offsets
  0/4/8/16/20/24/32/40/48/56/64/72).
- No duplication, no function over ~50 lines, no discarded errors.

## 2. Performance

Not applicable: one external process per session, `findAndAdopt` polls
every 200ms up to a 15s deadline (matches the spike's own retry-loop
guidance for sdl-freerdp's window-recreation-during-init behavior) — a
bounded, infrequent poll, not a hot path.

## 3. Architecture

- Implements `protocol.WindowProtocol` (introduced in stage 2.3), the
  intended reuse this session predicted when adding that interface: "RDP,
  AnyDesk — later stages" was literally the parenthetical in
  `window.go`'s doc comment written during 2.3.
- **Genuine, non-trivial integration test, not just construction stubs**:
  `TestEmbedChild_ReparentsARealExternalWindow` performs the *entire*
  validated recipe against a real OS-level window belonging to a genuinely
  separate process, and would fail if the DPI-awareness handling were
  wrong (that's exactly the failure mode the spike documented and this
  test's assertion — via `EmbedChild`'s own `GetAncestor` check — is
  built to catch). This is the same category of value as the Terminate()
  bug caught in stage 2.3: real testing against real OS primitives, not
  assumptions.
- **Test-methodology finding, worth recording**: `notepad.exe` is not a
  usable "launch and find its window" target on this Windows 11 install —
  it is MSIX-packaged, and the process actually reachable via
  `exec.Command("notepad.exe").Start()` exits/redirects, so its PID never
  owns the visible window (confirmed with
  `Get-Process | Select MainWindowHandle` before writing the test, not
  assumed). `mspaint.exe` was checked the same way and is not redirected,
  and is what the tests actually use. This matters beyond this stage:
  anything in a later phase that reasons about "launch app X, find its
  window by PID" needs to avoid inbox apps Microsoft has migrated to
  MSIX/UWP packaging.
- **Cross-compilation observation for Phase 4**: `GOOS=linux GOARCH=amd64
  CGO_ENABLED=0 go build ./...` fails, but **not because of this stage** —
  `internal/protocol/rdp` alone cross-compiles cleanly for Linux with no
  cgo. The failure is `internal/protocol/web` (stage 2.3) being
  unconditionally blank-imported into `cmd/mremoteng`: WebKitGTK support
  on Linux is *also* cgo, by the blueprint's own design, so
  `CGO_ENABLED=0` was never going to work for the full binary on any
  platform once 2.3 landed — this is expected, not a regression, but
  worth flagging now for whoever designs Phase 4's cross-compilation
  matrix: it will need a real Linux C toolchain + WebKitGTK dev headers
  in CI, or build-tag-gated optional backends, not `CGO_ENABLED=0`.
- No new dependencies (pure `golang.org/x/sys/windows` + hand-bound
  `user32.dll` calls on Windows; plain `os/exec` on Linux).
- No impact on closed Phase 1 packages or on other stage 2.x backends'
  contracts.

## 4. Evidence

- `./scripts/check.sh`: **green**.
- `./scripts/smoke.sh`: **green** (binary builds with `rdp` now
  blank-imported).
- `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./internal/protocol/rdp/...`:
  **green** (this package's own Linux cross-compile; see the architecture
  note above for why the *whole-module* cross-compile fails for an
  unrelated, pre-existing reason).
- New tests: `internal/protocol/rdp/rdp_test.go` (platform-neutral —
  `New` validation, `WindowProtocol` conformance) plus, Windows-only via
  `//go:build windows`, `embed_windows_test.go`:
  `TestFindTopLevelForPID_ExternalProcess` (find-by-PID against a real
  external process) and `TestEmbedChild_ReparentsARealExternalWindow`
  (the full SetParent/DPI/restyle/verify recipe against a real external
  window) — both genuine integration tests against real OS state, not
  stubs. The platform-neutral tests were an oversight caught while
  writing this audit (the first draft only had the Windows-specific
  ones) and added before closing rather than left as a pending action.
  No tests exist for the Linux path: nothing to test yet, since window
  discovery isn't implemented there (see pending actions).
- `go test -race`: still not run module-wide (pre-existing gap).

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- **Linux window discovery/embedding is not implemented** —
  `NativeWindowHandle()` always returns 0 on Linux. Needs
  `github.com/BurntSushi/xgb` (per `docs/spike-x11.md` and the deleted
  spike's own dependency choice) and a real X11 environment to validate
  against, neither available in this session.
- **`/cert:ignore` accepts any RDP host certificate unconditionally** — no
  trust UI exists yet (Phase 3). Same category of gap as stage 2.2's SSH
  `InsecureIgnoreHostKey`.
- **`/p:<password>` on the FreeRDP command line is visible to other local
  processes** via the process list for the process's lifetime — accepted
  as v1 scope per the blueprint's own wording ("CLI flags and .rdp
  files"), but a temp `.rdp` file (as the spike's `mstsc` fallback used)
  would avoid this specific exposure; noted as a v2 candidate.
- **Cross-compilation of the full binary needs a real plan** (see the
  architecture section) — relevant to whoever picks up Phase 4.1.
- Commit the working tree — not done without explicit request.
