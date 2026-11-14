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

Scenarios 23 and 25 need the embedded windows, which the single build above does
not give you: RDP embedding needs the `x11` tag, and the web view is a separate
helper binary that is looked for next to the application first and on the PATH
second. Build both into the same directory when you intend to run them:

```bash
go build -tags x11 -o /tmp/mremoteng ./cmd/mremoteng
go build -tags x11 -o /tmp/mremoteng-webview ./cmd/mremoteng-webview
```

Scenario 23 needs a reachable address (`example.com` is enough) and scenario 25 a
reachable RDP host with `xfreerdp3` on the PATH; scenario 24 needs a serial
device on the machine, so it is optional. The other scenarios need nothing beyond
the trial inputs listed above.

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

19. **Settings** — File > Settings, choose `Dark` and press Apply: the window
    repaints at once and the status bar reports
    `Settings saved: dark colours, new connections start with SSH.` Repeat with
    `Light` and then `System`, which follows the desktop again. Pass when the
    last choice is still in force after a restart with the same `-profiles`
    file, including the very first paint, so the window does not start light and
    turn dark. Then set the protocol to `TELNET` and the port to `2323`, Apply,
    and press File > New connection: the Config panel starts with TELNET and
    2323. Clear the port field instead and a new connection starts with the
    protocol's own default (TELNET/23). A port of `abc` reports
    `the default port "abc" is not a number` and `70000` reports
    `the default port 70000 is outside 1-65535`, both inside the dialog, which
    stays open and leaves the previous settings untouched. The protocol picker
    here offers SSH, RDP, VNC, TELNET, RAW, rlogin and HTTP; SERIAL is created
    from the Config panel, not as a default. The colour choice, the defaults and
    the reopen toggle live in the window's preferences rather than in the
    `-profiles` file, so deleting the trial store does not reset them.
20. **Session restore** — open three connections so three session tabs exist,
    close the third with its own X, select the second, then quit and start again
    with the same `-profiles` file, leaving
    `Reopen the tabs of the last session at startup` ticked in File > Settings
    (that is the default). Pass when exactly the two tabs that were still open
    come back, in the same order, each unconnected — the placeholder is shown
    and the session bar offers Connect rather than Disconnect — and the tab that
    was selected when you quit is selected again. Then split one of them
    (View > Split pane right or down), connect both panes, quit and restart: the
    divided tab comes back divided, with both panes unconnected and the divider
    where you left it (a fresh split sits at half). A quick-connect tab never returns, a tab whose connection was
    deleted is dropped instead of being resurrected, and unticking the reopen
    toggle makes the next start open with the Welcome tab and no sessions.
21. **Notifications** — create a connection to a dead port (`127.0.0.1`, port
    `1`, username `tester` so the attempt reaches the network) and press Connect
    twice, waiting for each attempt to fail. Pass when
    the status-bar badge on the warning icon counts `1` and then `2`, its hover
    hint reads `Show 2 problems the status bar has already forgotten`, and the
    panel (View > Notifications or the badge) is titled `Notifications (2)`,
    lists both failures oldest first, each as a time, its level and the message
    the status bar has already replaced. Clear empties the list to
    `No problems reported.`, removes the badge count and reports
    `Notifications cleared.` The panel and the activity log share the one bottom
    slot: opening either closes the other, and neither takes height the terminal
    needs. The history is this run only, so a restart begins with an empty list.
22. **Manual tree order** — with a few connections in one folder, right-click
    one of them and choose Move up twice. Pass when the row climbs one place per
    click among the connections of that folder, the status bar reports
    `Moved <name> up.` each time and the row stays selected, folders keep the
    positions above the connections (so a connection never jumps over a folder,
    and at the top of its group Move up is greyed out), and the order is
    unchanged after a restart with the same `-profiles` file. Move down works
    the same way, folders can be reordered among themselves, and reordering
    while the filter is active still leaves the hidden rows where they were once
    the filter is cleared.
