import os
import stat
import tempfile
import unittest

from mxonepeer.hostname import (
    default_hostname,
    ensure_hostname,
    list_linux_interfaces,
    normalize_hostname,
    parse_ifconfig,
    read_hostname,
    select_mac,
    write_hostname,
)


def _iface(name, mac, physical=False, virtual=False, loopback=False):
    return {
        "name": name,
        "mac": mac,
        "physical": physical,
        "virtual": virtual,
        "loopback": loopback,
    }


class SanitizeTests(unittest.TestCase):
    def test_uses_hostname_and_mac_without_colons(self):
        name = default_hostname(
            "Grok-Bot-VM-294200885",
            [_iface("lo", "00:00:00:00:00:00", loopback=True, virtual=True), _iface("enp0s3", "82:2B:38:3D:BC:D0", physical=True)],
        )
        self.assertEqual(name, "grok-bot-vm-294200885-822b383dbcd0")
        self.assertEqual(normalize_hostname(name), name)

    def test_sanitizes_dots_underscores_and_uppercase(self):
        name = default_hostname("My.Host_Name", [_iface("eth0", "aa:bb:cc:dd:ee:ff", physical=True)])
        self.assertEqual(name, "my-host-name-aabbccddeeff")

    def test_strips_invalid_edges_and_collapses_hyphens(self):
        name = default_hostname("--Foo..Bar--", [_iface("eth0", "aa-bb-cc-dd-ee-ff", physical=True)])
        self.assertEqual(name, "foo-bar-aabbccddeeff")

    def test_truncates_to_dns_label_limit(self):
        host = "a" * 80
        name = default_hostname(host, [_iface("eth0", "aa:bb:cc:dd:ee:ff", physical=True)])
        self.assertEqual(len(name), 63)
        self.assertTrue(name.endswith("-aabbccddeeff"))
        self.assertEqual(name, "a" * 50 + "-aabbccddeeff")
        self.assertEqual(normalize_hostname(name), name)

    def test_truncation_does_not_leave_a_trailing_hyphen(self):
        host = "a" * 49 + "-"
        name = default_hostname(host, [_iface("eth0", "aa:bb:cc:dd:ee:ff", physical=True)])
        self.assertEqual(name, "a" * 49 + "-aabbccddeeff")
        self.assertEqual(len(name), 62)

    def test_falls_back_to_sanitized_hostname_without_mac(self):
        name = default_hostname("Office-Mac", [_iface("lo", "00:00:00:00:00:00", loopback=True, virtual=True)])
        self.assertEqual(name, "office-mac")

    def test_falls_back_to_mxonepeer_when_nothing_is_usable(self):
        self.assertEqual(default_hostname("***", []), "mxonepeer")


class SelectMacTests(unittest.TestCase):
    def test_skips_loopback_and_prefers_physical_over_docker(self):
        mac = select_mac(
            [
                _iface("lo", "00:00:00:00:00:00", loopback=True, virtual=True),
                _iface("docker0", "26:0b:8c:6f:84:05", virtual=True),
                _iface("enp0s3", "82:2b:38:3d:bc:d0", physical=True),
            ]
        )
        self.assertEqual(mac, "822b383dbcd0")

    def test_ignores_all_zero_and_broadcast_macs(self):
        mac = select_mac(
            [
                _iface("eth0", "00:00:00:00:00:00", physical=True),
                _iface("eth1", "ff:ff:ff:ff:ff:ff", physical=True),
                _iface("eth2", "11:22:33:44:55:66", physical=True),
            ]
        )
        self.assertEqual(mac, "112233445566")

    def test_is_stable_when_interface_order_changes(self):
        first = [
            _iface("enp1s0", "22:22:22:22:22:22", physical=True),
            _iface("enp0s3", "11:11:11:11:11:11", physical=True),
        ]
        second = list(reversed(first))
        self.assertEqual(select_mac(first), "111111111111")
        self.assertEqual(select_mac(second), "111111111111")

    def test_prefers_macos_ethernet_over_bridge_and_utun(self):
        mac = select_mac(
            [
                _iface("lo0", "00:00:00:00:00:00", loopback=True, virtual=True),
                _iface("utun0", "00:00:00:00:00:00", virtual=True),
                _iface("bridge0", "aa:aa:aa:aa:aa:aa", virtual=True),
                _iface("awdl0", "bb:bb:bb:bb:bb:bb", virtual=True),
                _iface("en0", "cc:cc:cc:cc:cc:cc", physical=True),
            ]
        )
        self.assertEqual(mac, "cccccccccccc")

    def test_uses_a_virtual_mac_only_when_no_physical_nic_exists(self):
        mac = select_mac(
            [
                _iface("lo", "00:00:00:00:00:00", loopback=True, virtual=True),
                _iface("docker0", "26:0b:8c:6f:84:05", virtual=True),
            ]
        )
        self.assertEqual(mac, "260b8c6f8405")


