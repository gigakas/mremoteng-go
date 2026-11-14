---
timestamp: 2026-11-14T20:33:52Z
agent: mimocode
files:
  - internal/protocol/rdp/rdp.go
  - internal/protocol/rdp/rdp_test.go
  - internal/ui/session.go
  - internal/ui/rdp_test.go
  - auditory/linux-l7-gateway-20261114-mimocode.md
  - blueprint/backlog.md
  - blueprint/linux-preview.md
---

Pass RD Gateway in the form the client accepts

The gateway was emitted as three flags, /g:, /gu: and /gd:, and the installed FreeRDP 3.32 rejects all three with Unexpected keyword, so any gateway connection died at argument parsing with the usage banner as its only output. The unit test that covered it asserted the old syntax, which is why it stayed green while the feature was broken. Gateway becomes host, port and usage method, rendered as the single argument /gateway:g:host:port[,usage-method:direct|detect], validated before it reaches argv because a comma, a colon or a space in the host ends the argument early. Gateway credentials stay out on purpose: the client refuses a gateway user or domain without a gateway password in the same argument, and a password in argv is readable by every process, so FreeRDP asks for it and the dialog says so. The options dialog gains the gateway section and keeps itself open when the values are refused. Verified against the real client with the exact argv the application builds: the gateway is dialled, usage-method detect asks the target first, and no usage banner appears; a full gateway session was not verified because no RD Gateway is reachable here, which the audit records as pending. Validated with ./scripts/check.sh and ./scripts/smoke.sh.
