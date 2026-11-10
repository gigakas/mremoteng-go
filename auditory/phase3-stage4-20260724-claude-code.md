# Audit — Phase 3, Stage 3.4

- **Date (UTC)**: 2026-07-24
- **Agent**: claude-code
- **Audited stage**: Connection properties panel (with inheritance UI)
- **Commits covered**: uncommitted at audit time; see the stage 3.4
  implementation and closing-audit changelog fragments created alongside
  this file

## 1. Code quality

- `internal/ui/properties.go` — `PropertiesPanel`: reflection-driven
  rather than ~75 hand-written field bindings (one per
  `connection.InheritanceFlags` field). `buildPropertyFields` runs once at
  package init, matching `connection.ConnectionValues`' fields to
  `connection.InheritanceFlags`' by name — a field with no counterpart
  (e.g. `Name`, `Hostname`: confirmed by reading
  `internal/connection/inheritance.go` directly rather than assumed, since
  guessing which fields the original model treats as inheritable is easy
  to get wrong and a first draft of this stage's own tests did exactly
  that) gets `inheritIdx = -1` and renders with no "Inherit" checkbox.
  This mirrors how the original C# app's own property grid worked
  (.NET's `PropertyGrid` is itself reflection-based) rather than being a
  novel approach for this kind of UI.
- **A real feedback-loop bug found and fixed by a failing test**: the
  first version of `commit` (called from every widget's `OnChanged`)
  unconditionally wrote the widget's *currently displayed* value back into
  `Raw`. `refresh` — called at the end of every `commit`, to update the
  display after a change — calls `widget.Entry.SetText`/
  `widget.Check.SetChecked` to show the (possibly now-inherited) value,
  but Fyne's `SetText`/`SetChecked` fire `OnChanged` exactly the same as a
  genuine user edit would. Net effect: checking "Inherit" would trigger
  `commit` → set the flag → `refresh` → `SetText(effectiveValue)` →
  `OnChanged` fires again → a *second*, recursive `commit` call → writes
  the just-displayed *effective* value back into `Raw`, silently
  overwriting whatever local override was there a moment before.
  `TestPropertiesPanel_TogglingInherit_UpdatesFlagsAndDisplay` caught this
  on the first run (not found by inspection). Fixed with a `refreshing`
  guard bool on `PropertiesPanel`, set for the duration of `refresh`, that
  `commit` checks and bails out on — the standard fix for this exact class
  of programmatic-update-reenters-the-change-handler bug, documented
  inline with the failing test's name so it isn't quietly reintroduced.
- v1 scope, stated in `PropertiesPanel`'s own doc comment rather than
  discovered by the reader: a **flat list** in model declaration order,
  not the original app's categorized/tabbed grid (Go reflection can't see
  the source comments that delimit `model.go`'s Display/Connection/
  Protocol/... sections, so categorizing would need a hand-maintained
  field→category map — deferred, not attempted); every field is a plain
  text `Entry` (numbers reparsed via `strconv.Atoi`) or a `Check` for
  bools, not the enum-aware dropdowns a polished grid would have for the
  ~20 named string-enum types (`ProtocolType`, `RDPVersion`, etc.) — both
  gaps chosen because this phase's stated inability to see the running
  app makes iterating on a denser, more form-like layout hard to get
  right blind, where a flat scrollable list is at least fully functional
  and easy to reason about without seeing it.
- No duplication, no function over ~50 lines beyond `buildRows` (which is
  one loop building one row per field — splitting it further would add
  indirection without reducing complexity), no discarded errors.

## 2. Performance

Not applicable: ~75 rows built once per panel instance; `refresh` does one
reflect field read per row on selection change, not a hot path.

## 3. Architecture

- Stays inside `internal/ui/`; imports only `internal/connection` (Phase
  1, closed) and Fyne — no new dependencies.
- `Shell` (stage 3.1) gains a third pane: `SetProperties`, with the tree
  and properties panel stacked in a `container.NewVSplit` on the left,
  tabs filling the rest. This is a `Shell` structural change made in this
  stage (justified: 3.4 is exactly what needed a place to render), not an
  out-of-scope touch — `shell_test.go` was updated alongside it (new
  placeholder-type assertion, new `TestShell_SetProperties_
  ReplacesPlaceholder` test) rather than left stale.
- `cmd/mremoteng/main.go`: `tree.OnSelect` now calls
  `properties.SetTarget(node)` for *any* selected node (connection or
  folder — both are `connection.Node` with the same value/inheritance
  shape), before its existing leaf-only tab-opening logic. Real wiring,
  reachable as soon as stage 3.5 populates the tree with anything.
- No impact on closed Phase 1/2 packages or other Phase 3 stages' public
  contracts.

## 4. Evidence — same visual-verification limitation as 3.1-3.3

- `./scripts/check.sh`: **green**.
- `./scripts/smoke.sh`: **green**.
- **No visual verification was possible or attempted**, the same
  phase-wide limitation recorded in every prior stage's audit and in
  `blueprint/phase-3-ui.md`'s note. This stage's flat-list layout and the
  tree/properties vertical split are the least likely of anything built
  so far to be the *right* layout choice on first guess — a form-style
  properties grid is exactly the kind of UI that benefits most from
  seeing it and iterating, which wasn't possible here.
- New tests (`internal/ui/properties_test.go`, 10 tests; plus 1 new test
  in `shell_test.go` for `SetProperties`): row generation matches the
  model (including the Username-is-inheritable/Hostname-is-not distinction
  confirmed against the real model rather than assumed), bool fields use
  `Check` and string fields use `Entry`, raw-value population, the
  inherited-value-display case, editing commits to `Raw`, **the
  toggle-inherit feedback-loop bug** (now passing after the fix above),
  int parsing including the invalid-input-leaves-value-unchanged case, and
  clearing via `SetTarget(nil)`.

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- **No visual confirmation of the layout** — repeating the phase-wide
  note, with this stage's flat-list/no-categorization choice specifically
  flagged as the most likely candidate for a real redesign once someone
  can see it.
- **No categorization/tabs** (Display/Connection/Protocol/Gateway/
  Appearance/Redirection/Misc/VNC, matching the original grid's sections)
  — a hand-maintained field→category map is the natural fix, deferred for
  the reason in section 1.
- **No enum-aware dropdowns** for the ~20 named string-enum fields
  (`ProtocolType`, `RDPVersion`, `VNCEncoding`, ...) — currently plain
  text entry, so a typo produces an unrecognized enum value silently
  accepted at the UI layer (protocol.Create or the serializer would be
  where an invalid value actually surfaces as an error, not this panel).
- **Selecting a connection immediately attempts to connect** (via stage
  3.3's `tree.OnSelect` → `tabs.Open`, unchanged by this stage but now
  more noticeable since properties also populate on the same click) —
  worth reconsidering later whether opening a session should require an
  explicit action (double-click, a "Connect" button) instead of a single
  selection click, once there's a real UI to feel out; not changed here
  since it's a stage 3.3 decision this stage didn't own.
- Commit the working tree — not done without explicit request.
