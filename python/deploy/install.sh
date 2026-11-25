#!/bin/sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root" >&2
  exit 1
fi

python3 -c 'import sys; raise SystemExit(0 if sys.version_info >= (3, 13) else 1)' || {
  echo "python3 3.13 or newer is required" >&2
  exit 1
}

root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

if ! id mxonepeer >/dev/null 2>&1; then
  useradd --system --user-group --shell /usr/sbin/nologin --home-dir /var/lib/mxonepeer --no-create-home mxonepeer
fi

install -d -m 0750 -o mxonepeer -g mxonepeer /var/lib/mxonepeer
install -d -m 0755 /opt/mxonepeer
rm -rf /opt/mxonepeer/mxonepeer
cp -a "$root/mxonepeer" /opt/mxonepeer/mxonepeer
chown -R root:root /opt/mxonepeer/mxonepeer

if [ ! -x /opt/mxonepeer/venv/bin/python ]; then
  if ! python3 -m venv /opt/mxonepeer/venv; then
    echo "python3 -m venv failed; install python3-venv and run this again" >&2
    exit 1
  fi
fi

/opt/mxonepeer/venv/bin/pip install --disable-pip-version-check "tailscale-py==0.7.1"
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
