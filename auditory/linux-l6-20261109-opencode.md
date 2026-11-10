# Audit — Linux track, stage L6

- **Date (UTC)**: 2026-11-09
- **Agent**: opencode
- **Audited stage**: L6 — folder defaults and inheritance
- **Scope**: `internal/profiles` (schema v2, Resolve), `internal/ui` (toggles, folder defaults dialog, session targets)

## 1. Code quality

- Native schema version 2 adds `folder_defaults`; version 1 files load unchanged
  and are rewritten as version 2 on the next save (`store.go:Load`), so no manual
  migration step is required.
- `profiles.Resolve` is a pure function with a per-field, nearest-ancestor rule;
  it does not mutate its input and stops searching once every inherited field is
  satisfied. Covered by tests for multi-level chains and flags-off.
- `Profile.Validate` now accepts an unset local port when `InheritPort` is on and
  still rejects out-of-range values, so inheritance cannot bypass validation.
- The Config panel shows inherited values as placeholders and disables the
  corresponding field, which keeps "what will be used" visible without a second
  read-only grid. `refreshInheritance` centralises that.
- Folder defaults are edited from the folder context menu and stored with the
  document; saving empty values removes the record.

## 2. Performance

- Resolution is O(depth × defaults) with a map lookup per level and runs only when
  a profile is loaded or a session starts; folder-depth is small.

## 3. Architecture

- The native document stays the source of truth: defaults are a property of the
  folder path, and profiles carry only opt-in flags.
- Session targets (SSH and RDP) resolve at connect time, so a folder change takes
  effect for the next session without rewriting every profile.
- Scope limit, recorded in the backlog: inheritance covers username and port
  only. Domain/credentials arrive with L9, and the remaining mRemoteNG fields
  stay per-connection until there is a concrete need.

## 4. Evidence

- `./scripts/check.sh`: green.
- New tests: resolver (closest ancestor per field, flags off), validation with
  inherited port, v1→v2 migration, defaults round trip and validation, UI tests
  for saving/clearing defaults, invalid port, and the toggles' placeholders,
  persistence and resolved value.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Manual check: set defaults on a folder, mark two connections as inheriting, and
  confirm a session uses the folder username/port.
- Imported mRemoteNG files still flatten inheritance (L5 mapping); importing after
  L6 could preserve flags as a follow-up.
- Next: L7 (RDP parity) per `blueprint/backlog.md`.
