from mxonepeer.status import dumps


def parse_request(header):
    line = header.split(b"\r\n", 1)[0].decode("latin1")
    parts = line.split(" ")
    if len(parts) != 3 or not parts[2].startswith("HTTP/"):
        raise ValueError("bad request")
    path = parts[1].split("?", 1)[0]
    return parts[0], path


def response_for(method, path, hostname, dns_name, ips, backend_state, runtime=""):
    if method == "GET" and path == "/status":
        body = dumps(hostname, dns_name, ips, backend_state, runtime=runtime)
        return _response(200, "OK", "application/json", body, allow=False)
    if path == "/status":
        return _response(405, "Method Not Allowed", "text/plain", b"Method Not Allowed\n", allow=True)
    return _response(404, "Not Found", "text/plain", b"Not Found\n", allow=False)


def _response(status, reason, content_type, body, allow):
    lines = [
        "HTTP/1.1 %d %s" % (status, reason),
        "Content-Type: %s" % content_type,
        "Content-Length: %d" % len(body),
        "Connection: close",
    ]
    if allow:
        lines.append("Allow: GET")
    head = ("\r\n".join(lines) + "\r\n\r\n").encode("ascii")
    return head + body
