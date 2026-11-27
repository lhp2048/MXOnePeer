import io
import os
import platform
import shutil
import threading
import zipfile

MAX_BYTES = 64 * 1024 * 1024
EXIT_CODE = 75
_BAD = "包的内容不正确"
_replace_lock = threading.Lock()


def runtime_id():
    system = platform.system()
    names = {"Linux": "linux", "Darwin": "darwin", "Windows": "windows"}
    os_name = names.get(system, system.lower())
    machine = platform.machine().lower()
    if machine in ("x86_64", "amd64"):
        arch = "amd64"
    elif machine in ("aarch64", "arm64"):
        arch = "arm64"
    else:
        arch = "unknown"
    return "python/%s/%s" % (os_name, arch)


def filename_ok(runtime, name):
    parts = runtime.split("/")
    if len(parts) != 3 or parts[0] != "python" or parts[2] in ("", "unknown"):
        return False
    if "/" in name or "\\" in name:
        return False
    return name == "mxonepeer-python-%s-%s.zip" % (parts[1], parts[2])


def check_zip(data):
    try:
        archive = zipfile.ZipFile(io.BytesIO(data))
    except zipfile.BadZipFile:
        return _BAD
    with archive:
        total = 0
        names = set()
        for info in archive.infolist():
            name = info.filename
            if "\\" in name or name.startswith("/") or (len(name) >= 2 and name[1] == ":"):
                return _BAD
            if not name.startswith("mxonepeer/"):
                return _BAD
            if any(".." in part or ":" in part for part in name.split("/")):
                return _BAD
            if (info.external_attr >> 16) & 0o170000 == 0o120000:
                return _BAD
            total += info.file_size
            if total > MAX_BYTES:
                return _BAD
            if not name.endswith("/"):
                names.add(name)
        if "mxonepeer/__init__.py" not in names or "mxonepeer/__main__.py" not in names:
            return _BAD
    return ""


def apply_tree(program_dir, data):
    with _replace_lock:
        if check_zip(data):
            raise ValueError(_BAD)
        incoming = program_dir + ".incoming"
        previous = program_dir + ".previous"
        shutil.rmtree(incoming, ignore_errors=True)
        os.makedirs(incoming)
        try:
            _extract(data, incoming)
        except OSError:
            shutil.rmtree(incoming, ignore_errors=True)
            raise OSError("写不进去")
        except zipfile.BadZipFile:
            shutil.rmtree(incoming, ignore_errors=True)
            raise ValueError(_BAD)
        except ValueError:
            shutil.rmtree(incoming, ignore_errors=True)
            raise
        try:
            if os.path.exists(previous):
                shutil.rmtree(previous)
            os.rename(program_dir, previous)
            os.rename(incoming, program_dir)
        except OSError:
            if not os.path.exists(program_dir) and os.path.exists(previous):
                try:
                    os.rename(previous, program_dir)
                except OSError:
                    pass
            shutil.rmtree(incoming, ignore_errors=True)
            raise OSError("写不进去")


def _extract(data, incoming):
    with zipfile.ZipFile(io.BytesIO(data)) as archive:
        for info in archive.infolist():
            if info.filename.endswith("/"):
                continue
            rel = info.filename[len("mxonepeer/"):]
            target = os.path.join(incoming, *rel.split("/"))
            root = os.path.realpath(incoming)
            resolved = os.path.realpath(target)
            if resolved != root and not resolved.startswith(root + os.sep):
                raise ValueError(_BAD)
            parent = os.path.dirname(target)
            if parent:
                os.makedirs(parent, exist_ok=True)
            with archive.open(info) as source, open(target, "wb") as output:
                shutil.copyfileobj(source, output)


def decide(method, headers, body, runtime, program_dir):
    if method != "POST":
        return 405, "method not allowed", False
    if headers.get("x-mx-runtime") != runtime:
        return 400, "runtime 不一致", False
    if not filename_ok(runtime, headers.get("x-mx-filename", "")):
        return 400, "文件名不正确", False
    if check_zip(body):
        return 400, "包的内容不正确", False
    try:
        apply_tree(program_dir, body)
    except ValueError:
        return 400, "包的内容不正确", False
    except OSError:
        return 500, "写不进去", False
    return 204, "", True
