#!/usr/bin/env bash
# Rebuild the desktop app, replace any running instance and start the new one.
# Every UI change needs this flow: the window title carries version and commit,
# so it is obvious which build is on screen.
#
# Usage: scripts/dev.sh [profiles.json]
#   The optional argument overrides the profile store (handy for throwaway
#   attempts); without it the app uses the user configuration directory.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
command -v go >/dev/null 2>&1 || export PATH="$HOME/.local/go/bin:$PATH"

version=$(cat VERSION 2>/dev/null || echo devel)
commit=$(git rev-parse --short HEAD 2>/dev/null || echo dev)
built=$(date -u +%Y-%m-%dT%H:%M:%SZ)

mkdir -p bin
# The x11 tag makes Fyne expose the X11 window handle, which RDP embedding needs;
# Wayland sessions still work through XWayland.
go build -tags x11 -ldflags "-X main.version=$version -X main.commit=$commit -X main.built=$built" \
    -o bin/mremoteng ./cmd/mremoteng

pkill -x mremoteng || true
sleep 1

args=()
if [ "$#" -ge 1 ] && [ -n "$1" ]; then
    args=(-profiles "$1")
fi
setsid -f bin/mremoteng "${args[@]}" </dev/null >/tmp/mremoteng-dev.log 2>&1
sleep 3

pid=$(pgrep -x mremoteng | head -1 || true)
if [ -z "$pid" ]; then
    echo "dev: app failed to start; last log lines:" >&2
    tail -20 /tmp/mremoteng-dev.log >&2
    exit 1
fi
label="$version ($commit, built $built)"
echo "dev: running $label as pid $pid"
if command -v notify-send >/dev/null 2>&1; then
    notify-send "mremoteng-go restarted" "$label" >/dev/null 2>&1 || true
fi
