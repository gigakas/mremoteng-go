#!/usr/bin/env bash
# Build an rpm from the release tarball.
#
# Usage: scripts/package-rpm.sh
#   Run scripts/release.sh first: this packages dist/mremoteng-go-<version>-linux-<arch>.tar.gz
#   into dist/mremoteng-go-<version>.<arch>.rpm.
#
# Both binaries the release ships are installed (the application and the WebKit helper the
# embedded web view runs), with the desktop entry and the icon. The dependency list mirrors
# scripts/package-deb.sh; keep the two in step.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

version=$(cat VERSION 2>/dev/null || echo devel)
arch=$(uname -m)
release_root="${RPMBUILD_ROOT:-$PWD/dist/rpmbuild}"
tarball="dist/mremoteng-go-${version}-linux-${arch}.tar.gz"

if ! command -v rpmbuild >/dev/null 2>&1; then
    cat >&2 <<EOF
package-rpm: rpmbuild is not installed.
  Debian/Ubuntu: sudo apt install rpm
  Fedora/RHEL:   sudo dnf install rpm-build
Nothing was built.
EOF
    exit 2
fi

if [ ! -f "$tarball" ]; then
    echo "package-rpm: $tarball is missing; run scripts/release.sh first." >&2
    exit 2
fi

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
tar -C "$work" -xzf "$tarball"
source_dir=$(find "$work" -maxdepth 1 -mindepth 1 -type d | head -1)
if [ -z "$source_dir" ]; then
    echo "package-rpm: the tarball has no top-level directory." >&2
    exit 1
fi

# rpm forbids dashes in Version; the preview suffix carries one.
rpm_version=$(echo "${version//-/\~}" | sed 's/[^0-9A-Za-z.~]//g')
spec="$work/mremoteng-go.spec"
cat >"$spec" <<EOF
Name: mremoteng-go
Version: ${rpm_version}
Release: 1
Summary: mRemoteNG-style connection manager for Linux
License: MIT
BuildArch: ${arch}
Source0: ${tarball##*/}
Requires: freerdp3-x11 or freerdp-x11
Requires: libwebkit2gtk-4.1-0
Requires: libgtk-3-0
Suggests: openssh-client
%description
A Linux-first connection manager that mirrors the mRemoteNG desktop layout: a
connection tree, session tabs, SSH, RDP, VNC, Telnet, serial and HTTP sessions.
%prep
%build
%install
mkdir -p %{buildroot}/usr/bin %{buildroot}/usr/share/applications \\
         %{buildroot}/usr/share/icons/hicolor/scalable/apps
install -m 755 "${source_dir}/mremoteng" %{buildroot}/usr/bin/mremoteng
install -m 755 "${source_dir}/mremoteng-webview" %{buildroot}/usr/bin/mremoteng-webview
install -m 644 "${source_dir}/icons/mremoteng-go.svg" \\
    %{buildroot}/usr/share/icons/hicolor/scalable/apps/mremoteng-go.svg
sed 's|^Exec=.*|Exec=/usr/bin/mremoteng|' "${source_dir}/mremoteng-go.desktop" \\
    >%{buildroot}/usr/share/applications/mremoteng-go.desktop
chmod 644 %{buildroot}/usr/share/applications/mremoteng-go.desktop
%files
/usr/bin/mremoteng
/usr/bin/mremoteng-webview
/usr/share/applications/mremoteng-go.desktop
/usr/share/icons/hicolor/scalable/apps/mremoteng-go.svg
EOF

mkdir -p "$release_root"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
cp "$tarball" "$release_root/SOURCES/"
rpmbuild --define "_topdir $release_root" --define "debug_package %{nil}" -bb "$spec"

built=$(find "$release_root/RPMS" -name '*.rpm' | head -1)
if [ -z "$built" ]; then
    echo "package-rpm: rpmbuild reported success but produced no rpm." >&2
    exit 1
fi
cp "$built" "dist/"
(cd dist && sha256sum "$(basename "$built")" >"$(basename "$built").sha256")
echo "package-rpm: $(basename "$built")"