23. **Embedded web view** — build the helper as described in Preparation, create
    a connection with protocol HTTP, host `example.com`, port `443`, Save and
    press Connect. Pass when the page is painted inside the tab (the address is
    `https://example.com`, because port 443 is the scheme's own and is not
    shown), the status bar reports
    `HTTP session for example.com embedded in the tab.`, the page disappears
    while another tab is selected and returns with its own tab, a dialog opened
    on top of the session is fully visible instead of hidden behind the page,
    and resizing the window keeps the page inside its tab without spilling over
    the tree, the status bar or the neighbouring panes. Then close the helper
    case: move `/tmp/mremoteng-webview` aside, restart and connect the same
    profile — the placeholder above the terminal already says the system browser
    will be used, Connect does exactly that, and the status bar names the reason
    as `Opened https://example.com in the system browser because …` (the
    embedded view could not start, or there was no window to embed into).
24. **Serial (optional — needs a device)** — skip this one on a machine without
    a serial port, it needs hardware. Create a connection with protocol SERIAL
    and host `/dev/ttyS0` (or a real device such as `/dev/ttyUSB0`), port left
    empty. Pass when saving the bare name `ttyS0` is refused with
    `… is not a device path`, the `Line` button on the session bar opens the line
    settings with device, Baud rate, Data bits, Parity and Stop bits, Apply
    stores them in the profile and the status bar reports
    `Serial settings saved: /dev/ttyS0 9600 8N1.` (9600 8N1 are the defaults,
    the dialog shows what is stored), the settings are still there after a
    restart, and Connect opens a terminal in the tab that shows what the device
    sends and passes what you type back to it.
25. **RDP keys, full screen and options** — open an RDP connection to a
    reachable host and let it embed in the tab. With that tab selected, use
    View > Send keys > Ctrl+Alt+Del: the remote desktop shows its own security
    screen and the status bar reports `Sent Ctrl+Alt+Del to the session.` The
    computer-icon button on the session bar sends the same chord, and its hover
    hint names it. Pass when the menu and the button both reach the embedded
    session, and a session that ended up in its own window (multiple monitors)
    reports `Special keys need the session embedded in the tab; …` instead of
    silently doing nothing. View > Full screen then grows the window to the
    screen and reports
    `Full screen. View > Full screen returns to the window.`, the menu item is
    marked while it lasts, and choosing it again reports `Left full screen.`
    Edit the RDP options on the session bar (clipboard, audio, dynamic
    resolution, the gateway): Apply reports `RDP options saved for <name>…`, and
    the ticks, along with what the next Connect uses, are unchanged after a
    restart, because they belong to the profile and no longer to the window.
    Finally point the connection at a port nothing listens on and press Connect:
    the status bar reports the failure and a dialog titled `RDP session failed`
    shows the connection and its address,
    `FreeRDP ended before opening a session: ` with FreeRDP's own sentence, and
    the hint that sentence calls for.

## Known limits

- Modifier-key input is not available in the terminal: the toolkit delivers no
  modifier state, so Ctrl-C and Ctrl-D are buttons next to the terminal. The
  View > Send keys chords are the exception, and they only reach an embedded
  session.
- A pane never reconnects by itself when a tab is restored: no credential is
  stored, so every pane waits for its own Connect.
- The embedded web view has no navigation controls of its own — no back,
  forward, reload or address bar. Use the system browser for that.
- Serial connections offer no flow control beyond the handshake the Line dialog
  sets (none, RTS/CTS or XON/XOFF), and no modem lines (DTR, DSR, RI, DCD).
- Notifications are a plain text list, not clickable: an entry cannot be
  selected, clicked or copied, and the same line lives on in the activity log
  only until one of the two is cleared.

## Report

Record for each scenario: pass/fail, the window size, and a screenshot of any
layout problem. Sessions are in scope: SSH with the embedded terminal, RDP
through the embedded FreeRDP window, VNC, Telnet/RAW/rlogin, the embedded HTTP
web view with the system browser as its fallback, and serial when a device is at
hand. This script is still pending human execution; stage L1 in
`blueprint/linux-preview.md` stays open until the recorded run exists, and
scenario 18 does not close stage L9.
