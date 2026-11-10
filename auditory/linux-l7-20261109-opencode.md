# Audit — Linux track, stage L7

- **Date (UTC)**: 2026-11-09
- **Agent**: opencode
- **Audited stage**: L7 — RDP parity (device redirection and display flags)
- **Scope**: `internal/protocol/rdp`, RDP panel options in `internal/ui/session.go`

## 1. Code quality

- Redirection is expressed as `Options` fields and mapped in `Arguments`, which
  stays the single place that builds the command line; the option tests assert
  both the enabled and the default (absent) case, so no flag can leak into a
  plain session.
- Blank drive entries are ignored, so a trailing comma in the UI cannot produce an
  invalid `/drive:` argument.
- The RDP panel keeps these choices in `rdpConfig` and copies them into the next
  Connect/Test, with password, certificate policy and security mode applied on
  top. The options popup is a fixed-size modal, consistent with the other dialogs
  and free of horizontal scrolling.
- Options are intentionally window-session state: persisting them needs schema
  fields and belongs with L9/L10, where per-connection settings are designed.

## 2. Performance

- No change to the hot paths: flags are appended once per launch.

## 3. Architecture

- `rdp.Options` remains the boundary between the UI and FreeRDP; no new
  dependency was introduced.
- Dynamic resolution is opt-in with an explicit warning because it breaks xrdp
  servers (documented field note and backlog risk).
- Multi-monitor is marked "separate window" in the UI: `/multimon` is meaningless
  for an embedded child window, so the label sets the expectation.

## 4. Evidence

- `./scripts/check.sh`: green.
- New tests: `TestArguments_DeviceRedirectionAndDisplayFlags` (flags present when
  enabled, absent by default, blank drive ignored).
- Manual acceptance still required against a real desktop host.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Manual session with clipboard and drive redirection against a real Windows RDP
  host (the xrdp container does not exercise clipboard/redirection well).
- RD Gateway flags and gateway credentials are not wired yet; the model has the
  fields, the launcher does not. Tracked as the remainder of L7 in the backlog.
- Options persistence per connection: decide in L9/L10.
