import os
import re

_LABEL = re.compile(r"^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$")


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
