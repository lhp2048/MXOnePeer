import json
import unittest

from mxonepeer.status import document, dumps


class StatusTests(unittest.TestCase):
    def test_running_orders_ipv4_first_and_uses_given_name(self):
        got = document(
            "mxonepeer-1",
            "mxonepeer.example.ts.net.",
            ["fd7a:115c:a1e0::1", "100.64.0.1"],
            "Running",
        )
        self.assertTrue(got["online"])
        self.assertEqual(got["hostname"], "mxonepeer-1")
        self.assertEqual(got["dnsName"], "mxonepeer.example.ts.net")
        self.assertEqual(got["tailscaleIPs"], ["100.64.0.1", "fd7a:115c:a1e0::1"])
        self.assertEqual(got["backendState"], "Running")

    def test_running_without_address_is_offline(self):
        got = document("mxonepeer", "", [], "Running")
        self.assertFalse(got["online"])
        self.assertEqual(got["tailscaleIPs"], [])

    def test_not_running_is_offline(self):
        got = document("mxonepeer", "", ["100.64.0.1"], "Offline")
        self.assertFalse(got["online"])

    def test_empty_dns_name_stays_in_json(self):
        raw = json.loads(dumps("mxonepeer", "", [], "Offline"))
        self.assertEqual(raw["dnsName"], "")
        self.assertIn("dnsName", raw)
        self.assertIs(raw["online"], False)

    def test_runtime_is_included_when_given(self):
        got = document("mxonepeer", "", ["100.64.0.1"], "Running", runtime="python/linux/amd64")
        self.assertEqual(got["runtime"], "python/linux/amd64")

    def test_runtime_is_absent_when_empty(self):
        raw = json.loads(dumps("mxonepeer", "", [], "Offline"))
        self.assertNotIn("runtime", raw)


if __name__ == "__main__":
    unittest.main()
