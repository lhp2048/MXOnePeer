#!/bin/sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root" >&2
  exit 1
fi

if [ "$(uname -s)" != "Linux" ] || [ "$(uname -m)" != "x86_64" ]; then
  echo "this installer is for 64-bit Linux x86_64" >&2
  exit 1
fi

root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
bin="$root/dist/mxonepeer-linux-amd64"
if [ ! -x "$bin" ]; then
  if command -v go >/dev/null 2>&1; then
    (cd "$root" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "$bin" ./cmd/mxonepeer)
  else
    echo "missing $bin; run ./go/deploy/build.sh on a machine with Go, then install again" >&2
    exit 1
  fi
fi

if ! id mxonepeer >/dev/null 2>&1; then
  useradd --system --user-group --shell /usr/sbin/nologin --home-dir /var/lib/mxonepeer --no-create-home mxonepeer
fi

install -d -m 0750 -o mxonepeer -g mxonepeer /var/lib/mxonepeer
install -d -m 0755 /opt/mxonepeer
install -m 0755 -o root -g root "$bin" /opt/mxonepeer/mxonepeer
install -m 0644 "$root/deploy/mxonepeer.service" /etc/systemd/system/mxonepeer.service

if [ ! -f /etc/mxonepeer.env ]; then
  printf '%s\n' 'TS_AUTHKEY=' > /etc/mxonepeer.env
  chown root:mxonepeer /etc/mxonepeer.env
  chmod 0640 /etc/mxonepeer.env
fi

systemctl daemon-reload
systemctl enable mxonepeer.service
systemctl restart mxonepeer.service
echo "on this machine open http://127.0.0.1:8788/ to save an auth key or start Tailscale login" >&2
