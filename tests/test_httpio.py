import unittest

from mxgrokbot.httpio import parse_request, response_for


class HttpTests(unittest.TestCase):
    def test_get_status(self):
        method, path = parse_request(b"GET /status?x=1 HTTP/1.1\r\nHost: mx\r\n\r\n")
        body = response_for(method, path, "mxgrokbot", "mxgrokbot.example.ts.net", ["100.64.0.1"], "Running")
        head, payload = body.split(b"\r\n\r\n", 1)
        self.assertTrue(head.startswith(b"HTTP/1.1 200 OK"))
        self.assertIn(b"Content-Type: application/json", head)
        self.assertIn(b'"online": true', payload)
        self.assertIn(b"100.64.0.1", payload)

    def test_post_status_is_method_not_allowed(self):
        body = response_for("POST", "/status", "mxgrokbot", "", [], "Offline")
        self.assertTrue(body.startswith(b"HTTP/1.1 405 "))
        self.assertIn(b"Allow: GET", body)

    def test_other_path_is_not_found(self):
        body = response_for("GET", "/other", "mxgrokbot", "", [], "Offline")
        self.assertTrue(body.startswith(b"HTTP/1.1 404 "))


if __name__ == "__main__":
    unittest.main()
