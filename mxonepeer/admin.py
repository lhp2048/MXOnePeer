import asyncio
from urllib.parse import parse_qs

from mxonepeer.keys import mask_auth_key, read_auth_key, write_auth_key
from mxonepeer.mcp import handle as handle_mcp

HEADER_LIMIT = 8192
BODY_LIMIT = 4096


class Board:
    def __init__(self, token):
        self.token = token
        self.login_url = ""
        self.login_error = ""
        self.notice = ""
        self.configured = False
        self.masked = ""
        self.online = False
        self.hostname = ""
        self.ips = []
        self.action = ""
        self.generation = 0
        self.admin_port = 8788
        self.wake = asyncio.Event()

    def poke(self, action):
        self.action = action
        self.generation += 1
        self.wake.set()

    def refresh_key(self, state_dir):
        key = read_auth_key(state_dir)
        self.configured = bool(key)
        self.masked = mask_auth_key(key)


def page(board):
    online = board.online
    pill = "on" if online else "off"
    state = "已连接" if online else "未连接"
    ips = _esc(", ".join(board.ips)) if board.ips else "还没有"
    host = _esc(board.hostname) if board.hostname else "还没有"
    key_state = "已配置 %s" % _esc(board.masked) if board.configured else "未配置"
    notice = '<p class="banner">%s</p>' % _esc(board.notice) if board.notice else ""
    error = '<p class="banner bad">%s</p>' % _esc(board.login_error) if board.login_error else ""
    if board.login_url:
        url = _esc(board.login_url)
        login = '<a class="link" href="%s">打开 Tailscale 登录链接</a><p class="url">%s</p>' % (url, url)
    else:
        login = '<p class="hint">批准之后，这台机器会出现在你的 Tailscale 网络里。</p>'
    mcp = _esc("http://127.0.0.1:%d/mcp" % board.admin_port)
    return """<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>MXOnePeer</title>
<style>
:root { color-scheme: light; }
body { margin: 0; min-height: 100vh; background: #f3efe6; color: #1c1915; font-family: "Avenir Next", "Segoe UI", sans-serif; }
.shell { max-width: 40rem; margin: 0 auto; padding: 2.5rem 1.25rem 3rem; }
header { display: flex; align-items: center; gap: 0.9rem; }
.mark { width: 2.7rem; height: 2.7rem; margin: 0; border-radius: 0.8rem; background: #0f6e6e; color: #f7f3ea; display: grid; place-items: center; font-weight: 700; letter-spacing: 0.04em; }
h1 { margin: 0; font-size: 1.45rem; letter-spacing: -0.03em; }
.sub { margin: 0.15rem 0 0; color: #625c53; font-size: 0.92rem; }
.pill { margin-left: auto; border-radius: 999px; padding: 0.28rem 0.7rem; font-size: 0.82rem; font-weight: 650; }
.on { background: #e5f4ea; color: #146c43; }
.off { background: #f3e6e1; color: #8a3b2d; }
.facts { display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.7rem; margin: 1.3rem 0; }
.fact { background: #fffdf8; border: 1px solid #e4ddd0; border-radius: 0.9rem; padding: 0.8rem 0.9rem; }
.fact span { display: block; color: #625c53; font-size: 0.78rem; }
.fact strong { display: block; margin-top: 0.25rem; font-size: 0.95rem; overflow-wrap: anywhere; }
.card { background: #fffdf8; border: 1px solid #e4ddd0; border-radius: 1rem; padding: 1.1rem 1.15rem 1.2rem; margin-top: 0.8rem; }
h2 { margin: 0 0 0.35rem; font-size: 1.05rem; }
.card p { margin: 0.35rem 0 0; color: #4e4942; }
label { display: block; margin-top: 0.9rem; font-size: 0.86rem; color: #4e4942; }
input[type=password] { width: 100%%; box-sizing: border-box; margin-top: 0.35rem; padding: 0.7rem 0.75rem; border-radius: 0.7rem; border: 1px solid #d9d0c2; background: #fff; color: #1c1915; font: inherit; }
button { margin-top: 0.9rem; border: 0; border-radius: 0.7rem; padding: 0.65rem 0.95rem; font: inherit; font-weight: 650; cursor: pointer; }
.primary { background: #0f6e6e; color: #f7f3ea; }
.quiet { background: #efeae1; color: #1c1915; }
.banner { margin-top: 0.8rem; padding: 0.65rem 0.75rem; border-radius: 0.7rem; background: #e7f3f3; color: #0f4f4f; }
.bad { background: #f8ebe6; color: #8a3b2d; }
.link { display: inline-block; margin-top: 0.8rem; color: #0f6e6e; font-weight: 700; }
.url, .mcp { overflow-wrap: anywhere; font-family: ui-monospace, "SFMono-Regular", monospace; font-size: 0.86rem; }
footer { margin-top: 1.1rem; color: #625c53; font-size: 0.86rem; }
.hint { color: #625c53; }
@media (max-width: 640px) { .facts { grid-template-columns: 1fr; } header { align-items: flex-start; } .pill { margin-top: 0.2rem; } }
</style>
</head>
<body>
<main class="shell">
<header>
<p class="mark">MX</p>
<div><h1>MXOnePeer</h1><p class="sub">本机登录控制</p></div>
<span class="pill %s">%s</span>
</header>
<section class="facts">
<div class="fact"><span>主机名</span><strong>%s</strong></div>
<div class="fact"><span>Tailscale 地址</span><strong>%s</strong></div>
<div class="fact"><span>auth key</span><strong>%s</strong></div>
</section>
%s%s
<section class="card">
<h2>auth key</h2>
<p>从 Tailscale 后台复制以 tskey-auth- 开头的密钥。保存后只显示末四位。</p>
<form method="post" action="/authkey">
<input type="hidden" name="token" value="%s">
<label>auth key<input type="password" name="authkey" autocomplete="off"></label>
<button class="primary" type="submit">保存 auth key</button>
</form>
</section>
<section class="card">
<h2>账号登录</h2>
<p>没有密钥时，用你的 Tailscale 账号批准这台设备。</p>
<form method="post" action="/login">
<input type="hidden" name="token" value="%s">
<button class="quiet" type="submit">用 Tailscale 账号登录</button>
</form>
%s
</section>
<footer>本地 MCP <span class="mcp">%s</span></footer>
</main>
</body>
</html>
""" % (pill, state, host, ips, key_state, notice, error, _esc(board.token), _esc(board.token), login, mcp)


