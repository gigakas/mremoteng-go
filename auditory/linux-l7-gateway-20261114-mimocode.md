# Audit — Linux track, Stage L7 (RD Gateway re-review)

- **Date (UTC)**: 2026-11-14
- **Agent**: mimocode
- **Audited stage**: L7 — RDP parity, RD Gateway part (`blueprint/backlog.md`)
- **Commits covered**: this commit (rewrites the gateway argument builder and adds the gateway section of the RDP options dialog). The earlier L7 audit `auditory/linux-l7-20261109-opencode.md` stays as written; it described the redirection flags and the options popup.

## 1. Code quality

**The defect this re-review fixes.** `internal/protocol/rdp/rdp.go` emitted the
gateway as three separate flags (`/g:`, `/gu:`, `/gd:`). The installed client,
FreeRDP 3.32.2, rejects all three:

```
$ xfreerdp3 /v:127.0.0.1:13389 /u:ubuntu /from-stdin:force /cert:ignore \
      +auth-only /g:127.0.0.1:1 /gu:gu /gd:GD
[ERROR][com.winpr.commandline] - [CommandLineParseArgumentsA]: Failed at index 6:
Unexpected keyword
```

so every gateway connection failed before the first packet, with the client's
usage banner as the only output. The gateway arguments had a unit test
(`TestArguments_Gateway_AddsHostAndCredentials`) that asserted the *old* syntax,
which is why the rot went unnoticed: the test pinned the code, not the client.

**What replaced it.** `Gateway` is now `{Host, Port, UsageMethod}` and renders one
argument, `/gateway:g:<host>[:<port>][,usage-method:direct|detect]`, through
`gatewayArgument` (`rdp.go`). `Gateway.Problem()` validates host and port so an
argument that would break the client's comma-separated parser never reaches
`argv`; the client accepts `:`, `,` or a space in the host by ending the
argument early, and a wrong port range makes it fail at parse time.
`gatewayUsageMethod` keeps only the two documented values.

**Gateway credentials are deliberately not supported.** The installed client
rejects a gateway user or domain unless a gateway password follows it in the same
argument (`/gateway:g:h,u:user` → usage banner; `/gateway:g:h,u:user,p:pass` →
accepted). Passing that password means argv, which `AGENTS.md` and the project
rules forbid — argv is readable by every process on the machine and this
application already keeps the RDP password on stdin for the same reason. The
options dialog therefore has **no** gateway password field and the note says so.
Consequence, stated plainly: a gateway that requires its own credentials asks
FreeRDP to prompt; if the prompt has nowhere to go the session fails and reports
the client's line (see the pending action below).

**New code size and coverage.** `rdp.go` grows by ~70 lines (gateway argument,
validation, usage-method filter); `internal/ui/session.go` gained
`rdpGatewayForm` (fields, `gateway()`, label/value helpers, `gatewaySummary`) and
`requestRDPOptions` was split before it crossed the 50-line rule — the gateway
section builds its own box, so the dialog stays a list of widgets. Tests:
`TestArguments_Gateway_UsesTheCombinedForm`,
`TestArguments_Gateway_CarriesPortAndUsageMethod`,
`TestGatewayArgument_RejectsWhatWouldBreakTheArgument`,
`TestGatewayUsageMethod_KeepsOnlyTheDocumentedValues` (protocol) and
`TestRDPGatewayForm_Fields_BuildTheGateway`,
`TestRDPGatewayForm_EmptyPort_KeepsTheClientDefault`,
`TestRDPGatewayForm_InvalidInput_IsRefusedWithAReason` (UI).

**Options are still per tab.** `p.rdpConfig` holds the gateway for the next
Connect in that tab, like the redirection flags. Persisting them in the profile
store is the other half of L7 and is not part of this change.

## 2. Performance

Not applicable: the change builds one extra string per Connect. No new
allocations on a hot path, no I/O.

## 3. Architecture

No new dependency. The change stays inside `internal/protocol/rdp` (argument
builder) and `internal/ui` (dialog). The gateway is the one place where the
protocol package validates its own input instead of trusting the caller, because
the client turns a malformed host into a parse failure that looks like "FreeRDP
is broken"; the UI also validates, so the user gets the reason while the dialog
is still open instead of a status line behind a closed window.

Server compatibility, measured with the installed client (FreeRDP 3.32.2):
`/gateway:g:<host>:<port>` is accepted and the client resolves and dials the
gateway; `usage-method:direct` forces the gateway hop; `usage-method:detect`
resolves the target first and skips the gateway when it is reachable, which is
the documented meaning. The separate `/g:`, `/gu:`, `/gd:` flags are rejected.
`/gateway:...,p:<password>` is accepted, i.e. the client *can* take the password
in argv; this application does not.

## 4. Evidence

- `./scripts/check.sh`: OK (gofmt, go vet, all tests).
- `./scripts/smoke.sh`: OK.
- Real-client check of the exact arguments the application builds (generated with
  a temporary probe test, since removed):

```
/v:desk.example.com:3389 /u:user /from-stdin:force /cert:tofu /gateway:g:gw.example.com:443
  → getaddrinfo(gw.example.com) failed: No address associated with hostname
/v:... /gateway:g:gw.example.com:8443,usage-method:direct
  → getaddrinfo(gw.example.com) failed
/v:... /gateway:g:gw.example.com:443,usage-method:detect
  → getaddrinfo(desk.example.com) failed   (detect asks the target first)
```

No usage banner and no `Unexpected keyword` in any of the three, and the first
two dial the gateway rather than the target: the arguments parse and carry the
intended meaning. A full gateway session was **not** verified — no RD Gateway is
reachable from this machine, and xrdp (the local container) does not implement
RD Gateway. That limit is the pending action below, not a claim of parity.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Human/agent with a reachable RD Gateway: one manual session with
  `usage-method:direct`, recording the gateway host, the certificate policy and
  whether FreeRDP could ask for the gateway credentials when they differ from the
  session's. Until then the gateway path is proven at the argument level only.
- L7 still owes *options persistence* (redirection flags and the gateway stored
  per profile instead of per tab) and a readable failure report for immediate
  RDP exits; both are separate increments.
