import json

from mxonepeer.keys import write_auth_key

PROTOCOLS = ("2024-11-05", "2025-03-26", "2025-06-18")
PREFERRED = "2025-03-26"
INSTRUCTIONS = (
    "MXOnePeer 的本机登录控制。"
    "status 查看节点是否在线、主机名、Tailscale 地址、auth key 是否已配置，以及账号登录链接。"
    "set_auth_key 保存以 tskey-auth- 开头的密钥。"
    "start_login 开始 Tailscale 账号登录；链接还没返回时，过几秒再调用 status。"
)


def handle(method, headers, body, board, state_dir):
    if method == "GET":
        return 405, "text/plain; charset=utf-8", b"POST required\n", {"Allow": "POST"}
    if method != "POST":
        return 405, "text/plain; charset=utf-8", b"POST required\n", {"Allow": "POST"}
    lowered = {key.lower(): value for key, value in headers.items()}
    if "application/json" not in lowered.get("content-type", "").lower():
        return _rpc_error(415, None, -32600, "application/json required", PREFERRED)
    accept = lowered.get("accept", "")
    if "application/json" not in accept and "text/event-stream" not in accept:
        return _rpc_error(406, None, -32600, "Accept must include application/json or text/event-stream", PREFERRED)
    try:
        message = json.loads(body.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError):
        return _rpc_error(400, None, -32700, "Parse error", PREFERRED)
    if not isinstance(message, dict) or message.get("jsonrpc") != "2.0" or not isinstance(message.get("method"), str):
        return _rpc_error(400, message.get("id") if isinstance(message, dict) else None, -32600, "Invalid Request", PREFERRED)
    rpc_method = message["method"]
    requested = lowered.get("mcp-protocol-version")
    if rpc_method != "initialize" and requested and requested not in PROTOCOLS:
        return _rpc_error(400, message.get("id"), -32600, "Unsupported protocol version", PREFERRED)
    version = _version(message, requested)
    if "id" not in message:
        return 202, "text/plain; charset=utf-8", b"", {"MCP-Protocol-Version": version}
    try:
        result = _dispatch(rpc_method, message, board, state_dir)
    except _Invalid as exc:
        return _rpc_error(200, message.get("id"), -32602, str(exc), version)
    except _Missing as exc:
        return _rpc_error(200, message.get("id"), -32601, str(exc), version)
    except _ToolError as exc:
        result = _tool_text(str(exc), True)
    return _rpc_result(message.get("id"), result, version)


def _dispatch(method, message, board, state_dir):
    if method == "initialize":
        return _initialize(message)
    if method in ("ping", "logging/setLevel"):
        return {}
    if method == "tools/list":
        return _tools()
    if method == "tools/call":
        return _call(message, board, state_dir)
    raise _Missing("Method not found")


def _initialize(message):
    return {
        "protocolVersion": _agree(message),
        "capabilities": {"tools": {"listChanged": False}},
        "serverInfo": {"name": "MXOnePeer", "version": "0.1.0"},
        "instructions": INSTRUCTIONS,
    }


def _tools():
    empty = {"type": "object", "properties": {}, "additionalProperties": False}
    key = {
        "type": "object",
        "properties": {"auth_key": {"type": "string"}},
        "required": ["auth_key"],
        "additionalProperties": False,
    }
    return {
        "tools": [
            {"name": "status", "description": "查看本机节点是否在线、主机名、Tailscale 地址、auth key 是否已配置，以及账号登录链接。", "inputSchema": empty},
            {"name": "set_auth_key", "description": "保存 Tailscale auth key。必须以 tskey-auth- 开头。", "inputSchema": key},
            {"name": "start_login", "description": "开始 Tailscale 账号登录。登录链接可能要过几秒才出现，之后再调用 status。", "inputSchema": empty},
        ]
    }


def _call(message, board, state_dir):
    params = message.get("params") or {}
    if not isinstance(params, dict) or not isinstance(params.get("name"), str):
        raise _Invalid("Invalid params")
    name = params["name"]
    arguments = params.get("arguments") or {}
    if not isinstance(arguments, dict):
        raise _Invalid("Invalid params")
    if name == "status":
        board.refresh_key(state_dir)
        return _tool_text(json.dumps(_status(board), ensure_ascii=False), False)
    if name == "set_auth_key":
        try:
            write_auth_key(state_dir, arguments.get("auth_key", ""))
        except ValueError as exc:
            raise _ToolError("auth key 必须以 tskey-auth- 开头。") from exc
        board.login_error = ""
        board.login_url = ""
        board.notice = "auth key 已保存，正在连接。"
        board.poke("key")
        board.refresh_key(state_dir)
        return _tool_text(json.dumps(_status(board), ensure_ascii=False), False)
    if name == "start_login":
        board.login_error = ""
        board.notice = "正在向 Tailscale 申请登录链接。"
        board.poke("login")
        board.refresh_key(state_dir)
        return _tool_text(json.dumps(_status(board), ensure_ascii=False), False)
    raise _ToolError("Unknown tool")


def _status(board):
    return {
        "online": board.online,
        "hostname": board.hostname,
        "tailscaleIPs": list(board.ips),
        "authKeyConfigured": board.configured,
        "authKeyMasked": board.masked,
        "loginUrl": board.login_url,
        "loginError": board.login_error,
        "notice": board.notice,
        "mcpUrl": "http://127.0.0.1:%d/mcp" % board.admin_port,
    }


def _tool_text(text, is_error):
    return {"content": [{"type": "text", "text": text}], "isError": is_error}


def _agree(message):
    params = message.get("params") or {}
    version = params.get("protocolVersion") if isinstance(params, dict) else None
    if version in PROTOCOLS:
        return version
    return PREFERRED


def _version(message, requested):
    if message.get("method") == "initialize":
        return _agree(message)
    if requested in PROTOCOLS:
        return requested
    return PREFERRED


def _rpc_result(request_id, result, version):
    payload = json.dumps({"jsonrpc": "2.0", "id": request_id, "result": result}, ensure_ascii=False).encode("utf-8")
    return 200, "application/json", payload, {"MCP-Protocol-Version": version}


def _rpc_error(status, request_id, code, message, version):
    body = {"jsonrpc": "2.0", "id": request_id, "error": {"code": code, "message": message}}
    payload = json.dumps(body, ensure_ascii=False).encode("utf-8")
    return status, "application/json", payload, {"MCP-Protocol-Version": version}


class _Invalid(Exception):
    pass


class _Missing(Exception):
    pass


class _ToolError(Exception):
    pass
