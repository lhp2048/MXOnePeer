import json
import unittest

from mxgrokbot.status import document, dumps


class StatusTests(unittest.TestCase):
    def test_running_orders_ipv4_first_and_uses_given_name(self):
        got = document(
            "mxgrokbot-1",
            "mxgrokbot.example.ts.net.",
            ["fd7a:115c:a1e0::1", "100.64.0.1"],
            "Running",
        )
        self.assertTrue(got["online"])
        self.assertEqual(got["hostname"], "mxgrokbot-1")
        self.assertEqual(got["dnsName"], "mxgrokbot.example.ts.net")
        self.assertEqual(got["tailscaleIPs"], ["100.64.0.1", "fd7a:115c:a1e0::1"])
        self.assertEqual(got["backendState"], "Running")

    def test_running_without_address_is_offline(self):
        got = document("mxgrokbot", "", [], "Running")
        self.assertFalse(got["online"])
        self.assertEqual(got["tailscaleIPs"], [])

    def test_not_running_is_offline(self):
        got = document("mxgrokbot", "", ["100.64.0.1"], "Offline")
        self.assertFalse(got["online"])

    def test_empty_dns_name_stays_in_json(self):
        raw = json.loads(dumps("mxgrokbot", "", [], "Offline"))
        self.assertEqual(raw["dnsName"], "")
        self.assertIn("dnsName", raw)
        self.assertIs(raw["online"], False)


if __name__ == "__main__":
    unittest.main()
