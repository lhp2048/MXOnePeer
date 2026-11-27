import asyncio
from urllib.parse import urlsplit

PROXY_PORT = 18021
ADMIN_PORT = 8788
TARGET_HEADER = "x-mx-target-port"
HEADER_LIMIT = 65536
HEADER_TIMEOUT = 5
RESPONSE_TIMEOUT = 30

_HOP = {
    "connection",
    "keep-alive",
    "proxy-connection",
    "transfer-encoding",
    "upgrade",
    "te",
    "trailer",
}


def check_port(value):
    try:
        port = int(str(value).strip())
    except (TypeError, ValueError):
        return None, "端口不正确"
    if port < 1 or port > 65535:
        return None, "端口不正确"
    if port in (PROXY_PORT, ADMIN_PORT):
        return None, "这个端口不能打开"
    return port, ""


def split_head(buf):
    head, sep, rest = buf.partition(b"\r\n\r\n")
    if not sep:
        return None
    return head, rest


def parse_headers(head):
    text = head.decode("iso-8859-1")
    lines = text.split("\r\n")
    if not lines or not lines[0]:
        raise ValueError("bad request")
    headers = []
    for line in lines[1:]:
        if not line:
            continue
        name, sep, value = line.partition(":")
        if not sep:
            continue
        headers.append((name, value.strip()))
    return lines[0], headers


def header_value(headers, name):
    wanted = name.lower()
    for key, value in headers:
        if key.lower() == wanted:
            return value
    return ""


def content_length(headers):
    raw = header_value(headers, "content-length")
    if raw == "":
        return 0
    try:
        length = int(raw)
    except ValueError:
        return None
    if length < 0:
        return None
    return length


def forward_request(request_line, headers, port):
    kept = ["Host: 127.0.0.1:%d" % port, "Connection: close"]
    for name, value in headers:
        lower = name.lower()
        if lower == TARGET_HEADER or lower == "host" or lower in _HOP:
            continue
        kept.append("%s: %s" % (name, value))
    return (request_line + "\r\n" + "\r\n".join(kept) + "\r\n\r\n").encode("iso-8859-1")


def rewrite_response_head(head):
    text = head.decode("iso-8859-1")
    lines = text.split("\r\n")
    if not lines:
        return head + b"\r\n\r\n"
    out = [lines[0]]
    for line in lines[1:]:
        if not line:
            continue
        name, sep, value = line.partition(":")
        if not sep:
            out.append(line)
            continue
        if name.lower() == "connection":
            continue
        if name.lower() == "location":
            line = "Location: %s" % relative_location(value.strip())
        out.append(line)
    out.append("Connection: close")
    return ("\r\n".join(out) + "\r\n\r\n").encode("iso-8859-1")


def relative_location(value):
    parsed = urlsplit(value)
    if parsed.hostname not in ("127.0.0.1", "localhost", "::1"):
        return value
    path = parsed.path or "/"
    if parsed.query:
        path += "?" + parsed.query
    return path


def error_response(status, message):
    body = (message + "\n").encode("utf-8")
    reason = {400: "Bad Request", 502: "Bad Gateway"}.get(status, "OK")
    head = "HTTP/1.1 %d %s\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: %d\r\nConnection: close\r\n\r\n" % (
        status,
        reason,
        len(body),
    )
    return head.encode("ascii") + body


async def handle_proxy(stream):
    try:
        buf = await asyncio.wait_for(_read_until(stream, b"\r\n\r\n", HEADER_LIMIT), HEADER_TIMEOUT)
        parts = split_head(buf)
        if parts is None:
            return
        head, rest = parts
        request, headers = parse_headers(head)
    except (asyncio.TimeoutError, ValueError, ConnectionError, OSError):
        return
    port, message = check_port(header_value(headers, TARGET_HEADER))
    if message:
        await _send_all(stream, error_response(400, message))
        return
    length = content_length(headers)
    if length is None:
        await _send_all(stream, error_response(400, "端口不正确"))
        return
    try:
        body = await _read_body(stream, rest, length)
        payload = forward_request(request, headers, port) + body
        reader, writer = await asyncio.wait_for(asyncio.open_connection("127.0.0.1", port), HEADER_TIMEOUT)
    except (asyncio.TimeoutError, ConnectionError, OSError):
        try:
            await _send_all(stream, error_response(502, "打不开这个端口"))
        except (ConnectionError, OSError):
            return
        return
    try:
        writer.write(payload)
        await writer.drain()
        if writer.can_write_eof():
            writer.write_eof()
        response = await asyncio.wait_for(_read_until_reader(reader, b"\r\n\r\n", HEADER_LIMIT), RESPONSE_TIMEOUT)
        resp_parts = split_head(response)
        if resp_parts is None:
            return
        resp_head, resp_rest = resp_parts
        await _send_all(stream, rewrite_response_head(resp_head) + resp_rest)
        while True:
            chunk = await reader.read(65536)
            if not chunk:
                break
            await _send_all(stream, chunk)
    except (asyncio.TimeoutError, ConnectionError, OSError):
        return
    finally:
        writer.close()
        wait = getattr(writer, "wait_closed", None)
        if wait is not None:
            try:
                await wait()
            except (ConnectionError, OSError):
                return


async def _read_body(stream, rest, length):
    buf = rest
    while len(buf) < length:
        chunk = await stream.recv()
        if not chunk:
            raise ConnectionError("closed")
        buf += chunk
    return buf[:length]


async def _read_until(stream, marker, limit):
    buf = b""
    while marker not in buf:
        if len(buf) > limit:
            raise ValueError("headers too large")
        chunk = await stream.recv()
        if not chunk:
            raise ConnectionError("closed")
        buf += chunk
    return buf


async def _read_until_reader(reader, marker, limit):
    buf = b""
    while marker not in buf:
        if len(buf) > limit:
            raise ValueError("headers too large")
        chunk = await reader.read(65536)
        if not chunk:
            raise ConnectionError("closed")
        buf += chunk
    return buf


async def _send_all(stream, payload):
    view = payload
    while view:
        sent = await stream.send(view)
        if not isinstance(sent, int) or sent <= 0:
            raise ConnectionError("send failed")
        view = view[sent:]
