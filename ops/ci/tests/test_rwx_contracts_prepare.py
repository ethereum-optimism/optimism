#!/usr/bin/env python3
"""Exercise the RWX contract tool installation, its download retries and failures."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


SCRIPT = Path(__file__).resolve().parents[1] / 'runtime' / 'rwx-contracts-prepare.sh'

# Commands are stubbed at the process boundary: the real script decides which
# tools and compilers to install, without downloading anything.
STUB = r'''#!/usr/bin/env python3
import json
import os
from pathlib import Path
import sys

command = Path(sys.argv[0]).name
args = sys.argv[1:]
env = os.environ
with Path(env["CALL_LOG"]).open("a") as log:
    log.write(json.dumps({"command": command, "args": args}) + "\n")

if command == "mise" and args == ["bin-paths"]:
    print(env["STUB_BIN"])
    sys.exit(0)
elif command == "mise" and args == ["install", "forge", "cast", "svm-rs"]:
    label = "mise"
elif command == "svm" and len(args) == 2 and args[0] == "which":
    sys.exit(0 if args[1] in env.get("INSTALLED_SOLC", "").split(",") else 1)
elif command == "svm" and len(args) == 2 and args[0] == "install":
    label = "svm-" + args[1]
elif command == "sleep":
    sys.exit(0)
else:
    sys.exit("Unexpected command: " + command + " " + repr(args))

counter = Path(env["COUNTERS"]) / label
count = int(counter.read_text()) + 1 if counter.exists() else 1
counter.write_text(str(count))
if count <= int(env.get("FAIL_" + label.upper().replace("-", "_").replace(".", "_"), "0")):
    sys.exit(17)
print("Prepared " + label)
'''


class RwxContractsPrepareTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.script = self.root / "ops/ci/runtime" / SCRIPT.name
        self.script.parent.mkdir(parents=True)
        shutil.copyfile(SCRIPT, self.script)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        for name in ("mise", "svm", "sleep"):
            path = self.bin / name
            path.write_text(STUB, encoding="utf-8")
            path.chmod(0o755)
        self.log = self.root / "calls.jsonl"
        self.counters = self.root / "counters"
        self.counters.mkdir()

    def run_helper(self, mode="tools", **overrides):
        env = dict(os.environ)
        env.update(
            PATH=str(self.bin) + os.pathsep + os.environ["PATH"],
            CALL_LOG=str(self.log), COUNTERS=str(self.counters), STUB_BIN=str(self.bin),
            RWX_ENV=str(self.root / "rwx-env"),
        )
        env.update(overrides)
        return subprocess.run(["bash", str(self.script), mode], cwd=self.root, env=env, text=True, capture_output=True)

    def calls(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()] if self.log.exists() else []

    def test_tools_install_only_pinned_contract_tools_and_exact_missing_compilers(self):
        result = self.run_helper(FAIL_MISE="1", FAIL_SVM_0_8_25="1", INSTALLED_SOLC="0.8.15")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        calls = self.calls()
        installs = [call["args"] for call in calls if call["command"] == "mise" and call["args"][0] == "install"]
        self.assertEqual(installs, [["install", "forge", "cast", "svm-rs"]] * 2)
        self.assertEqual([call["args"][1] for call in calls if call["command"] == "svm" and call["args"][0] == "which"],
                         ["0.8.15", "0.8.19", "0.8.25", "0.8.28"])
        self.assertEqual([call["args"][1] for call in calls if call["command"] == "svm" and call["args"][0] == "install"],
                         ["0.8.19", "0.8.25", "0.8.25", "0.8.28"])
        self.assertTrue((self.root / "rwx-env/PATH").is_file())

    def test_failed_tool_or_compiler_download_stops_bootstrap(self):
        for overrides, expected_command in (({"FAIL_MISE": "5"}, "mise"), ({"FAIL_SVM_0_8_15": "5"}, "svm")):
            with self.subTest(overrides=overrides):
                for path in self.counters.iterdir():
                    path.unlink()
                self.log.unlink(missing_ok=True)
                result = self.run_helper(**overrides)
                self.assertEqual(result.returncode, 17, result.stdout + result.stderr)
                downloads = [call for call in self.calls() if call["command"] == expected_command and call["args"][0] == "install"]
                self.assertEqual(len(downloads), 5)
                self.assertFalse((self.root / "rwx-env/PATH").exists())

    def test_unknown_mode_installs_nothing(self):
        result = self.run_helper("source")
        self.assertEqual((result.returncode, self.calls()), (1, []))
        self.assertIn("Usage:", result.stderr)


if __name__ == "__main__":
    unittest.main()
