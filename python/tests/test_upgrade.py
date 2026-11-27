import io
import os
import tempfile
import unittest
import zipfile

from mxonepeer.upgrade import EXIT_CODE, apply_tree, check_zip, decide, filename_ok


def _zip(entries):
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w") as archive:
        for name, content in entries:
            archive.writestr(name, content)
    return buf.getvalue()


class UpgradeTests(unittest.TestCase):
    def test_filename(self):
        self.assertTrue(filename_ok("python/linux/amd64", "mxonepeer-python-linux-amd64.zip"))
        self.assertFalse(filename_ok("python/linux/amd64", "mxonepeer-python-linux-arm64.zip"))
        self.assertFalse(filename_ok("python/linux/unknown", "mxonepeer-python-linux-unknown.zip"))
        self.assertFalse(filename_ok("go/linux/amd64", "mxonepeer-linux-amd64"))

    def test_zip_rejects_parent_path(self):
        payload = _zip([("mxonepeer/../../outside.txt", "x"), ("mxonepeer/__init__.py", ""), ("mxonepeer/__main__.py", "")])
        self.assertEqual(check_zip(payload), "包的内容不正确")

    def test_zip_rejects_colon_in_segment(self):
        for name in ("mxonepeer/D:outside.txt", "mxonepeer/C:../outside.txt"):
            payload = _zip([
                (name, "x"),
                ("mxonepeer/__init__.py", ""),
                ("mxonepeer/__main__.py", ""),
            ])
            self.assertEqual(check_zip(payload), "包的内容不正确")

    def test_apply_keeps_previous(self):
        payload = _zip([
            ("mxonepeer/__init__.py", "new"),
            ("mxonepeer/__main__.py", "main"),
        ])
        root = tempfile.mkdtemp()
        program = os.path.join(root, "mxonepeer")
        os.makedirs(program)
        with open(os.path.join(program, "__init__.py"), "w", encoding="utf-8") as handle:
            handle.write("old")
        apply_tree(program, payload)
        with open(os.path.join(program, "__init__.py"), encoding="utf-8") as handle:
            self.assertEqual(handle.read(), "new")
        with open(os.path.join(program + ".previous", "__init__.py"), encoding="utf-8") as handle:
            self.assertEqual(handle.read(), "old")

    def test_decide_rejects_and_accepts(self):
        root = tempfile.mkdtemp()
        program = os.path.join(root, "mxonepeer")
        os.makedirs(program)
        with open(os.path.join(program, "__init__.py"), "w", encoding="utf-8") as handle:
            handle.write("old")
        status, text, should_exit = decide(
            "POST",
            {"x-mx-runtime": "python/darwin/arm64", "x-mx-filename": "mxonepeer-python-linux-amd64.zip"},
            b"zip",
            "python/linux/amd64",
            program,
        )
        self.assertEqual(status, 400)
        self.assertEqual(text, "runtime 不一致")
        self.assertFalse(should_exit)
        self.assertTrue(os.path.exists(os.path.join(program, "__init__.py")))
        payload = _zip([("mxonepeer/__init__.py", "new"), ("mxonepeer/__main__.py", "main")])
        status, text, should_exit = decide(
            "POST",
            {"x-mx-runtime": "python/linux/amd64", "x-mx-filename": "mxonepeer-python-linux-amd64.zip"},
            payload,
            "python/linux/amd64",
            program,
        )
        self.assertEqual((status, text, should_exit), (204, "", True))
        self.assertEqual(EXIT_CODE, 75)


if __name__ == "__main__":
    unittest.main()
