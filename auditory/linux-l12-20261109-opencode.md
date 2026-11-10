# Audit — Linux track, stage L12 (part 1: Debian package)

- **Date (UTC)**: 2026-11-09
- **Agent**: opencode
- **Audited stage**: L12 partial — `.deb` packaging
- **Scope**: `scripts/package-deb.sh`

## 1. Code quality

- The script reuses `scripts/release.sh`, so the packaged binary is exactly the
  release binary (x11 tag, stripped, version stamped) instead of a second build
  path.
- The staging tree is temporary (`mktemp` + trap), the control file is generated
  from `VERSION`, and `dpkg-deb --root-owner-group` produces reproducible
  ownership without root.
- Runtime dependencies are declared (`xfreerdp3 | xfreerdp`) plus `openssh-client`
  as a recommendation, matching how sessions are launched.

## 2. Performance

- Package is 10.5 MB (25.8 MB installed), dominated by the statically linked Go
  binary and Fyne.

## 3. Architecture

- The package installs `/usr/bin/mremoteng`, the scalable icon and a `.desktop`
  entry pointing at the packaged path; no configuration is written at install
  time, and profiles stay in the user's config directory.
- `.rpm`/Flatpak and Windows remain out of scope for this block.

## 4. Evidence

- `./scripts/package-deb.sh`: builds
  `dist/mremoteng-go_0.4.0~linux~preview_amd64.deb`; `dpkg-deb --info` shows the
  expected control fields, `--contents` lists the three installed files.
- Installing requires root, so `dpkg -i` was not run here.

## 5. Verdict

- [ ] Stage closed unconditionally
- [x] Stage closed with pending actions
- [ ] Stage NOT closed — rework required

## 6. Pending actions

- Install on a clean machine/VM, check the menu entry and the runtime dependency
  prompt, then uninstall.
- Remaining L12: `.rpm`/Flatpak, release notes, Windows port.
