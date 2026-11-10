---
timestamp: 2026-07-23T19:36:52Z
agent: claude-code
files:
  - auditory/phase2-stage7-20260723-claude-code.md
  - blueprint/phase-2-protocols.md
  - cmd/mremoteng/main.go
  - internal/protocol/anydesk/anydesk.go
  - internal/protocol/anydesk/anydesk_test.go
  - internal/protocol/anydesk/embed_linux.go
  - internal/protocol/anydesk/embed_windows.go
  - internal/protocol/rdp/embed_windows.go
  - internal/protocol/rdp/embed_windows_test.go
  - internal/protocol/winembed/winembed.go
  - internal/protocol/winembed/winembed_test.go
---

Phase 2 stage 2.7: add the AnyDesk backend and extract shared winembed

Implemented AnyDesk as protocol.WindowProtocol, the same external-process + reparent pattern as stage 2.5's RDP, per the blueprint's own description of this stage. Before duplicating stage 2.5's find-and-adopt/DPI-aware-SetParent code a second time, extracted it into a new shared package internal/protocol/winembed (EmbedChild, FindAndAdopt, FindTopLevelForPID, SetMixedDpiHostingBehavior, SetProcessMixedDpiAwareness) and updated RDP to use it too -- anticipated by the Phase 0 spike's own notes ('AnyDesk prep, stage 2.7'). Moved the corresponding test (TestEmbedChild_ReparentsARealExternalWindow and friends, using mspaint.exe as an external-process stand-in) from internal/protocol/rdp to internal/protocol/winembed since it exercises the shared mechanism, not anything RDP-specific; verified RDP's own tests still pass unchanged after the extraction. Deliberately did NOT download and run the actual AnyDesk client, unlike the mingw compiler fetched for stage 2.3: AnyDesk is proprietary live remote-access software with its own account/ID/telemetry behavior, a different and higher-stakes category of action than fetching a build tool, so it wasn't done unattended without the user's awareness -- stated explicitly in the package doc comment and the audit rather than silently worked around. The client-launch/CLI-argument code (address as the connect target, --with-password reading the password from stdin per AnyDesk's documented CLI -- notably avoiding the process-list password exposure RDP's /p: flag has) follows documented behavior but is unverified against a real binary; only Connect's 'client not found' error path is actually testable here. Linux window discovery/embedding is not implemented, same environment gap as RDP's Linux path, plus a genuine (not just environmental) additional gap: AnyDesk has no /parent-window-equivalent launch flag, so Linux embedding would need the generic reparent-after-launch approach the spike explicitly left unfinished. check.sh (including winembed and rdp after the extraction) and smoke.sh green; Linux cross-compile of the anydesk package verified clean.