class LinuxSysfsTests(unittest.TestCase):
    def test_reads_sysfs_and_marks_virtual_nics(self):
        with tempfile.TemporaryDirectory() as directory:
            class_net = os.path.join(directory, "class_net")
            os.makedirs(class_net)
            _write_sysfs(directory, class_net, "lo", "00:00:00:00:00:00", "772", virtual=True)
            _write_sysfs(directory, class_net, "docker0", "26:0b:8c:6f:84:05", "1", virtual=True, bridge=True)
            _write_sysfs(directory, class_net, "enp0s3", "82:2b:38:3d:bc:d0", "1", virtual=False)
            found = {item["name"]: item for item in list_linux_interfaces(class_net)}
        self.assertTrue(found["lo"]["loopback"])
        self.assertTrue(found["docker0"]["virtual"])
        self.assertTrue(found["enp0s3"]["physical"])
        self.assertFalse(found["enp0s3"]["virtual"])
        self.assertEqual(select_mac(found.values()), "822b383dbcd0")


class IfconfigTests(unittest.TestCase):
    def test_parses_macos_ifconfig_output(self):
        text = """lo0: flags=8049<UP,LOOPBACK,RUNNING,MULTICAST> mtu 16384
	inet 127.0.0.1 netmask 0xff000000
en0: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	ether 82:2B:38:3D:BC:D0
	inet 192.168.1.10 netmask 0xffffff00
awdl0: flags=8863<UP,BROADCAST,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	ether aa:aa:aa:aa:aa:aa
bridge0: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	ether bb:bb:bb:bb:bb:bb
utun0: flags=8051<UP,POINTOPOINT,RUNNING,MULTICAST> mtu 1380
"""
        found = {item["name"]: item for item in parse_ifconfig(text)}
        self.assertTrue(found["lo0"]["loopback"])
        self.assertTrue(found["en0"]["physical"])
        self.assertFalse(found["en0"]["virtual"])
        self.assertTrue(found["bridge0"]["virtual"])
        self.assertTrue(found["awdl0"]["virtual"])
        self.assertTrue(found["utun0"]["virtual"])
        self.assertEqual(select_mac(found.values()), "822b383dbcd0")
        self.assertEqual(
            default_hostname("Office-Mac.local", found.values()),
            "office-mac-local-822b383dbcd0",
        )


class EnsureTests(unittest.TestCase):
    def test_writes_generated_default_when_unset(self):
        with tempfile.TemporaryDirectory() as directory:
            name = default_hostname("Desk-1", [_iface("eth0", "aa:bb:cc:dd:ee:ff", physical=True)])
            saved = _ensure_with(directory, "", "Desk-1", [_iface("eth0", "aa:bb:cc:dd:ee:ff", physical=True)])
            self.assertEqual(saved, "desk-1-aabbccddeeff")
            self.assertEqual(read_hostname(directory), name)
            mode = stat.S_IMODE(os.stat(os.path.join(directory, "hostname")).st_mode)
            self.assertEqual(mode, 0o600)

    def test_keeps_a_name_the_user_already_saved(self):
        with tempfile.TemporaryDirectory() as directory:
            write_hostname(directory, "my-custom-node")
            saved = ensure_hostname(directory, "other-name")
            self.assertEqual(saved, "my-custom-node")
            self.assertEqual(read_hostname(directory), "my-custom-node")

    def test_uses_an_explicit_cli_name_only_on_first_run(self):
        with tempfile.TemporaryDirectory() as directory:
            saved = ensure_hostname(directory, "From-CLI")
            self.assertEqual(saved, "from-cli")
            self.assertEqual(ensure_hostname(directory, "later-name"), "from-cli")


def _ensure_with(directory, explicit, os_hostname, interfaces):
    from mxonepeer import hostname as module

    original = module.default_hostname
    module.default_hostname = lambda: default_hostname(os_hostname, interfaces)
    try:
        return ensure_hostname(directory, explicit)
    finally:
        module.default_hostname = original


def _write_sysfs(root, class_net, name, address, typ, virtual, bridge=False):
    if virtual:
        real = os.path.join(root, "devices", "virtual", "net", name)
    else:
        real = os.path.join(root, "devices", "pci0000:00", "0000:00:03.0", "virtio2", "net", name)
        os.makedirs(os.path.join(real, "device"), exist_ok=True)
    os.makedirs(real, exist_ok=True)
    with open(os.path.join(real, "address"), "w", encoding="utf-8") as handle:
        handle.write(address + "\n")
    with open(os.path.join(real, "type"), "w", encoding="utf-8") as handle:
        handle.write(typ + "\n")
    if bridge:
        os.makedirs(os.path.join(real, "bridge"), exist_ok=True)
    os.symlink(real, os.path.join(class_net, name))


if __name__ == "__main__":
    unittest.main()
