# Audit — Phase 4, Stage 4.2

- **Date (UTC)**: 2026-11-16
- **Agent**: opencode
- **Audited stage**: 4.2 — CI: build + check + smoke on every PR
- **Commits covered**: the working-tree change that adds
  `.github/workflows/ci.yml` and `.github/workflows/package-linux.yml` (left
  uncommitted at audit time), plus the stage row in
  `blueprint/phase-4-packaging.md`

## 1. Code quality

**The workflows own no build logic.** `ci.yml:44-48` calls
`./scripts/check.sh` and `./scripts/smoke.sh`, the same two scripts a developer
runs, so the gate cannot drift from the local procedure. `package-linux.yml:49-53`
likewise calls `scripts/release.sh` and `scripts/package-deb.sh` instead of
re-implementing the build, and gets the `-tags x11` and `-trimpath -ldflags`
choices from those scripts rather than duplicating them.

**Payload verification.** `package-linux.yml:57-67` lists the `.deb` and asserts
the four installed paths (`mremoteng`, `mremoteng-webview`, the desktop entry
and the scalable icon) before `-version` proves the packaged binary starts. A
script failure that still produced a package would be caught here; this is the
only check that would notice a lost helper binary.

**Fixture honesty.** `package-linux.yml:102-108` refuses a tag that disagrees
with `VERSION` instead of renaming the artifacts, so a release file name can
never describe a different build.

**Hardening.** Action versions are pinned by commit SHA with the tag in a
trailing comment (`ci.yml:23,28`; `package-linux.yml:28,33,70,90,95`);
`persist-credentials: false` keeps the checkout token out of `.git/config`; the
workflow-level permission is `contents: read` and only the `release` job raises
it (`package-linux.yml:86-87`); the release step uses the runner's `gh` with
`GITHUB_TOKEN` instead of a third-party release action; `if-no-files-found:
error` and a 30-day retention keep a broken artifact from disappearing silently.

**Tests.** None added: the change is declarative YAML and the behaviour it
encodes is the existing scripts. A negative-control test (a red `check.sh`) would
have to run on GitHub, which this environment cannot do; the local evidence in
section 4 is the substitute.

**Known rough edge.** Both workflows install the same apt package list
(`ci.yml:36-42`, `package-linux.yml:38-44`). Duplication is deliberate for now:
one is the merge gate, the other builds artifacts, and a shared composite action
would be a third file to maintain. It should be extracted if a third workflow
needs the same list.

## 2. Performance

- `scripts/check.sh` runs `go test -race ./...`, so a cold run dominates the job
  (about 3-4 minutes on `ubuntu-24.04` for this repository); `go.mod`-driven
  `setup-go` caching (`ci.yml:28-31`) removes the module download on repeat runs.
- `concurrency` with `cancel-in-progress` (`ci.yml:13-15`,
  `package-linux.yml:18-20`) discards superseded pushes, so a branch with several
  quick commits costs one run.
- The package job builds twice (release tarball, then the `.deb` from the same
  staged tree), which is inherent to `package-deb.sh` calling `release.sh`; the
  second build reuses the Go cache. Estimated 4-6 minutes per run.
- No benchmark is meaningful here; there is no runtime code path involved.

## 3. Architecture

- Files stay inside the stage's ownership (`blueprint/phase-4-packaging.md:6`:
  `Makefile`, `.github/workflows/`, `packaging/`). No Go package is touched, no
  dependency is added.
- The `blueprint/linux-preview.md` track overrides the historical 4.1 → 4.2
  dependency: 4.1 (cross-compilation matrix) is still pending and 4.2 does not
  consume its targets, because `release.sh` builds only the host architecture.
  Recording that explicitly here is the documented exception, not an oversight.
- Only `linux/amd64` is built. arm64 needs cgo, which `go build` does not
  cross-compile from an amd64 runner without a cross toolchain; the honest
  options are a second `ubuntu-24.04-arm` job (public repositories only) or a
  container-based build, and both are deferred to 4.3/4.5.
- `.deb` runtime dependencies (`freerdp3-x11 | freerdp-x11`,
  `libwebkit2gtk-4.1-0`, `libgtk-3-0`, see `scripts/package-deb.sh:38`) are
  declared but never exercised in CI. The workflow proves the package is
  well-formed, not that it installs on a clean system; that remains part of 4.3.
- Flatpak and rpm are not built in CI (`packaging/flatpak` needs a GNOME 47
  runtime and `flatpak-builder`), so 4.3 stays `in progress` in the phase table.

## 4. Evidence

- `./scripts/check.sh`: OK (gofmt, `go vet`, `go test -race ./...`) on
  2026-11-16.
- `./scripts/smoke.sh`: OK (all binaries build with `-tags x11`, `mremoteng
  -version` prints the build label, `changelog compile` reproduces `CHANGELOG.md`)
  on 2026-11-16.
- Workflow YAML parses with a strict YAML loader and both files expose the
  intended `on`/`permissions`/`jobs` keys.
- New tests in this stage: none (see section 1).
- Not verifiable here: the first GitHub Actions run itself. The workflows must
  be confirmed green on the remote, and a `v*` tag must be tried once before the
  release path can be called working.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions (listed below)
- [ ] Stage NOT closed — rework required

## 6. Pending actions

1. **First remote run** (opencode, next push): confirm `CI` is green and that
   branch protection on `main` requires it.
2. **arm64 artifacts** (opencode): add an `ubuntu-24.04-arm` job once the
   repository visibility allows free arm runners, or a cross toolchain.
3. **Install verification** (claude-code, stage 4.3): `lintian` plus a container
   test that installs the produced `.deb` and starts `mremoteng -version`, so the
   declared runtime dependencies are proven.
4. **Release policy** (human + claude-code, stage 4.5): decide the tag format.
   `VERSION` is `0.6.4-linux-preview`, so the guard currently accepts only
   `v0.6.4-linux-preview` (or the bare version); a `v0.6.5`-style tag would be
   rejected until the channel suffix policy is settled.
5. **Dependency list de-duplication** (opencode): extract the apt list into a
   composite action if a third workflow needs it.
