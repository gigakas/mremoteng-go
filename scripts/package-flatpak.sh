#!/usr/bin/env bash
# Build a Flatpak bundle from the release tarball.
#
# Usage: scripts/package-flatpak.sh
#   Run scripts/release.sh first: this unpacks dist/mremoteng-go-<version>-linux-<arch>.tar.gz
#   into packaging/flatpak/build/ and hands it to flatpak-builder, using
#   packaging/flatpak/org.mremoteng.go.yml.
#
# Known limit: the manifest declares a local `dir` source, so nothing is downloaded. A real
# Flatpak also needs runtime org.gnome.Platform (>= 47) and the Sdk installed; without them
# flatpak-builder stops at the first step, which this script reports instead of hiding.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

version=$(cat VERSION 2>/dev/null || echo devel)
arch=$(uname -m)
tarball="dist/mremoteng-go-${version}-linux-${arch}.tar.gz"
build_dir="packaging/flatpak/build"
manifest="packaging/flatpak/org.mremoteng.go.yml"
repo_dir="${FLATPAK_REPO:-dist/flatpak-repo}"
bundle="dist/mremoteng-go-${version}-${arch}.flatpak"

if ! command -v flatpak-builder >/dev/null 2>&1; then
    cat >&2 <<EOF
package-flatpak: flatpak-builder is not installed.
  Debian/Ubuntu: sudo apt install flatpak flatpak-builder
  Fedora/RHEL:   sudo dnf install flatpak flatpak-builder
  Then: flatpak remote-add --if-not-exists flathub https://flathub.org/repo/flathub.flatpakrepo
        flatpak install flathub org.gnome.Platform org.gnome.Sdk
Nothing was built.
EOF
    exit 2
fi

if [ ! -f "$tarball" ]; then
    echo "package-flatpak: $tarball is missing; run scripts/release.sh first." >&2
    exit 2
fi

rm -rf "$build_dir"
mkdir -p "$build_dir"
tar -C "$build_dir" --strip-components=1 -xzf "$tarball"

# The manifest points at a directory whose name carries the version, so the source path is
# filled in here rather than kept in a placeholder nobody updates.
staged="packaging/flatpak/source"
rm -rf "$staged"
cp -r "$build_dir" "$staged"
sed "s|../../dist/mremoteng-go-PLACEHOLDER|../source|" "$manifest" >"$build_dir/manifest.yml"

flatpak-builder --force-clean --repo="$repo_dir" "$build_dir/build" "$build_dir/manifest.yml"
flatpak build-bundle "$repo_dir" "$bundle" org.mremoteng.go
(cd dist && sha256sum "$(basename "$bundle")" >"$(basename "$bundle").sha256")
rm -rf "$staged"
echo "package-flatpak: $bundle"
