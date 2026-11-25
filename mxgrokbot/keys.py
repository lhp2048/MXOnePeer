import os


def normalize_auth_key(value):
    text = (value or "").strip()
    if not text:
        return ""
    if not text.startswith("tskey-auth-"):
        raise ValueError("auth key must start with tskey-auth-")
    return text


def auth_path(state_dir):
    return os.path.join(state_dir, "authkey")


def read_auth_key(state_dir):
    try:
        with open(auth_path(state_dir), "r", encoding="utf-8") as handle:
            return normalize_auth_key(handle.read())
    except FileNotFoundError:
        return ""
    except ValueError:
        return ""


def write_auth_key(state_dir, value):
    key = normalize_auth_key(value)
    if not key:
        raise ValueError("auth key must start with tskey-auth-")
    os.makedirs(state_dir, mode=0o750, exist_ok=True)
    os.chmod(state_dir, 0o750)
    path = auth_path(state_dir)
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(descriptor, "w", encoding="utf-8") as handle:
        handle.write(key + "\n")
    os.chmod(path, 0o600)


def mask_auth_key(value):
    text = (value or "").strip()
    if len(text) < 8:
        return ""
    return "…" + text[-4:]
