# Linux preview — manual usability script

This is the checklist a human runs against a built binary. Automated tests cover
storage, validation and UI actions headlessly; this script covers what only a
real desktop and real decisions can confirm.

## Preparation

```bash
go build -o /tmp/mremoteng ./cmd/mremoteng
/tmp/mremoteng -profiles /tmp/usability-profiles.json
```

Use `-profiles` for the trial so your real profile store stays untouched.
Delete `/tmp/usability-profiles.json` afterwards to repeat from scratch.

## Scenarios

1. **Empty start** — the Welcome tab is shown, the tree is empty, the Config
   panel defaults to SSH/22, and the status bar invites creating a connection.
2. **Create** — File > New connection, fill Name `Lab server`, Folder
   `Lab/Servers`, Host `192.0.2.10`, Port `22`, Username `tester`, Save. The
   connection appears under `Lab/Servers`, a session tab opens, and the status
   bar confirms the save.
3. **Validation** — clear the Host and press Save: an error appears in the
   status bar, nothing is written, and your other fields keep their values.
   Repeat with Port `abc` and Folder `../escape`.
4. **Persistence** — quit, start the binary again with the same `-profiles`
   file: the connection and its folder are still there.
5. **Filtering** — type `lab` and then `rdp` in Filter connections: the tree
   narrows and folders without matches disappear; clearing restores it.
6. **Editing** — select the connection, change the port, press Revert: values
   return. Change the port again and switch to another connection: a discard
   prompt appears. Decline and confirm the edit survives; accept and confirm it
   is replaced.
7. **Deleting** — select the connection, press Delete, decline once (nothing
   happens), then accept: the tree entry and its session tab are gone.
8. **Panel toggle** — View > Toggle Config panel hides and restores the panel.
9. **Corruption safety** — with the binary closed, write `{broken` into the
   profile file, start again: the status bar reports the read failure and Save
   and Delete stay disabled, so the file is not overwritten.
10. **Readability** — check that text is legible at your usual window size and
    that the tree, tabs and Config panel resize without clipping.

## Report

Record for each scenario: pass/fail, the window size, and a screenshot of any
layout problem. Session behaviour (SSH/RDP) is out of scope for this preview and
is tracked as stage L2 in `blueprint/linux-preview.md`.
