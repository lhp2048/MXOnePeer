import os
import re
import subprocess
import sys

_LABEL = re.compile(r"^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$")
_MAC = re.compile(r"^[0-9a-f]{12}$")
_IFACE_START = re.compile(r"^([A-Za-z0-9_.-]+):")
MAX_HOSTNAME_LEN = 63
_MAC_SUFFIX_LEN = 13

_VIRTUAL_PREFIXES = (
    "lo",
    "docker",
    "veth",
    "br-",
    "virbr",
    "cni",
    "flannel",
    "cbr",
    "lxc",
    "podman",
    "nerdctl",
    "tun",
    "tap",
    "utun",
    "wg",
    "tailscale",
    "zt",
    "dummy",
    "sit",
    "gre",
    "gretap",
    "erspan",
    "ip6tnl",
    "vxlan",
    "geneve",
    "vcan",
    "nlmon",
    "awdl",
    "llw",
    "anpi",
    "ap",
    "gif",
    "stf",
    "vmnet",
    "vboxnet",
    "vmenet",
    "bridge",
)

_PREFERRED_PREFIXES = (
    "eno",
    "ens",
    "enp",
    "enx",
    "eth",
    "em",
    "igb",
    "ix",
    "igc",
    "wlan",
    "wl",
    "en",
)


def normalize_hostname(value):
    text = (value or "").strip().lower()
    if not _LABEL.match(text):
        raise ValueError("invalid hostname")
    return text


def hostname_path(state_dir):
    return os.path.join(state_dir, "hostname")


def read_hostname(state_dir):
    try:
        with open(hostname_path(state_dir), "r", encoding="utf-8") as handle:
            return normalize_hostname(handle.read())
    except (OSError, ValueError):
        return ""


def write_hostname(state_dir, value):
    name = normalize_hostname(value)
    os.makedirs(state_dir, mode=0o750, exist_ok=True)
    os.chmod(state_dir, 0o750)
    path = hostname_path(state_dir)
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(descriptor, "w", encoding="utf-8") as handle:
        handle.write(name + "\n")
    os.chmod(path, 0o600)
    return name


def ensure_hostname(state_dir, explicit=""):
    saved = read_hostname(state_dir)
    if saved:
        return saved
    candidate = (explicit or "").strip()
    if candidate:
        try:
            return write_hostname(state_dir, candidate)
        except ValueError:
            pass
    return write_hostname(state_dir, default_hostname())


def default_hostname(os_hostname=None, interfaces=None):
    if interfaces is None:
        interfaces = list_interfaces()
    if os_hostname is None:
        os_hostname = current_os_hostname()
    mac = select_mac(interfaces)
    host = _sanitize_token(os_hostname, MAX_HOSTNAME_LEN - _MAC_SUFFIX_LEN if mac else MAX_HOSTNAME_LEN)
    if mac and host:
        name = "%s-%s" % (host, mac)
    elif mac:
        name = mac
    elif host:
        name = host
    else:
        name = "mxonepeer"
    return normalize_hostname(name)


def current_os_hostname():
    try:
        return os.uname().nodename
    except OSError:
        return ""


def list_interfaces(sys_class_net="/sys/class/net", platform=None, ifconfig_text=None):
    system = sys.platform if platform is None else platform
    if ifconfig_text is not None:
        return parse_ifconfig(ifconfig_text)
    if system == "darwin":
        return parse_ifconfig(_ifconfig_output())
    return list_linux_interfaces(sys_class_net)


def list_linux_interfaces(sys_class_net="/sys/class/net"):
    try:
        names = os.listdir(sys_class_net)
    except OSError:
        return []
    found = []
    for name in names:
        path = os.path.join(sys_class_net, name)
        mac = _read_text(os.path.join(path, "address"))
        typ = _read_text(os.path.join(path, "type"))
        real = os.path.realpath(path)
        virtual_path = "/devices/virtual/" in real.replace("\\", "/")
        physical = os.path.isdir(os.path.join(path, "device")) and not virtual_path
        bridge = os.path.isdir(os.path.join(path, "bridge"))
        loopback = typ == "772" or _is_loopback_name(name)
        virtual = (not physical) or virtual_path or bridge or _is_virtual_name(name)
        found.append(
            {
                "name": name,
                "mac": mac,
                "physical": physical,
                "virtual": virtual,
                "loopback": loopback,
            }
        )
    return found


def parse_ifconfig(text):
    found = []
    current = None
    for raw in (text or "").splitlines():
        if not raw:
            continue
        if not raw[0].isspace():
            match = _IFACE_START.match(raw)
            if match:
                if current is not None:
                    found.append(current)
                name = match.group(1)
                loopback = "LOOPBACK" in raw or _is_loopback_name(name)
                current = {
                    "name": name,
                    "mac": "",
                    "physical": not loopback and not _is_virtual_name(name),
                    "virtual": loopback or _is_virtual_name(name),
                    "loopback": loopback,
                }
            continue
        if current is None:
            continue
        stripped = raw.strip().lower()
        if stripped.startswith("ether ") or stripped.startswith("lladdr "):
            current["mac"] = stripped.split(None, 1)[1].split()[0]
            if current["mac"] in ("00:00:00:00:00:00", "ff:ff:ff:ff:ff:ff"):
                current["physical"] = False
    if current is not None:
        found.append(current)
    return found


def select_mac(interfaces):
    candidates = []
    for item in interfaces or []:
        mac = normalize_mac(item.get("mac", ""))
        if not mac or item.get("loopback") or _is_loopback_name(item.get("name", "")):
            continue
        candidates.append((item, mac))
    if not candidates:
        return ""
    physical = [(item, mac) for item, mac in candidates if item.get("physical") and not item.get("virtual")]
    if physical:
        chosen = physical
    else:
        ordinary = [(item, mac) for item, mac in candidates if not item.get("virtual")]
        chosen = ordinary or candidates
    chosen.sort(key=lambda pair: (_preference(pair[0].get("name", "")), pair[0].get("name", ""), pair[1]))
    return chosen[0][1]


def normalize_mac(value):
    text = re.sub(r"[^0-9a-fA-F]", "", value or "").lower()
    if not _MAC.match(text):
        return ""
    if text in ("000000000000", "ffffffffffff"):
        return ""
    return text


def _sanitize_token(value, max_len):
    limit = max(1, min(max_len, MAX_HOSTNAME_LEN))
    text = re.sub(r"[^a-z0-9]+", "-", (value or "").strip().lower()).strip("-")
    if len(text) > limit:
        text = text[:limit].rstrip("-")
    return text


def _preference(name):
    for index, prefix in enumerate(_PREFERRED_PREFIXES):
        if name.startswith(prefix):
            return index
    return len(_PREFERRED_PREFIXES)


def _is_virtual_name(name):
    lowered = (name or "").lower()
    if _is_loopback_name(lowered) or re.match(r"^br\d+$", lowered):
        return True
    return any(lowered == prefix or lowered.startswith(prefix) for prefix in _VIRTUAL_PREFIXES)


def _is_loopback_name(name):
    return re.match(r"^lo\d*$", (name or "").lower()) is not None


def _ifconfig_output():
    try:
        result = subprocess.run(
            ["ifconfig", "-a"],
            check=False,
            capture_output=True,
            text=True,
        )
    except OSError:
        return ""
    return result.stdout or ""


def _read_text(path):
    try:
        with open(path, "r", encoding="utf-8") as handle:
            return handle.read().strip()
    except OSError:
        return ""
