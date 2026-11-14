# Audit — Linux track, Stage L7 (RDP options persistence and exit reporting)

- **Date (UTC)**: 2026-11-14
- **Agent**: mimocode
- **Audited stage**: L7 — RDP parity, the two items left open by
  `auditory/linux-l7-gateway-20261114-mimocode.md`: per-profile option storage and
  a readable report when FreeRDP dies at once
- **Commits covered**: `f742b11` (options in the profile) and the commit holding the
  exit report. The earlier L7 audits stay as written.

## 1. Code quality

**Options move into the profile.** `profiles.Profile` gains
`RDP RDPOptions` (`profiles.go`), a JSON block with the redirection, display,
security and gateway choices. `RDPOptions.Validate` and `Gateway.Validate` refuse
what the launch line cannot spell — a gateway host with a comma, a colon, a space
or a scheme, a port outside 1-65535, an unknown usage method, a shared drive with
a comma or an empty drive — because those values are pasted into one FreeRDP
argument where a comma changes the meaning of the whole line.

**One source of truth.** `sessionPanel.rdpConfig` is gone. Connect, the credentials
test and the options dialog all read `rdpOptionsFromProfile(p.profile)`
(`internal/ui/rdp_options.go`), and the two session-bar checkboxes are initialised
from the profile with a direct field assignment rather than `SetChecked`, which
would fire the handler and write the store every time a tab opens.
`persistRDPToggle` writes one checkbox and puts the box back when the write is
refused, so a tick that cannot survive the next Connect never appears to have
been accepted.

**Write what is stored, not what the panel remembers.** `saveRDPOptions` starts
from `profileByID`, not from the panel's copy: the Config panel can rename or
re-address a connection while its tab is open, and writing the stale copy back
would silently undo that edit. `TestSessionPanel_SaveRDPOptions_StartsFromTheStoredProfile`
covers it.

**Comparison had to change.** `Profile` is no longer comparable (the drives are a
slice), and the Config panel does not edit the RDP block, so
`s.rawForm() != s.baseline` broke twice: it stopped compiling and, had it compiled,
a connection with stored options would have looked permanently modified and sat
behind the discard guard. `Profile.SameFormAs` compares the form-editable fields
and is now what `isDirty` uses; `TestProfile_SameFormAs_IgnoresTheStoredOptions`
pins the intent.

**Exit report.** The RDP watcher used to say "RDP session ended immediately" with a
guess about certificates. It now reduces the client's own log to one line
(`rdp.Summarize`) and turns that line into the next thing to try
(`rdpFailureHint`), in a dialog (`screen.showMessage`, reached through the
`screen.notice` seam so a test can read it). A session that ran for a while still
reports in the status bar, with the cause — the dialog is reserved for failures.
`stripLogPrefixes` also drops the `- ` FreeRDP puts between its logger prefix and
the message, which otherwise led every sentence.

**Measured limitation, stated plainly.** The hint mapping is a keyword table; an
unrecognised cause falls back to a generic sentence rather than inventing a
diagnosis.

## 2. Performance

One extra JSON block per profile, read and written only when the user changes an
option or connects. No polling, no allocation on a hot path.

## 3. Architecture

No new dependency. The store schema gains an optional block: a document written
before it existed loads with the client defaults
(`TestStore_FileWithoutRDPOptions_LoadsTheDefaults`), so no version bump is needed.
The protocol package is unchanged except `stripLogPrefixes`; the profile never
holds a credential and `rdpOptionsFromProfile` cannot produce one, which
`TestRDPOptionsFromProfile_MapsEveryStoredField` asserts.

Deliberate asymmetry: `IgnoreCertificate` and `LegacyTLS` are persisted, which
means a security-relevant choice outlives the session that made it. They are kept
visible as checked boxes on the session bar — the user can always see that a
connection accepts any certificate — instead of being buried in a dialog.

## 4. Evidence

- `./scripts/check.sh`: OK.
- Real client, real failure, through the application's own `Launch`
  (`127.0.0.1:1`, auth-only):

```
CAUSE: connect to localhost:1 failed with error: Connection refused [111]
```

  which is what the dialog shows, followed by the transport hint.
- New tests: `TestStore_RDPOptions_RoundTripThroughTheFile`,
  `TestStore_FileWithoutRDPOptions_LoadsTheDefaults`,
  `TestProfile_Validate_RejectsUnusableRDPOptions`,
  `TestProfile_SameFormAs_IgnoresTheStoredOptions`,
  `TestRDPOptionsFromProfile_MapsEveryStoredField`,
  `TestRDPOptionsToProfile_RoundTripsTheStoredChoices`,
  `TestSessionPanel_SaveRDPOptions_WritesTheProfile`,
  `TestSessionPanel_SaveRDPOptions_StartsFromTheStoredProfile`,
  `TestSessionPanel_SaveRDPOptions_RejectedWrite_KeepsTheOldOptions`,
  `TestSessionPanel_RDPToggle_PersistsAndRevertsOnFailure`,
  `TestSessionPanel_RDPConnect_UsesTheStoredOptions`,
  `TestSessionPanel_RDPImmediateExit_ReportsTheClientCause`,
  `TestSessionPanel_RDPLaterExit_ReportsInTheStatus`,
  `TestRDPFailureHint_PointsAtTheCause`,
  `TestSummarize_DropsTheLoggerSeparator`.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Human: open **RDP options**, set a share and a gateway, press Apply, close the
  app and reopen — the options must still be there, and a failing Connect (a dead
  port) must show the dialog with the client's sentence.
- A reachable RD Gateway is still needed for the manual gateway check recorded in
  `auditory/linux-l7-gateway-20261114-mimocode.md`.
- L7's remaining scope from `blueprint/backlog.md` — "send special keys",
  fullscreen and multi-monitor acceptance on a real host — is untouched.
