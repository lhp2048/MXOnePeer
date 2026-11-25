import asyncio
import os
import sys

from mxonepeer.httpio import parse_request, response_for

HEADER_LIMIT = 8192
HEADER_TIMEOUT = 5


async def connect_device(auth_key, hostname, state_dir):
    os.makedirs(state_dir, mode=0o750, exist_ok=True)
    os.chmod(state_dir, 0o750)
    tailscale = _import_tailscale()
    return await tailscale.connect(
        os.path.join(state_dir, "node-keys.json"),
        auth_key or None,
        hostname=hostname,
    )


async def serve_device(dev, port, stop, board=None):
    addresses = await _addresses(dev)
    if not addresses:
        raise RuntimeError("tailscale connected without an address")
    listeners = []
    for address in addresses:
        listeners.append(await dev.tcp_listen((address, port)))
    tasks = [asyncio.create_task(_accept(dev, listener, stop)) for listener in listeners]
    tasks.append(asyncio.create_task(_watch(dev, board, stop)))
    try:
        await stop.wait()
    finally:
        for task in tasks:
            task.cancel()
        await asyncio.gather(*tasks, return_exceptions=True)
        if board is not None:
            board.online = False


async def _watch(dev, board, stop):
    while not stop.is_set():
        if board is not None:
            try:
                hostname, _dns, ips, backend = await _snapshot(dev)
                board.hostname = hostname
                board.ips = ips
                board.online = backend == "Running"
            except Exception:
                board.online = False
        await asyncio.sleep(3)


async def _accept(dev, listener, stop):
    while not stop.is_set():
        accept_task = asyncio.ensure_future(listener.accept())
        stop_task = asyncio.ensure_future(stop.wait())
        _done, pending = await asyncio.wait(
            {accept_task, stop_task},
            return_when=asyncio.FIRST_COMPLETED,
        )
        for item in pending:
            item.cancel()
        for item in pending:
            try:
                await item
            except (asyncio.CancelledError, Exception):
                pass
        if stop.is_set():
            return
        error = accept_task.exception()
        if error is not None:
            print("accept: %s" % error, file=sys.stderr)
            await asyncio.sleep(1)
            continue
        asyncio.create_task(_handle(dev, accept_task.result()))


async def _handle(dev, stream):
    try:
        header = await asyncio.wait_for(_read_header(stream), HEADER_TIMEOUT)
        method, path = parse_request(header)
        hostname, dns_name, ips, backend = await _snapshot(dev)
        payload = response_for(method, path, hostname, dns_name, ips, backend)
        await _send_all(stream, payload)
    except (asyncio.TimeoutError, ValueError, ConnectionError, OSError):
        return
    finally:
        await _close_stream(stream)


async def _read_header(stream):
    buf = b""
    while b"\r\n\r\n" not in buf:
        if len(buf) > HEADER_LIMIT:
            raise ValueError("headers too large")
        chunk = await stream.recv()
        if not chunk:
            raise ConnectionError("closed")
        buf += chunk
    return buf.split(b"\r\n\r\n", 1)[0]


async def _send_all(stream, payload):
    view = payload
    while view:
        sent = await stream.send(view)
        if not isinstance(sent, int):
            return
        if sent <= 0:
            raise ConnectionError("send failed")
        view = view[sent:]


async def _snapshot(dev):
    node = await dev.self_node()
    hostname = getattr(node, "hostname", "") or ""
    tailnet = getattr(node, "tailnet", None) or ""
    dns_name = "%s.%s" % (hostname, tailnet) if hostname and tailnet else ""
    ips = await _addresses(dev)
    backend = "Running" if ips else "Offline"
    return hostname, dns_name, ips, backend


async def _addresses(dev):
    found = []
    for getter in (dev.ipv4_addr, dev.ipv6_addr):
        try:
            value = await getter()
        except Exception:
            continue
        if value:
            found.append(str(value))
    return found


async def _close_stream(stream):
    for name in ("close", "aclose"):
        closer = getattr(stream, name, None)
        if closer is None:
            continue
        result = closer()
        if asyncio.iscoroutine(result):
            await result
        return


def _import_tailscale():
    import tailscale
    return tailscale
