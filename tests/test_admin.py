import asyncio
import json
import os
import stat
import tempfile
import unittest
from urllib.parse import urlencode

from mxonepeer.admin import Board, authorized, decide, extract_login_url, serve_admin
from mxonepeer.keys import normalize_auth_key, read_auth_key, write_auth_key


class KeyTests(unittest.TestCase):
    def test_blank_is_empty(self):
        self.assertEqual(normalize_auth_key("  "), "")

    def test_rejects_other_key_kinds(self):
        with self.assertRaises(ValueError):
            normalize_auth_key("tskey-api-not-a-node-key")

    def test_round_trip_is_private(self):
        with tempfile.TemporaryDirectory() as directory:
            write_auth_key(directory, "tskey-auth-example")
            self.assertEqual(read_auth_key(directory), "tskey-auth-example")
            mode = stat.S_IMODE(os.stat(os.path.join(directory, "authkey")).st_mode)
            self.assertEqual(mode, 0o600)


class PageTests(unittest.TestCase):
    def test_get_shows_both_actions(self):
        board = Board("token-1")
        with tempfile.TemporaryDirectory() as directory:
            status, content_type, body = decide("GET", "/", b"", board, directory)
        self.assertEqual(status, 200)
        self.assertIn("text/html", content_type)
        text = body.decode("utf-8")
        self.assertIn("保存 auth key", text)
        self.assertIn("用 Tailscale 账号登录", text)
        self.assertIn("未配置", text)
        self.assertIn("本地 MCP", text)
        self.assertIn("http://127.0.0.1:8788/mcp", text)

    def test_post_without_token_is_forbidden(self):
        board = Board("token-1")
        with tempfile.TemporaryDirectory() as directory:
            status, _kind, _body = decide("POST", "/authkey", b"authkey=tskey-auth-example", board, directory)
        self.assertEqual(status, 403)
        self.assertEqual(board.generation, 0)

    def test_post_saves_key_and_wakes(self):
        board = Board("token-1")
        body = urlencode({"token": "token-1", "authkey": "tskey-auth-example"}).encode("utf-8")
        with tempfile.TemporaryDirectory() as directory:
            status, _kind, _body = decide("POST", "/authkey", body, board, directory)
            self.assertEqual(read_auth_key(directory), "tskey-auth-example")
        self.assertEqual(status, 303)
        self.assertEqual(board.action, "key")
        self.assertTrue(board.configured)

    def test_login_requests_account_flow(self):
        board = Board("token-1")
        body = urlencode({"token": "token-1"}).encode("utf-8")
        with tempfile.TemporaryDirectory() as directory:
            status, _kind, _body = decide("POST", "/login", body, board, directory)
        self.assertEqual(status, 303)
        self.assertEqual(board.action, "login")

    def test_bad_key_stays_on_the_page(self):
        board = Board("token-1")
        body = urlencode({"token": "token-1", "authkey": "nope"}).encode("utf-8")
        with tempfile.TemporaryDirectory() as directory:
            status, content_type, payload = decide("POST", "/authkey", body, board, directory)
            self.assertFalse(os.path.exists(os.path.join(directory, "authkey")))
        self.assertEqual(status, 400)
        self.assertIn("text/html", content_type)
        self.assertIn("tskey-auth-", payload.decode("utf-8"))


class LoginUrlTests(unittest.TestCase):
    def test_reads_url_attribute(self):
        class State:
            url = "https://login.tailscale.com/a/abc"

        self.assertEqual(extract_login_url(State()), "https://login.tailscale.com/a/abc")
        self.assertFalse(authorized(State()))

    def test_reads_url_from_text(self):
        self.assertEqual(
            extract_login_url("NotAuthorized(https://login.tailscale.com/a/abc)"),
            "https://login.tailscale.com/a/abc",
        )
        self.assertTrue(authorized("Authorized"))


class ServerTests(unittest.TestCase):
    def test_loopback_page(self):
        asyncio.run(self._page())

    async def _page(self):
        board = Board("token-1")
        with tempfile.TemporaryDirectory() as directory:
            server = await serve_admin("127.0.0.1", 0, board, directory)
            port = server.sockets[0].getsockname()[1]
            reader, writer = await asyncio.open_connection("127.0.0.1", port)
            writer.write(b"GET / HTTP/1.1\r\nHost: 127.0.0.1\r\n\r\n")
            await writer.drain()
            data = await reader.read()
            writer.close()
            server.close()
            await server.wait_closed()
        self.assertIn(b"HTTP/1.1 200", data)
        self.assertIn("保存 auth key".encode("utf-8"), data)
        self.assertIn("用 Tailscale 账号登录".encode("utf-8"), data)

    def test_post_body_arrives_with_headers(self):
        asyncio.run(self._post())

    async def _post(self):
        board = Board("token-1")
        with tempfile.TemporaryDirectory() as directory:
            server = await serve_admin("127.0.0.1", 0, board, directory)
            port = server.sockets[0].getsockname()[1]
            payload = urlencode({"token": "token-1", "authkey": "not-a-key"}).encode("utf-8")
            request = b"POST /authkey HTTP/1.1\r\nHost: 127.0.0.1\r\nContent-Length: %d\r\n\r\n" % len(payload)
            reader, writer = await asyncio.open_connection("127.0.0.1", port)
            writer.write(request + payload)
            await writer.drain()
            data = await asyncio.wait_for(reader.read(), 2)
            writer.close()
            server.close()
            await server.wait_closed()
        self.assertIn(b"HTTP/1.1 400", data)
        self.assertIn("必须以 tskey-auth-".encode("utf-8"), data)

    def test_local_mcp_lists_login_tools_and_hides_the_key(self):
        asyncio.run(self._mcp())

    async def _mcp(self):
        board = Board("token-1")
        with tempfile.TemporaryDirectory() as directory:
            server = await serve_admin("127.0.0.1", 0, board, directory)
            port = server.sockets[0].getsockname()[1]
            listed = await _rpc(port, {"jsonrpc": "2.0", "id": 1, "method": "tools/list"})
            names = [item["name"] for item in listed["result"]["tools"]]
            saved = await _rpc(port, {
                "jsonrpc": "2.0",
                "id": 2,
                "method": "tools/call",
                "params": {"name": "set_auth_key", "arguments": {"auth_key": "tskey-auth-example"}},
            })
            text = saved["result"]["content"][0]["text"]
            server.close()
            await server.wait_closed()
            self.assertEqual(read_auth_key(directory), "tskey-auth-example")
        self.assertEqual(names, ["status", "set_auth_key", "start_login"])
        self.assertNotIn("tskey-auth-example", text)
        self.assertIn("…mple", text)
        self.assertEqual(board.action, "key")


async def _rpc(port, message):
    payload = json.dumps(message).encode("utf-8")
    header = (
        "POST /mcp HTTP/1.1\r\nHost: 127.0.0.1\r\n"
        "Content-Type: application/json\r\n"
        "Accept: application/json, text/event-stream\r\n"
        "Content-Length: %d\r\n\r\n" % len(payload)
    ).encode("ascii")
    reader, writer = await asyncio.open_connection("127.0.0.1", port)
    writer.write(header + payload)
    await writer.drain()
    data = await reader.read()
    writer.close()
    self_status, body = data.split(b"\r\n\r\n", 1)
    if not self_status.startswith(b"HTTP/1.1 200"):
        raise AssertionError(self_status)
    return json.loads(body)


if __name__ == "__main__":
    unittest.main()
