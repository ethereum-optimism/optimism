import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("reth-forward-prefilter.py")
GIT_ENV = {
    "GIT_AUTHOR_NAME": "t",
    "GIT_AUTHOR_EMAIL": "t@example.com",
    "GIT_COMMITTER_NAME": "t",
    "GIT_COMMITTER_EMAIL": "t@example.com",
    "GIT_CONFIG_GLOBAL": os.devnull,
    "GIT_CONFIG_NOSYSTEM": "1",
}


class PrefilterTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.repo = Path(self._tmp.name)
        self.git("init", "-q", "-b", "main")

    def tearDown(self):
        self._tmp.cleanup()

    def git(self, *args):
        env = {**os.environ, **GIT_ENV}
        out = subprocess.run(
            ["git", *args], cwd=self.repo, env=env, check=True, capture_output=True, text=True
        )
        return out.stdout.strip()

    def commit(self, path, content, subject):
        (self.repo / path).write_text(content)
        self.git("add", path)
        self.git("commit", "-q", "-m", subject)
        return self.git("rev-parse", "HEAD")

    def rows(self, pin, upstream):
        out = subprocess.run(
            [sys.executable, str(SCRIPT), pin, upstream],
            cwd=self.repo,
            env={**os.environ, **GIT_ENV},
            check=True,
            capture_output=True,
            text=True,
        ).stdout.splitlines()
        self.assertTrue(out[0].startswith("# upstream tip: "))
        header = out[1].split("\t")
        return {r["subject"]: r for r in (dict(zip(header, l.split("\t"))) for l in out[2:])}

    def test_classifies_fixes(self):
        self.commit("a.txt", "1\n2\n3\n", "init")
        self.git("tag", "v1.0.0")
        one = self.commit("one.txt", "one\n", "fix: one")
        self.commit("feat.txt", "feat\n", "feat: unrelated")
        two = self.commit("two.txt", "two\n", "fix: two")
        three = self.commit("a.txt", "1\n2\nthree\n", "fix: three")
        self.commit("other.txt", "x\n", "fix(net): other")

        self.git("checkout", "-q", "-b", "release", "v1.0.0")
        self.git("cherry-pick", "-x", one)
        self.commit("rel.txt", "rel\n", "fix: release only")
        self.git("tag", "v1.0.1")

        self.git("checkout", "-q", "-b", "pin", "v1.0.0")
        self.git("cherry-pick", "-x", two)
        (self.repo / "a.txt").write_text("1\n2\nthree, ported\n")
        self.git("commit", "-qam", f"fix: three\n\n(cherry picked from commit {three})")

        rows = self.rows("pin", "main")

        self.assertEqual(set(rows), {"fix: one", "fix: three", "fix(net): other", "fix: release only"})
        self.assertEqual((rows["fix: one"]["release"], rows["fix: one"]["source"]), ("v1.0.1", "main"))
        self.assertEqual(rows["fix: one"]["apply"], "clean")
        self.assertEqual(
            (rows["fix: release only"]["release"], rows["fix: release only"]["source"]),
            ("v1.0.1", "release-only"),
        )
        self.assertEqual((rows["fix: three"]["picked"], rows["fix: three"]["apply"]), ("picked", "-"))
        self.assertEqual(rows["fix(net): other"]["release"], "main-only")
        self.assertEqual(rows["fix(net): other"]["files"], "0/1")

    def test_no_candidates(self):
        self.commit("a.txt", "1\n", "init")
        self.assertEqual(self.rows("main", "main"), {})


if __name__ == "__main__":
    unittest.main()
