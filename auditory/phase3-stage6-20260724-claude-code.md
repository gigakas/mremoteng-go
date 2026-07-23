# Audit — Phase 3, Stage 3.6

- **Date (UTC)**: 2026-07-24
- **Agent**: claude-code
- **Audited stage**: Theming
- **Commits covered**: uncommitted at audit time; see the stage 3.6
  implementation and closing-audit changelog fragments created alongside
  this file

## 1. Code quality

- `internal/ui/theme.go` (new): `ApplyTheme(a fyne.App, name string)`
  maps `Settings.Theme` (stage 3.5) to a Fyne theme. `"system"` (and any
  unrecognized value — a corrupt or future-version settings file
  degrades to this rather than erroring) uses `theme.DefaultTheme()`
  unchanged, which already tracks the OS light/dark preference on its
  own. `"light"`/`"dark"` wrap it in `forcedVariantTheme`, a small
  `fyne.Theme` decorator that overrides only `Color` to substitute a
  fixed variant for whatever variant Fyne's renderer actually asks for.
  Deliberately **not** `theme.DarkTheme()`/`LightTheme()` — both are
  marked deprecated in the Fyne v2.8.0 source specifically in favor of
  "set a custom theme" to ignore user/system preference, which is
  exactly this stage's requirement; using the deprecated functions would
  have been the more obvious-looking choice but the wrong one given
  their own doc comments.
- `forcedVariantTheme` embeds `fyne.Theme` so `Font`/`Icon`/`Size` pass
  straight through unmodified — verified against Fyne's own built-in
  theme implementation that those three methods don't vary by variant
  either, so there's no missing override.
- `internal/ui/optionsdialog.go`: the `themeChoices` package var (added
  in stage 3.5 as a placeholder, with a comment explicitly deferring its
  real meaning to this stage) is now `ui.ThemeChoices`, the single list
  this stage's `ApplyTheme` and stage 3.5's dialog both read — no
  duplicated literal list of theme names to keep in sync by hand. Local
  variable renamed `theme` → `themeSelect` in the same file, since the
  package now also imports `fyne.io/fyne/v2/theme` (in `theme.go`) and a
  same-named local was a latent shadowing footgun even though it never
  actually broke anything (different files, function-local scope).
- `cmd/mremoteng/main.go`: `ApplyTheme(a, cfg.Theme)` called once at
  startup (after settings load, before `ShowAndRun`) and again inside
  `shell.OnOptions`'s save callback, so a theme change made in the
  Options dialog takes effect immediately in the running app rather than
  only on next launch.
- No duplication, no function over ~50 lines, no discarded errors (this
  stage adds none — `ApplyTheme` returns nothing to discard, it's a
  setter).

## 2. Performance

Not applicable: `ApplyTheme` builds one small wrapper struct and calls
`Settings().SetTheme` — startup and options-save only, never a hot path.
Fyne itself owns the cost of re-rendering on a theme change.

## 3. Architecture

- `internal/ui` gains a dependency on `fyne.io/fyne/v2/theme`, already an
  indirect dependency of the whole package (every widget file imports it
  for icons); no new external dependency.
- No changes to `internal/settings`'s public shape — `Settings.Theme`
  already existed from stage 3.5, unused by anything until now. This
  stage is exactly the "3.6 owns theming itself" half the blueprint's
  stage-3.5 note anticipated.
- No impact on closed Phase 1/2 packages or other Phase 3 stages' public
  contracts.

## 4. Evidence — same visual-verification limitation as 3.1-3.5

- `./scripts/check.sh`: **green**.
- `./scripts/smoke.sh`: **green**.
- **No visual verification was possible or attempted**, the same
  phase-wide limitation recorded in every prior stage's audit and in
  `blueprint/phase-3-ui.md`'s note. Of everything built in this phase,
  theming is the stage where that limitation costs the least confidence:
  `forcedVariantTheme`'s behavior (which color a given name+variant
  lookup resolves to) is fully mechanical and directly assertable in a
  headless test, unlike a layout choice that genuinely needs a screen to
  judge.
- New tests (`internal/ui/theme_test.go`, 3 tests):
  `TestApplyTheme_Light_ForcesLightColorsRegardlessOfVariantAsked` and
  the dark equivalent assert that, after `ApplyTheme(a, "light")`, a
  color lookup passed `theme.VariantDark` still returns the *light*
  color (i.e., the override genuinely ignores the caller's requested
  variant, not just happens to match by coincidence) —
  `TestApplyTheme_SystemOrUnrecognized_UsesFynesDefaultTheme` covers
  `"system"`, `""`, and a garbage string all falling back identically to
  `theme.DefaultTheme()`.

## 5. Verdict

- [x] Stage closed with pending actions (listed below)
- [ ] Stage closed unconditionally
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- **No visual confirmation that "light"/"dark" actually look right** —
  the headless tests prove the color-substitution mechanism is correct,
  not that Fyne's built-in light/dark palettes read well for this
  specific app's widgets (tree, terminal, framebuffer view). Repeating
  the phase-wide note.
- **No custom color palette** — this stage reuses Fyne's own built-in
  light/dark themes wholesale rather than porting the original app's
  `Themes/` palette definitions (named theme files with specific
  accent/background colors). A v2 could add named custom palettes the
  same way `forcedVariantTheme` forces a variant, but there was nothing
  in this repo to port from *values* (no `Themes/` assets exist in this
  Go tree) without inventing colors blind, which didn't seem better than
  reusing Fyne's own tested defaults for v1.
- **No per-widget theme overrides** (e.g., the terminal's ANSI palette in
  `internal/ui/ansi.go` is independent of the app theme, by design —
  ANSI colors are a terminal-emulation concern tied to the SGR spec, not
  the app chrome) — worth restating so it isn't mistaken for an oversight
  of this stage.
- Commit the working tree — not done without explicit request.
