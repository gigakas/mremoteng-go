#!/usr/bin/env bash
# Build a Debian package from the release artifacts (dist/). Requires dpkg-deb.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

version=$(cat VERSION)
arch=$(go env GOARCH 2>/dev/null || echo amd64)
case "$arch" in
    amd64) debarch=amd64 ;;
    arm64) debarch=arm64 ;;
    *) debarch="$arch" ;;
esac

./scripts/release.sh >/dev/null

stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

install -d "$stage/DEBIAN" "$stage/usr/bin" \
    "$stage/usr/share/applications" \
    "$stage/usr/share/icons/hicolor/scalable/apps"

install -m 755 "dist/mremoteng-go-${version}-linux-${arch}/mremoteng" "$stage/usr/bin/mremoteng"
install -m 644 internal/ui/assets/icon.svg "$stage/usr/share/icons/hicolor/scalable/apps/mremoteng-go.svg"
sed 's|^Exec=.*|Exec=/usr/bin/mremoteng|' \
    "dist/mremoteng-go-${version}-linux-${arch}/mremoteng-go.desktop" \
    >"$stage/usr/share/applications/mremoteng-go.desktop"

installed=$(du -sk "$stage/usr" | cut -f1)
cat >"$stage/DEBIAN/control" <<EOF
Package: mremoteng-go
Version: ${version//-/\~}
Section: net
Priority: optional
Architecture: $debarch
Installed-Size: $installed
Depends: xfreerdp3 | xfreerdp
Recommends: openssh-client
Maintainer: mremoteng-go contributors
Description: mRemoteNG-style connection manager for Linux
 Manage SSH, RDP, VNC and Telnet connections in a single window.
 RDP sessions run FreeRDP as an external process and are embedded in the tab.
EOF

mkdir -p dist
output="dist/mremoteng-go_${version//-/\~}_${debarch}.deb"
dpkg-deb --root-owner-group --build "$stage" "$output" >/dev/null
dpkg-deb --info "$output" | sed -n '1,12p'
echo "package-deb: $output"
