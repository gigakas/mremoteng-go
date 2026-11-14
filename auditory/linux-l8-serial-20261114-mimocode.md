# Audit — Linux track, Stage L8 (serial sessions)

- **Date (UTC)**: 2026-11-14
- **Agent**: mimocode
- **Audited stage**: L8 — additional protocols, the "serial as a lower-priority extra"
  item of `blueprint/backlog.md`
- **Commits covered**: this commit

## 1. Code quality

`internal/protocol/serial` opens a device as a byte stream and hands it to the terminal
view, which is the same path Telnet and RAW already use.

**Raw line discipline, explicitly.** Every flag is cleared (`Lflag`, `Iflag`, `Oflag`
= 0) and `VMIN=1, VTIME=0` makes a read block until a byte arrives, because a serial
port that looks dead is almost always a line setting nobody looked at: echo, ONLCR or
canonical mode each swallow or mangle what a device sends. `Termios` is separated from
`Open` so the flag arithmetic is testable without a device.

**Settings are stored in the profile** (`profiles.SerialOptions`: baud, data bits,
parity, stop bits), not in the window: the speed is a property of the device. Zero
values mean 9600 8N1, so a store file written before the block existed still opens the
port with settings that work. `SerialOptions.Validate` and `serial.Config.Validate`
refuse the same set of nonsense (unknown speed, 9 data bits, parity "sometimes", three
stop bits) at both ends.

**A serial connection addresses a device, so its host is a path.** `Profile.Validate`
branches for `SERIAL`: the host must start with `/`, carry no control characters,
spaces or `..`, and the record has no network port (`DefaultPort(SERIAL) == 0`). Without
that branch the shared host rule would have rejected `/dev/ttyS0` for containing a
slash — the same rule that keeps a network host from being a URL.

**The UI** adds the protocol to the Config panel, a `Line` button on the serial session
bar (only there; `TestSerialOptionsButton_OnlyOnSerialSessions`), a dialog that names the
device and the four settings, and a status line that spells the settings out
(`serial 115200 8N1 session with /dev/ttyS0 connected`). `startStream` lost its
`telnetclient.Mode` parameter and now takes a label, which is what let serial reuse it
instead of copying the stream path; four call sites were updated.

## 2. Performance

A blocking read per chunk, the same shape as the TCP streams. Nothing polls.

## 3. Architecture

No new dependency: `golang.org/x/sys/unix` was already required for the X11 embedding, and
it carries the termios calls. The protocol package is independent of `internal/ui` and of
the profiles store, like the other protocol packages.

**Verification limit, stated plainly:** there is no serial hardware on this machine. The
exchange test therefore runs against a **pseudo-terminal** created with `/dev/ptmx` +
`TIOCGPTN` + `TIOCSPTLCK`, which proves the open, the termios application and both
directions of the byte stream, but not a real UART's framing. The audit and the backlog
record that limitation instead of claiming hardware acceptance.

## 4. Evidence

- `./scripts/check.sh`: OK.
- New tests: `TestTermios_DefaultsToRaw8N1`,
  `TestTermios_MapsParityStopBitsAndDataBits`, `TestConfig_Validate_RejectsWhatThePortCannotDo`,
  `TestConfig_Description_NamesTheLineSettings`,
  `TestOpen_ExchangesBytesOverAPTYPair` (real read and write over a pty),
  `TestOpen_MissingDevice_ReturnsError`, `TestBaudRates_AreSortedAndIncludeTheDefaults`,
  `TestProfile_Serial_UsesADevicePathHostAndValidLineSettings`,
  `TestStore_SerialOptions_RoundTrip`,
  `TestSessionPanel_ConnectSerial_OpensThePortAndInstallsTheTerminal`,
  `TestSessionPanel_ConnectSerial_InvalidSettings_ReportsWithoutOpening`,
  `TestSessionPanel_ConnectSerial_OpenFailure_IsReported`,
  `TestSerialOptionsFromFields_RejectsWhatThePortCannotDo`,
  `TestSessionPanel_SaveSerialOptions_ReachesTheProfile`,
  `TestSessionPanel_SaveSerialOptions_ReadOnlyStore_ReportsIt`,
  `TestSerialOptionsButton_OnlyOnSerialSessions`.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Hardware check: open a real UART or USB-serial device (a loopback jumper is enough) and
  confirm the terminal echoes what is typed into it at the configured speed.
- Flow control (RTS/CTS, XON/XOFF) is not offered: raw mode clears both. mRemoteNG exposes
  it, so it is a candidate follow-up.
- The device has to exist and be readable by the user; there is no privileged helper and
  no udev rule.