def decide(method, path, body, board, state_dir):
    if method == "GET" and path == "/":
        board.refresh_key(state_dir)
        return 200, "text/html; charset=utf-8", page(board).encode("utf-8")
    if method != "POST" or path not in ("/authkey", "/login"):
        return 404, "text/plain; charset=utf-8", b"Not Found\n"
    form = parse_qs(body.decode("utf-8", "replace"), keep_blank_values=True)
    if form.get("token", [""])[0] != board.token:
        return 403, "text/plain; charset=utf-8", b"Forbidden\n"
    if path == "/login":
        board.login_error = ""
        board.notice = "正在向 Tailscale 申请登录链接。"
        board.poke("login")
        board.refresh_key(state_dir)
        return 303, "text/plain; charset=utf-8", b""
    try:
        write_auth_key(state_dir, form.get("authkey", [""])[0])
    except ValueError:
        board.login_error = "auth key 必须以 tskey-auth- 开头。"
        board.refresh_key(state_dir)
        return 400, "text/html; charset=utf-8", page(board).encode("utf-8")
    board.login_error = ""
    board.login_url = ""
    board.notice = "auth key 已保存，正在连接。"
    board.poke("key")
    board.refresh_key(state_dir)
    return 303, "text/plain; charset=utf-8", b""


def extract_login_url(state):
    if state is None:
        return ""
    url = getattr(state, "url", None)
    if isinstance(url, str) and url.startswith("http"):
        return url
    for part in str(state).replace("'", " ").replace('"', " ").replace("(", " ").replace(")", " ").split():
        if part.startswith("http://") or part.startswith("https://"):
            return part
    return ""


def authorized(state):
    if state is True:
        return True
    text = "%s %s" % (type(state).__name__, state)
    if "NotAuthorized" in text or extract_login_url(state):
        return False
    return "Authorized" in text


async def serve_admin(host, port, board, state_dir):
    async def client(reader, writer):
        try:
            await _client(reader, writer, board, state_dir)
        except (asyncio.TimeoutError, ValueError, ConnectionError, asyncio.IncompleteReadError):
            writer.close()

    server = await asyncio.start_server(client, host, port)
    board.admin_port = server.sockets[0].getsockname()[1]
    return server


async def _client(reader, writer, board, state_dir):
    header, rest = await asyncio.wait_for(_read_header(reader), 5)
    method, path = _request_line(header)
    length = _content_length(header)
    body = rest
    if len(body) < length:
        body += await asyncio.wait_for(reader.readexactly(length - len(body)), 5)
    elif len(body) > length:
        body = body[:length]
    if path == "/mcp":
        status, content_type, payload, extra = handle_mcp(method, _header_map(header), body, board, state_dir)
    else:
        status, content_type, payload = decide(method, path, body, board, state_dir)
        extra = {}
    lines = [
        "HTTP/1.1 %d %s" % (status, _reason(status)),
        "Content-Type: %s" % content_type,
        "Content-Length: %d" % len(payload),
        "Connection: close",
    ]
    if status == 303:
        lines.append("Location: /")
    for name, value in extra.items():
        lines.append("%s: %s" % (name, value))
    writer.write(("\r\n".join(lines) + "\r\n\r\n").encode("ascii") + payload)
    await writer.drain()
    writer.close()


def _request_line(header):
    line = header.split(b"\r\n", 1)[0].decode("latin1")
    method, target, version = line.split(" ")
    if not version.startswith("HTTP/"):
        raise ValueError("bad request")
    return method, target.split("?", 1)[0]


def _content_length(header):
    for line in header.split(b"\r\n")[1:]:
        if line.lower().startswith(b"content-length:"):
            length = int(line.split(b":", 1)[1].strip())
            if length < 0 or length > BODY_LIMIT:
                raise ValueError("body too large")
            return length
    return 0


async def _read_header(reader):
    buf = b""
    while b"\r\n\r\n" not in buf:
        if len(buf) > HEADER_LIMIT:
            raise ValueError("headers too large")
        chunk = await reader.read(1024)
        if not chunk:
            raise ConnectionError("closed")
        buf += chunk
    return buf.split(b"\r\n\r\n", 1)


def _header_map(header):
    found = {}
    for line in header.split(b"\r\n")[1:]:
        if b":" not in line:
            continue
        name, value = line.split(b":", 1)
        found[name.decode("latin1").strip().lower()] = value.decode("latin1").strip()
    return found


def _reason(status):
    return {
        200: "OK",
        202: "Accepted",
        303: "See Other",
        400: "Bad Request",
        403: "Forbidden",
        404: "Not Found",
        405: "Method Not Allowed",
        406: "Not Acceptable",
        415: "Unsupported Media Type",
    }.get(status, "OK")


def _esc(text):
    return (
        str(text)
        .replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
        .replace('"', "&quot;")
    )
