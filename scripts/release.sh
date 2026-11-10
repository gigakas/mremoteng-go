#!/usr/bin/env bash
# Build a release artifact and, optionally, install it for the current user.
#
#   scripts/release.sh                 # dist/ binary, desktop files and tarball
#   scripts/release.sh --install       # also install into ${PREFIX:-$HOME/.local}
#   scripts/release.sh --install --run # install and start the app
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
command -v go >/dev/null 2>&1 || export PATH="$HOME/.local/go/bin:$PATH"

version=$(cat VERSION)
commit=$(git rev-parse --short HEAD 2>/dev/null || echo dev)
built=$(date -u +%Y-%m-%dT%H:%M:%SZ)
arch=$(go env GOARCH)
dist="dist/mremoteng-go-${version}-linux-${arch}"

install=false
run=false
for arg in "$@"; do
    case "$arg" in
        --install) install=true ;;
        --run) run=true ;;
        *) echo "release: unknown argument $arg" >&2; exit 2 ;;
    esac
done

rm -rf "$dist"
mkdir -p "$dist/icons" dist

# The x11 tag is required for RDP window embedding; -s -w strips debug data.
go build -tags x11 -trimpath \
    -ldflags "-s -w -X main.version=$version -X main.commit=$commit -X main.built=$built" \
    -o "$dist/mremoteng" ./cmd/mremoteng

cp internal/ui/assets/icon.svg "$dist/icons/mremoteng-go.svg"
cat >"$dist/mremoteng-go.desktop" <<EOF
[Desktop Entry]
Type=Application
Version=1.0
Name=mRemoteNG Go
GenericName=Connection manager
Comment=Manage SSH and RDP connections, mRemoteNG style
Exec=mremoteng
Icon=mremoteng-go
Terminal=false
Categories=Network;RemoteAccess;
Keywords=ssh;rdp;remote;mremoteng;
StartupWMClass=mRemoteNG Go
EOF

tar -C dist -czf "$dist.tar.gz" "$(basename "$dist")"
(cd dist && sha256sum "$(basename "$dist").tar.gz" >"$(basename "$dist").tar.gz.sha256")

"$dist/mremoteng" -version
echo "release: package $dist.tar.gz"

if [ "$install" = true ]; then
    prefix="${PREFIX:-$HOME/.local}"
    bin_dir="$prefix/bin"
    icon_dir="$prefix/share/icons/hicolor/scalable/apps"
    app_dir="$prefix/share/applications"
    mkdir -p "$bin_dir" "$icon_dir" "$app_dir"
    install -m 755 "$dist/mremoteng" "$bin_dir/mremoteng"
    install -m 644 "$dist/icons/mremoteng-go.svg" "$icon_dir/mremoteng-go.svg"
    sed "s|^Exec=.*|Exec=$bin_dir/mremoteng|" "$dist/mremoteng-go.desktop" >"$app_dir/mremoteng-go.desktop"
    command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "$app_dir" >/dev/null 2>&1 || true
    command -v gtk-update-icon-cache >/dev/null 2>&1 && gtk-update-icon-cache -f -t "$prefix/share/icons/hicolor" >/dev/null 2>&1 || true
    echo "release: installed to $bin_dir/mremoteng"
    echo "release: desktop entry $app_dir/mremoteng-go.desktop"
fi

if [ "$run" = true ]; then
    setsid -f "${PREFIX:-$HOME/.local}/bin/mremoteng" </dev/null >/tmp/mremoteng-release.log 2>&1
    sleep 3
    pid=$(pgrep -x mremoteng | tail -1 || true)
    if [ -z "$pid" ]; then
        echo "release: app failed to start:" >&2
        tail -20 /tmp/mremoteng-release.log >&2
        exit 1
    fi
    echo "release: running installed app as pid $pid"
fi
