import asyncio
import unittest
from unittest.mock import patch

from mxonepeer.httpio import _response
from mxonepeer.node import EXIT_CODE, HEADER_LIMIT, _handle

MAX_BYTES = 64 * 1024 * 1024
_BAD = "包的内容不正确"


class FakeStream(object):
    def __init__(self, incoming):
        self._buf = incoming
        self.sent = bytearray()
        self.recv_calls = 0
        self.bytes_from_recv = 0
        self.requested = []
        self.closed = False

    async def recv(self, n=None):
        self.recv_calls += 1
        self.requested.append(n)
        if n is not None and n > MAX_BYTES:
            raise AssertionError("tried to read %d bytes" % n)
        if not self._buf:
            return b""
        if n is None or n < 0:
            data, self._buf = self._buf, b""
            self.bytes_from_recv += len(data)
            return data
        data, self._buf = self._buf[:n], self._buf[n:]
        self.bytes_from_recv += len(data)
        return data

    async def send(self, data):
        self.sent.extend(data)
        return len(data)

    def close(self):
        self.closed = True


class FakeDev(object):
    async def self_node(self):
        class Node(object):
            hostname = "mxonepeer"
            tailnet = ""

        return Node()

    async def ipv4_addr(self):
        return "100.64.0.1"

    async def ipv6_addr(self):
        return None


async def _fake_snapshot(_dev):
    return ("mxonepeer", "", ["100.64.0.1"], "Running")


def _request(extra_headers):
    lines = ["POST /upgrade HTTP/1.1"]
    lines.extend(extra_headers)
    return ("\r\n".join(lines) + "\r\n\r\n").encode("ascii")


def _run(stream, decide_result=None):
    with patch("mxonepeer.node._snapshot", new=_fake_snapshot):
        if decide_result is None:
            asyncio.run(_handle(FakeDev(), stream))
            return
        with patch("mxonepeer.node.decide", return_value=decide_result):
            asyncio.run(_handle(FakeDev(), stream))


class NodeUpgradeTests(unittest.TestCase):
    def test_missing_content_length_is_400_and_does_not_exit(self):
        stream = FakeStream(_request([]))
        try:
            _run(stream)
        except SystemExit:
            self.fail("missing Content-Length must not raise SystemExit")
        sent = bytes(stream.sent)
        self.assertTrue(sent.startswith(b"HTTP/1.1 400 "))
        self.assertIn(_BAD.encode("utf-8"), sent)

    def test_non_integer_content_length_is_400_and_does_not_exit(self):
        stream = FakeStream(_request(["Content-Length: abc"]))
        try:
            _run(stream)
        except SystemExit:
            self.fail("non-integer Content-Length must not raise SystemExit")
        sent = bytes(stream.sent)
        self.assertTrue(sent.startswith(b"HTTP/1.1 400 "))
        self.assertIn(_BAD.encode("utf-8"), sent)

    def test_oversized_content_length_is_400_without_reading_body(self):
        stream = FakeStream(_request(["Content-Length: %d" % (MAX_BYTES + 1)]))
        try:
            _run(stream)
        except SystemExit:
            self.fail("oversized Content-Length must not raise SystemExit")
        sent = bytes(stream.sent)
        self.assertTrue(sent.startswith(b"HTTP/1.1 400 "))
        self.assertIn(_BAD.encode("utf-8"), sent)
        self.assertEqual(stream.recv_calls, 1)
        self.assertLess(stream.bytes_from_recv, HEADER_LIMIT)
        for size in stream.requested:
            if size is not None and size > MAX_BYTES:
                self.fail("tried to read %s bytes" % size)

    def test_accept_sends_complete_204_before_exit(self):
        stream = FakeStream(_request(["Content-Length: 0"]))
        expected = _response(204, "No Content", "text/plain", b"", False)
        try:
            _run(stream, decide_result=(204, "", True))
        except SystemExit as err:
            sent = bytes(stream.sent)
            self.assertEqual(err.code, EXIT_CODE)
            self.assertEqual(EXIT_CODE, 75)
            self.assertEqual(sent, expected)
            return
        self.fail("accepted upgrade must raise SystemExit")

    def test_exit_when_send_fails_after_replace(self):
        class SendFails(FakeStream):
            async def send(self, data):
                raise ConnectionError("send failed")

        stream = SendFails(_request(["Content-Length: 0"]))
        with self.assertRaises(SystemExit) as caught:
            _run(stream, decide_result=(204, "", True))
        self.assertEqual(caught.exception.code, 75)


if __name__ == "__main__":
    unittest.main()
