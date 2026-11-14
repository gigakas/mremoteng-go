---
timestamp: 2026-11-14T22:26:21Z
agent: mimocode-l12
files:
  - scripts/package-rpm.sh
  - scripts/package-flatpak.sh
  - packaging/flatpak/org.mremoteng.go.yml
  - blueprint/backlog.md
  - blueprint/linux-preview.md
---

Add rpm and Flatpak packaging, and record L9 and L12

Two package formats were missing from L12. scripts/package-rpm.sh generates a spec from the release tarball and calls rpmbuild, installing both binaries (the application and the WebKit helper), the desktop entry and the icon, with the same runtime dependencies as the deb. scripts/package-flatpak.sh stages the tarball and hands packaging/flatpak/org.mremoteng.go.yml to flatpak-builder, declaring network, X11, the config directory and the SSH agent socket. Neither builder is installed on this machine, so neither script has produced an artifact: both validate with bash -n and stop with the exact package list instead of failing halfway, and the backlog says so. The same commit records L9 and L12 in the plan: the five external credential providers were deleted in the preceding L11 commit after a wave in which nothing imported them and no real provider was reachable to verify a wiring, so L9 now states what it ships - the encrypted local vault, with a password stored only when the user asks. Validated with ./scripts/check.sh.
