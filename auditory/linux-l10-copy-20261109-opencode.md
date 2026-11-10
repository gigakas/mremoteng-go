# Audit — Linux track, stage L10 (part 3: terminal copy) and L11 repair

- **Date (UTC)**: 2026-11-09
- **Agent**: opencode
- **Audited stages**: L10 partial — copy visible terminal; L11 partial — store repair

## 1. Code quality

- `screenText` reads the emulator snapshot, replaces empty cells with spaces,
  trims trailing blanks per line and drops trailing empty lines, so the clipboard
  gets usable text rather than a padded grid. It is pure with respect to the
  emulator and covered by a test that writes two lines.
- Copy is a low-importance icon on the terminal bar for SSH and stream sessions;
  it reports when no terminal is open and when the platform has no clipboard.
- `repairStore` renames the damaged file to a timestamped `.bak-<UTC>` and writes
  an empty document through the regular store path, then reloads. The test starts
  from a corrupt file, checks that writes were disabled, repairs, and asserts one
  backup plus an empty store.

## 2. Performance

- Copy walks rows×columns once; repair is a rename plus one small write.

## 3. Architecture

- Repair uses the store interface plus the optional path capability, so stores
  without a file keep working (the menu item explains it instead of failing).
- No scrollback buffer was added: the emulator exposes only the visible screen, and
  a real history needs a buffer with its own tests and memory bound. Documented as
  a gap rather than faked.

## 4. Evidence

- `./scripts/check.sh`: green.
- New tests: terminal text extraction, repair backup/empty store/writes re-enabled.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Manual: copy from a live SSH session.
- Scrollback remains open; decide buffer size and copy/scroll UX first.
