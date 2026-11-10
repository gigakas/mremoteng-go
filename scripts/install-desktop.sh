#!/usr/bin/env bash
# Install the desktop entry and icon for the local build so the application
# shows up (with its icon) in the desktop application grid.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

binary="$(pwd)/bin/mremoteng"
if [ ! -x "$binary" ]; then
    echo "install-desktop: build first with scripts/dev.sh (missing $binary)" >&2
    exit 1
fi

icon_dir="$HOME/.local/share/icons/hicolor/scalable/apps"
app_dir="$HOME/.local/share/applications"
mkdir -p "$icon_dir" "$app_dir"
install -m 644 internal/ui/assets/icon.svg "$icon_dir/mremoteng-go.svg"

cat >"$app_dir/mremoteng-go.desktop" <<EOF
[Desktop Entry]
Type=Application
Version=1.0
Name=mRemoteNG Go
GenericName=Connection manager
Comment=Manage SSH and RDP connections, mRemoteNG style
Exec=$binary
Icon=mremoteng-go
Terminal=false
Categories=Network;RemoteAccess;
Keywords=ssh;rdp;remote;mremoteng;
StartupWMClass=mRemoteNG Go
EOF

command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "$app_dir" >/dev/null 2>&1 || true
command -v gtk-update-icon-cache >/dev/null 2>&1 && gtk-update-icon-cache -f -t "$HOME/.local/share/icons/hicolor" >/dev/null 2>&1 || true
echo "install-desktop: entry at $app_dir/mremoteng-go.desktop"
