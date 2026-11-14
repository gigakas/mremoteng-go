---
timestamp: 2026-11-14T20:40:44Z
agent: mimocode
files:
  - internal/profiles/profiles.go
  - internal/profiles/profiles_test.go
  - internal/ui/rdp_options.go
  - internal/ui/rdp_options_test.go
  - internal/ui/session.go
  - internal/ui/actions.go
  - internal/ui/actions_test.go
---

Store RDP options per connection instead of per tab

Redirection, display and gateway choices lived in the tab's memory, so every session started from the client defaults and a user who needed Ignore cert or a gateway retyped it on each connection. The profile now carries an rdp block (clipboard, printers, audio, shared drives, dynamic resolution, multi monitor, legacy TLS, ignore certificate and the gateway hop), validated at the store boundary because those values end up in a FreeRDP argument list where a comma or a colon changes the meaning of the whole line. The panel reads the options from the profile for Connect, for the credentials test and for the dialog, and writes them back through the workspace, so the Legacy TLS and Ignore cert boxes persist as well and revert visibly when the store rejects the write. An older store file without the block loads with client defaults. Profile comparisons moved to SameFormAs because the Config panel does not edit these options: comparing them directly would make every connection with stored options look permanently modified. Validated with ./scripts/check.sh.
