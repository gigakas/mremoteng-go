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

Two trial inputs are needed by later scenarios:

- A reachable SSH host you own (login with a password or an agent), for the
  session scenarios (16, 17, 18). Scenarios without a session still pass on a
  host-less machine if you only check the dialogs and the status messages.
- A scratch copy of an imported file and scratch export paths:
  `cp testdata/corpus/v26-default.xml /tmp/usability-import.xml`. Never point
  Import or Export at a file under `testdata/corpus/`.

`-profiles` does not redirect the credential vault: it lives at
`$XDG_CONFIG_HOME/mremoteng-go/secrets.json` (normally
`~/.config/mremoteng-go/secrets.json`). Move that file aside before scenario 18
if it already exists and restore it afterwards.

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
11. **Folders — create and rename** — File > New folder, enter `Lab`. Right-click
    `Lab` > New folder inside, then right-click the resulting
    `Lab/New folder` > Rename folder and change it to `Servers`. Put a
    connection into `Lab/Servers`. Pass when both folders exist without holding
    anything, the rename moves the subtree (the connection is still listed under
    `Lab/Servers` and its Folder field shows the new path), and both folders and
    the connection are still there after a restart with the same `-profiles`
    file.
12. **Folders — recursive delete** — put one connection inside `Lab/Servers`,
    then right-click `Lab` > Delete folder. Pass when the confirmation names the
    number of connections it will remove (1) and that count matches the
    subtree, declining leaves folder, connection and any open session tab
    untouched, and accepting removes `Lab`, `Lab/Servers`, the connection and its
    session tab, with nothing left after a restart.
13. **Folders — drag and Move to** — drag the `Lab server` row onto the `Lab`
    folder row, then drag it onto empty space in the tree. Pass when the status
    bar reports `Moved … to …` for the first drop and `Moved … to the root.` for
    the second, no duplicate row appears, the Config panel Folder field matches
    the drop target, right-click > Move to lists the root plus every folder and
    marks the current one with `(current)`, and the final position survives a
    restart.
14. **Import and export round trip** — File > Import connections and pick
    `/tmp/usability-import.xml`. Pass when the preview names the format and the
    number of connections and folders it found, mentions the skipped unsupported
    rows and the dropped passwords if the file contains any, Merge keeps existing
    entries and skips duplicates, Replace replaces the tree, Cancel changes
    nothing, and
    `sha256sum /tmp/usability-import.xml` is unchanged afterwards. Then File >
    Export as XML to `/tmp/usability-export.xml` (leave the optional password
    empty) and File > Export as CSV to `/tmp/usability-export.csv`. Pass when the
    status bar reports the number of exported connections, re-importing
    `/tmp/usability-export.xml` into a fresh `-profiles` file reproduces the same
    names, folders, hosts and ports, and an export written with a non-empty
    password asks for exactly that password when it is imported again.
15. **Quick connect** — File > Quick connect, Host `192.0.2.10`, protocol SSH,
    empty port, Username `tester`, Connect. Pass when a tab titled
    `Quick: 192.0.2.10` opens, the status bar says the session is not saved, the
    tree gains no entry, and the tab is gone after a restart. An empty host
    reports `Quick connect needs a host.` and port `abc` reports
    `Quick connect port must be a number.`
16. **Terminal zoom** — with an SSH session tab open and focused, press Ctrl+=
    and Ctrl+- a few times, then Ctrl+0, and repeat through View > Terminal zoom
    in/out/reset. Pass when the text grows and shrinks in small steps, Ctrl+0 and
    the reset item return to the default theme size, only the focused terminal
    changes, the grid refits without clipped columns or rows, and the size stays
    with that tab while it is open. The wheel over the terminal scrolls the
    history instead of zooming.
17. **Scrollback and copy** — run `seq 1 5000` in an SSH session, then scroll up
    with the wheel and with PageUp, then return with PageDown. Pass when the view
    walks into earlier output while the shell receives nothing, the wheel and
    PageDown bring the live prompt back, the history stops at 2000 lines so the
    earliest numbers are gone and the application stays responsive. Select a few
    lines with the mouse and right-click: the status bar reports
    `Selection copied to the clipboard.`, the highlight clears, and the block
    pastes into another application. With nothing selected, the copy icon on the
    session bar reports `Whole buffer copied (scrollback plus screen)…` and the
    clipboard holds the earlier lines too.
18. **Credential vault** — File > Credentials shows `Credential vault: Locked`.
    Create with `short` and confirm it is rejected for the 8-character minimum;
    create with a longer master password and confirm the status bar says the
    vault was created and unlocked. Open an SSH connection, type the password,
    press the save icon on the session bar and confirm
    `Password stored for … in the encrypted vault.`. Pass when
    `grep -c '<the password>' /tmp/usability-profiles.json` finds nothing, the
    vault file is mode `0600`, connecting again with an empty password field
    while the vault is unlocked uses the stored secret without prompting, the
    save icon while locked reports
    `Unlock the credential vault first (File > Credentials).`, the same
    connection no longer authenticates with an empty password field after Lock, a
    wrong master password reports a failure and the vault still unlocks with the
    correct one, and a fresh start of the binary begins locked.

## Report

Record for each scenario: pass/fail, the window size, and a screenshot of any
layout problem. Sessions are in scope: SSH with the embedded terminal, RDP
through the embedded FreeRDP window, VNC, Telnet/RAW/rlogin, and the HTTP
handoff to the system browser (the embedded webview is not implemented). This
script is still pending human execution; stage L1 in
`blueprint/linux-preview.md` stays open until the recorded run exists, and
scenario 18 does not close stage L9.
