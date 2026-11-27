import asyncio
import os
import sys

from mxonepeer.httpio import _response, parse_request, response_for
from mxonepeer.portproxy import PROXY_PORT, handle_proxy
from mxonepeer.upgrade import EXIT_CODE, decide, runtime_id

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
        listeners.append((await dev.tcp_listen((address, port)), lambda stream: _handle(dev, stream)))
        listeners.append((await dev.tcp_listen((address, PROXY_PORT)), handle_proxy))
    tasks = [asyncio.create_task(_accept(listener, stop, handle)) for listener, handle in listeners]
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


async def _accept(listener, stop, handle):
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
        asyncio.create_task(handle(accept_task.result()))


async def _handle(dev, stream):
    should_exit = False
    try:
        head, rest = await asyncio.wait_for(_read_message(stream), HEADER_TIMEOUT)
        method, path = parse_request(head)
        if path == "/upgrade":
            headers = _header_map(head)
            try:
                body = await _read_body(stream, rest, headers)
            except ValueError as err:
                if str(err) != "包的内容不正确":
                    raise
                payload = _response(400, "Bad Request", "text/plain", "包的内容不正确\n".encode("utf-8"), False)
                await _send_all(stream, payload)
                return
            status, text, should_exit = decide(
                method,
                headers,
                body,
                runtime_id(),
                os.path.dirname(os.path.abspath(__file__)),
            )
            payload = _upgrade_payload(status, text)
        else:
            hostname, dns_name, ips, backend = await _snapshot(dev)
            payload = response_for(method, path, hostname, dns_name, ips, backend, runtime=runtime_id())
        await _send_all(stream, payload)
    except (asyncio.TimeoutError, ValueError, ConnectionError, OSError):
        pass
    finally:
        try:
            await _close_stream(stream)
        except (ConnectionError, OSError):
            pass
    if should_exit:
        sys.exit(EXIT_CODE)


def _header_map(head):
    headers = {}
    for line in head.split(b"\r\n")[1:]:
        if b":" not in line:
            continue
        name, value = line.split(b":", 1)
        headers[name.decode("latin1").strip().lower()] = value.decode("latin1").strip()
    return headers


async def _read_message(stream):
    buf = b""
    while b"\r\n\r\n" not in buf:
        if len(buf) > HEADER_LIMIT:
            raise ValueError("headers too large")
        chunk = await stream.recv()
        if not chunk:
            raise ConnectionError("closed")
        buf += chunk
    return buf.split(b"\r\n\r\n", 1)


async def _read_body(stream, rest, headers):
    try:
        length = int(headers.get("content-length", ""))
    except ValueError:
        raise ValueError("包的内容不正确")
    if length < 0 or length > 64 * 1024 * 1024:
        raise ValueError("包的内容不正确")
    buf = bytearray(rest)
    while len(buf) < length:
        chunk = await stream.recv()
        if not chunk:
            raise ValueError("包的内容不正确")
        buf += chunk
    return bytes(buf[:length])


def _upgrade_payload(status, text):
    if status == 204:
        return _response(204, "No Content", "text/plain", b"", False)
    reason = {
        400: "Bad Request",
        405: "Method Not Allowed",
        500: "Internal Server Error",
    }.get(status, "OK")
    return _response(status, reason, "text/plain", (text + "\n").encode("utf-8"), False)


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
