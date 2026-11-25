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

if [ "$(uname -m)" != "arm64" ]; then
  echo "Apple silicon is required; tailscale-py 0.7.1 has no Intel macOS wheel" >&2
  exit 1
fi

python=""
for candidate in python3.13 python3.12 python3.14 python3; do
  if command -v "$candidate" >/dev/null 2>&1 && "$candidate" -c 'import sys; raise SystemExit(0 if sys.version_info >= (3, 12) else 1)'; then
    python=$candidate
    break
  fi
done

if [ -z "$python" ]; then
  echo "python 3.12 or newer is required" >&2
  exit 1
fi

src=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
dest="$HOME/Library/Application Support/mxonepeer"
plist="$HOME/Library/LaunchAgents/com.mxonepeer.plist"
xml_escape() {
  printf '%s' "$1" | sed -e 's/&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g'
}
shell_quote() {
  printf '%s' "$1" | sed "s/'/'\\\\''/g"
}
uid=$(id -u)
domain="gui/$uid"
label="$domain/com.mxonepeer"

mkdir -p "$dest" "$HOME/Library/LaunchAgents" "$HOME/Library/Logs"
chmod 700 "$dest"

rm -rf "$dest/mxonepeer"
cp -R "$src/mxonepeer" "$dest/mxonepeer"

if [ ! -x "$dest/venv/bin/python" ]; then
  if ! "$python" -m venv "$dest/venv"; then
    echo "python venv failed; install Python 3.12 or newer and run this again" >&2
    exit 1
  fi
fi

"$dest/venv/bin/pip" install --disable-pip-version-check "tailscale-py==0.7.1"

if [ ! -f "$dest/env" ]; then
  printf '%s\n' 'TS_AUTHKEY=' > "$dest/env"
  chmod 600 "$dest/env"
fi

quoted_dest=$(shell_quote "$dest")
cat > "$dest/run.sh" <<EOF
#!/bin/sh
set -eu
dest='$quoted_dest'
if [ -f "\$dest/env" ]; then
  key=\$(awk -F= '/^TS_AUTHKEY=/ {print substr(\$0, index(\$0, "=") + 1); exit}' "\$dest/env")
  if [ -n "\$key" ]; then
    export TS_AUTHKEY="\$key"
  fi
fi
export PYTHONPATH="\$dest"
export TS_RS_EXPERIMENT=this_is_unstable_software
exec "\$dest/venv/bin/python" -m mxonepeer --hostname mxonepeer --state-dir "\$dest" --port 18020 --admin-host 127.0.0.1 --admin-port 8788
EOF
chmod 700 "$dest/run.sh"

cat > "$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.mxonepeer</string>
  <key>ProgramArguments</key>
  <array>
    <string>$(xml_escape "$dest/run.sh")</string>
  </array>
  <key>WorkingDirectory</key>
  <string>$(xml_escape "$dest")</string>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <dict>
    <key>SuccessfulExit</key>
    <false/>
  </dict>
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
