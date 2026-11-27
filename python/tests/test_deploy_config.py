import os
import unittest

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))


def _read(*parts):
    with open(os.path.join(ROOT, *parts), encoding="utf-8") as handle:
        return handle.read()


class DeployConfigTests(unittest.TestCase):
    def test_systemd_restarts_on_any_exit(self):
        for path in (("go", "deploy", "mxonepeer.service"), ("python", "deploy", "mxonepeer.service")):
            text = _read(*path)
            self.assertIn("Restart=always", text)
            self.assertNotIn("Restart=on-failure", text)
            self.assertIn("RestartSec=5", text)

    def test_linux_program_is_owned_by_service_user(self):
        go_install = _read("go", "deploy", "install.sh")
        self.assertIn("install -d -m 0755 -o mxonepeer -g mxonepeer /opt/mxonepeer", go_install)
        self.assertIn('-o mxonepeer -g mxonepeer "$bin" /opt/mxonepeer/mxonepeer', go_install)
        python_install = _read("python", "deploy", "install.sh")
        self.assertIn("install -d -m 0755 -o mxonepeer -g mxonepeer /opt/mxonepeer", python_install)
        self.assertIn("chown -R mxonepeer:mxonepeer /opt/mxonepeer/mxonepeer", python_install)
        self.assertNotIn("chown -R root:root /opt/mxonepeer/mxonepeer", python_install)

    def test_launchagent_restarts_after_exit(self):
        for path in (("go", "deploy", "install-macos.sh"), ("python", "deploy", "install-macos.sh")):
            text = _read(*path)
            self.assertIn("<key>KeepAlive</key>\n  <true/>", text)
            self.assertNotIn("SuccessfulExit", text)

    def test_windows_runner_loops(self):
        for path in (("go", "deploy", "install-windows.ps1"), ("python", "deploy", "install-windows.ps1")):
            text = _read(*path)
            self.assertIn("goto loop", text)
            self.assertIn("ping -n 6 127.0.0.1", text)
        go_windows = _read("go", "deploy", "install-windows.ps1")
        self.assertIn("-Execute $runner", go_windows)
        self.assertNotIn("-Execute $installed", go_windows)


if __name__ == "__main__":
    unittest.main()
