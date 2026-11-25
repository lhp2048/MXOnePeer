import argparse
import asyncio
import ipaddress
import os
import signal
import sys

from mxonepeer.runtime import run

_CLEARED = (
    "TS_AUTH_KEY",
    "TS_CLIENT_SECRET",
    "TS_CLIENT_ID",
    "TS_ID_TOKEN",
    "TS_AUDIENCE",
    "TS_CONTROL_URL",
    "TS_HOSTNAME",
)


def main(argv=None):
    parser = argparse.ArgumentParser(prog="mxonepeer")
    parser.add_argument("--hostname", default="")
    parser.add_argument("--state-dir", default="/var/lib/mxonepeer")
    parser.add_argument("--port", type=int, default=18020)
    parser.add_argument("--admin-host", default="127.0.0.1")
    parser.add_argument("--admin-port", type=int, default=8788)
    args = parser.parse_args(argv)
    if not _loopback(args.admin_host):
        print("admin host must be 127.0.0.1 or ::1", file=sys.stderr)
        return 1
    auth_key = os.environ.get("TS_AUTHKEY", "")
    os.environ.pop("TS_AUTHKEY", None)
    for name in _CLEARED:
        os.environ.pop(name, None)
    try:
        asyncio.run(
            _run(
                args.hostname,
                args.state_dir,
                args.port,
                args.admin_host,
                args.admin_port,
                auth_key,
            )
        )
    except Exception as exc:
        print(_redact(str(exc), auth_key), file=sys.stderr)
        return 1
    return 0


async def _run(hostname, state_dir, port, admin_host, admin_port, auth_key):
    stop = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGTERM, signal.SIGINT):
        loop.add_signal_handler(sig, stop.set)
    await run(hostname, state_dir, port, admin_host, admin_port, auth_key, stop)


def _loopback(host):
    try:
        return ipaddress.ip_address(host).is_loopback
    except ValueError:
        return False


def _redact(message, auth_key):
    if auth_key:
        return message.replace(auth_key, "tskey-auth-***")
    return message
