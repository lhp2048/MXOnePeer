#!/bin/sh
set -eu

if [ "$(uname -s)" != "Darwin" ]; then
  echo "this installer is for macOS" >&2
  exit 1
fi

if [ "$(id -u)" -eq 0 ]; then
  echo "run as the regular user, not root" >&2
  exit 1
fi

case "$(uname -m)" in
  arm64) arch=arm64 ;;
  x86_64) arch=amd64 ;;
  *)
    echo "unsupported Mac architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
bin="$root/dist/mxonepeer-darwin-$arch"
if [ ! -x "$bin" ]; then
  if command -v go >/dev/null 2>&1; then
    (cd "$root" && CGO_ENABLED=0 go build -trimpath -o "$bin" ./cmd/mxonepeer)
  else
    echo "missing $bin; run ./go/deploy/build.sh on a machine with Go, then install again" >&2
    exit 1
  fi
fi

dest="$HOME/Library/Application Support/mxonepeer"
plist="$HOME/Library/LaunchAgents/com.mxonepeer.plist"
xml_escape() {
  printf '%s' "$1" | sed -e 's/&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g'
}
uid=$(id -u)
domain="gui/$uid"
label="$domain/com.mxonepeer"

mkdir -p "$dest" "$HOME/Library/LaunchAgents" "$HOME/Library/Logs"
chmod 700 "$dest"
cp "$bin" "$dest/mxonepeer"
chmod 755 "$dest/mxonepeer"

if [ ! -f "$dest/env" ]; then
  printf '%s\n' 'TS_AUTHKEY=' > "$dest/env"
  chmod 600 "$dest/env"
fi

cat > "$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.mxonepeer</string>
  <key>ProgramArguments</key>
  <array>
    <string>$(xml_escape "$dest/mxonepeer")</string>
    <string>--hostname</string>
    <string>mxonepeer</string>
    <string>--state-dir</string>
    <string>$(xml_escape "$dest")</string>
    <string>--port</string>
    <string>18020</string>
    <string>--admin-host</string>
    <string>127.0.0.1</string>
    <string>--admin-port</string>
    <string>8788</string>
  </array>
  <key>WorkingDirectory</key>
  <string>$(xml_escape "$dest")</string>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>$(xml_escape "$HOME/Library/Logs/mxonepeer.log")</string>
  <key>StandardErrorPath</key>
  <string>$(xml_escape "$HOME/Library/Logs/mxonepeer.err.log")</string>
</dict>
</plist>
EOF

launchctl bootout "$label" >/dev/null 2>&1 || true
launchctl bootstrap "$domain" "$plist"
launchctl enable "$label"
launchctl kickstart -k "$label"
echo "on this Mac open http://127.0.0.1:8788/ to save an auth key or start Tailscale login" >&2
