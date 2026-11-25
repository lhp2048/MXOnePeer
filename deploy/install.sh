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

if ! id mxgrokbot >/dev/null 2>&1; then
  useradd --system --user-group --shell /usr/sbin/nologin --home-dir /var/lib/mxgrokbot --no-create-home mxgrokbot
fi

install -d -m 0750 -o mxgrokbot -g mxgrokbot /var/lib/mxgrokbot
install -d -m 0755 /opt/mxgrokbot
rm -rf /opt/mxgrokbot/mxgrokbot
cp -a "$root/mxgrokbot" /opt/mxgrokbot/mxgrokbot
chown -R root:root /opt/mxgrokbot/mxgrokbot

if [ ! -x /opt/mxgrokbot/venv/bin/python ]; then
  if ! python3 -m venv /opt/mxgrokbot/venv; then
    echo "python3 -m venv failed; install python3-venv and run this again" >&2
    exit 1
  fi
fi

/opt/mxgrokbot/venv/bin/pip install --disable-pip-version-check "tailscale-py==0.7.1"
install -m 0644 "$root/deploy/mxgrokbot.service" /etc/systemd/system/mxgrokbot.service

if [ ! -f /etc/mxgrokbot.env ]; then
  printf '%s\n' 'TS_AUTHKEY=' > /etc/mxgrokbot.env
  chown root:mxgrokbot /etc/mxgrokbot.env
  chmod 0640 /etc/mxgrokbot.env
fi

systemctl daemon-reload
systemctl enable mxgrokbot.service
systemctl restart mxgrokbot.service
echo "on this machine open http://127.0.0.1:8788/ to save an auth key or start Tailscale login" >&2
