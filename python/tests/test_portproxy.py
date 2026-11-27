import asyncio
import unittest

from mxonepeer.portproxy import (
    check_port,
    forward_request,
    handle_proxy,
    parse_headers,
    relative_location,
)


class PortTests(unittest.TestCase):
    def test_rejects_admin_and_proxy_ports(self):
        self.assertEqual(check_port("8080")[0], 8080)
        self.assertEqual(check_port("8788")[1], "这个端口不能打开")
        self.assertEqual(check_port("18021")[1], "这个端口不能打开")
        self.assertEqual(check_port("nope")[1], "端口不正确")

    def test_forward_sets_loopback_host(self):
        _request, headers = parse_headers(
            b"GET /main?q=1 HTTP/1.1\r\nHost: peer\r\nX-Mx-Target-Port: 8080\r\nAccept: text/html"
        )
        payload = forward_request("GET /main?q=1 HTTP/1.1", headers, 8080)
        text = payload.decode("iso-8859-1")
        self.assertIn("Host: 127.0.0.1:8080", text)
        self.assertNotIn("X-Mx-Target-Port", text)
        self.assertIn("GET /main?q=1 HTTP/1.1", text)

    def test_location_becomes_relative(self):
        self.assertEqual(relative_location("http://127.0.0.1:8080/next"), "/next")
        self.assertEqual(relative_location("https://example.com/next"), "https://example.com/next")


class RelayTests(unittest.TestCase):
    def test_relays_page(self):
        asyncio.run(self._relay())

    async def _relay(self):
        async def backend(reader, writer):
            await reader.readuntil(b"\r\n\r\n")
            body = b"page"
            writer.write(b"HTTP/1.1 200 OK\r\nContent-Length: 4\r\nLocation: http://127.0.0.1:9/next\r\n\r\n" + body)
            await writer.drain()
            writer.close()

        server = await asyncio.start_server(backend, "127.0.0.1", 0)
        port = server.sockets[0].getsockname()[1]
        incoming = asyncio.StreamReader()
        incoming.feed_data(
            ("GET /main HTTP/1.1\r\nHost: client\r\nX-Mx-Target-Port: %d\r\n\r\n" % port).encode("ascii")
        )
        incoming.feed_eof()
        sent = bytearray()

        class Stream(object):
            async def recv(self):
                return await incoming.read(65536)

            async def send(self, data):
                sent.extend(data)
                return len(data)

        await handle_proxy(Stream())
        server.close()
        await server.wait_closed()
        text = bytes(sent).decode("iso-8859-1")
        self.assertIn("HTTP/1.1 200 OK", text)
        self.assertIn("Location: /next", text)
        self.assertTrue(text.endswith("page"))


if __name__ == "__main__":
    unittest.main()
