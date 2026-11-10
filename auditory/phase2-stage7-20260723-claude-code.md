# Audit — Phase 2, Stage 2.7

- **Date (UTC)**: 2026-07-23
- **Agent**: claude-code
- **Audited stage**: AnyDesk (external process)
- **Commits covered**: uncommitted at audit time; see the stage 2.7
  implementation and closing-audit changelog fragments created alongside
  this file

## 1. Code quality

- `internal/protocol/anydesk/anydesk.go` + `embed_windows.go` +
  `embed_linux.go`: structurally a near-mirror of stage 2.5's RDP backend
  (external process, `protocol.WindowProtocol`, same lifecycle shape),
  which is appropriate — the blueprint describes this stage as literally
  "Same external-process + reparent pattern as RDP."
- **Refactor extracted before duplicating, not after**: rather than
  copy-pasting stage 2.5's find-and-adopt/DPI-aware-`SetParent` code a
  second time, it was pulled out into a new shared package,
  `internal/protocol/winembed`, in this same session, and RDP's own code
  was updated to use it too (verified: RDP's tests still pass unchanged
  after the extraction). This was anticipated, not improvised: the Phase 0
  spike's own notes said "The same loop re-embeds when a client re-creates
  its window later (AnyDesk prep, stage 2.7)" — this stage is exactly that
  prediction landing.
- **Explicit, load-bearing limitation stated up front**: the package doc
  comment says plainly that AnyDesk itself was not installed or fetched in
  this environment, and that the client-launch/CLI-argument code follows
  documented behavior but is unverified against a real binary. This is a
  different category of gap than, say, stage 2.5's Linux window-discovery
  gap (an environment limitation) — this one is a deliberate scope
  boundary chosen for a specific reason (see Architecture).
- No duplication, no function over ~50 lines, no discarded errors.

## 2. Performance

Not applicable: same shape as RDP, one external process per session.

## 3. Architecture

- Implements `protocol.WindowProtocol` (stage 2.3), same as RDP.
- **Deliberately did not download and run AnyDesk**, despite AnyDesk
  offering a portable, no-installer build the same way mingw-w64 does
  (which *was* fetched and used for stage 2.3). The distinction: mingw is
  an open-source, well-understood, side-effect-free build tool; AnyDesk is
  proprietary live remote-access software that creates a machine
  identity/ID and has its own network/telemetry behavior. Running it
  unattended in an automated session was judged to be a meaningfully
  different, higher-stakes kind of action than fetching a compiler, and
  was not done without the user's awareness — recorded here as a
  reasoned scope boundary, not an oversight.
- `buildArgs` sends the password via the client's stdin
  (`--with-password`, per AnyDesk's documented CLI) rather than as a
  command-line argument — notably *better* than RDP's own `/p:<password>`
  approach (stage 2.5's pending action about process-list password
  exposure), because AnyDesk's documented interface happens to support
  it. This is inherited/documented behavior, not independently verified.
- Linux: window discovery/embedding is not implemented, same environment
  gap as RDP's Linux path (no X server here to validate against) — with
  one added, genuine (not just environmental) gap: AnyDesk has no
  documented `/parent-window`-equivalent launch flag the way xfreerdp
  does, so even a fully resourced future attempt would need the *generic*
  reparent-after-launch approach, which `docs/spike-x11.md` explicitly
  left "unfinished on purpose" during the spike. Worth flagging clearly
  since it means Linux AnyDesk embedding is harder than Linux RDP
  embedding, not just untested the same way.
- Stays inside `internal/protocol/anydesk/` plus the new, shared
  `internal/protocol/winembed/`; `cmd/mremoteng/main.go` blank-imports it.
  No new dependencies.
- No impact on closed Phase 1 packages or on other stage 2.x backends'
  contracts; RDP's behavior is unchanged by the `winembed` extraction
  (same tests, same results, verified).

## 4. Evidence

- `./scripts/check.sh`: **green**, including `internal/protocol/winembed`
  and `internal/protocol/rdp` both passing after the extraction.
- `./scripts/smoke.sh`: **green** (binary builds with `anydesk` now
  blank-imported).
- `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build
  ./internal/protocol/anydesk/...`: **green**.
- New tests (`internal/protocol/anydesk/anydesk_test.go`, 3 tests):
  missing-address validation, `WindowProtocol` conformance, and —
  honestly, the only thing actually exercisable — `Connect` returning a
  clear "client not found" error when `AnyDesk.exe`/`anydesk` isn't on
  `PATH` (which it isn't, here). No test exercises real client launch or
  window discovery, because there is no real client to launch. This is
  stated as a real limitation, not implied to be equivalent to the
  genuine integration tests stages 2.4/2.5/2.6 have against fake or real
  external processes.
- `internal/protocol/winembed`'s own tests (moved from `rdp` in this
  session) continue to pass, now exercising the mechanism as shared
  infrastructure rather than RDP-private code.

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- **Not verified against a real AnyDesk client** — the CLI argument
  format (`<address> [--with-password]`, password via stdin) and the
  client binary name (`AnyDesk.exe` / `anydesk`) are taken from documented
  behavior, not confirmed by running it. Owner: whoever has AnyDesk
  installed (or is willing to install and test it deliberately, with that
  choice made consciously) should validate this before shipping.
- **Linux embedding needs the harder, generic reparent-after-launch
  approach** the spike left unfinished — not just "needs an X11
  environment to test," but a real, not-yet-designed mechanism (watch for
  window re-creation and re-embed, since there's no `/parent-window`
  equivalent to rely on).
- **`/cert`-equivalent / host-trust question**: AnyDesk has its own
  trust-on-first-use device authorization flow (typically requiring
  interactive confirmation from the *other* side), which this
  implementation doesn't address at all — worth scoping explicitly in a
  follow-up rather than assumed away.
- Commit the working tree — not done without explicit request.

## Phase 2 status

All seven stages are now closed (2.1-2.7). Task #5 (wrap-up: exit
criteria check, possible README update) is the next and final step for
this phase — not done as part of this stage's closure, since the
blueprint's exit criteria explicitly call for a broader check ("a demo
config file connects successfully over SSH, VNC and RDP on both
platforms") that this single-stage audit shouldn't presume to satisfy on
its own.
