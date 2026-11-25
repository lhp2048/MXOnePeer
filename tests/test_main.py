import unittest

from mxgrokbot.main import main


class MainTests(unittest.TestCase):
    def test_admin_host_must_be_loopback(self):
        self.assertEqual(main(["--admin-host", "0.0.0.0"]), 1)


if __name__ == "__main__":
    unittest.main()
