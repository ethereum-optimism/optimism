#!/usr/bin/env python3

import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).with_name("precommit-targets.py")


class PrecommitTargetsTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_dir = tempfile.TemporaryDirectory()
        self.repo = Path(self.temp_dir.name)

        self._write("op-deployer/pkg/example.go", "package pkg\n")
        self._write("op-deployer/justfile", "test:\n    true\n")
        self._write("op-acceptance-tests/pkg/example.go", "package pkg\n")
        self._write("op-acceptance-tests/justfile", "test:\n    true\n")
        self._write("op-node/pkg/example.go", "package pkg\n")
        self._write("op-node/justfile", "test:\n    true\n")
        self._write("op-e2e/pkg/example.go", "package pkg\n")
        self._write("root.go", "package root\n")
        self._write("cannon/example.txt", "cannon\n")
        self._write("rust/example/Cargo.toml", '[package]\nname = "example-crate"\nversion = "0.1.0"\n')
        self._write("rust/example/src/lib.rs", "pub fn example() {}\n")
        self._write("rust/kona/example.txt", "kona\n")
        self._write("packages/contracts-bedrock/src/Example.sol", "contract Example {}\n")
        self._write(".circleci/continue/example.yml", "version: 2.1\n")
        self._write("docs/example.md", "example\n")

        self._git("init", "-q")
        self._git("config", "user.email", "test@example.com")
        self._git("config", "user.name", "Test")
        self._git("add", ".")
        self._git("commit", "-qm", "initial")

    def tearDown(self) -> None:
        self.temp_dir.cleanup()

    def test_excluded_go_components_only_select_lint(self) -> None:
        for component in ("op-deployer", "op-acceptance-tests"):
            with self.subTest(component=component):
                output = self._select(f"{component}/pkg/example.go")
                self.assertIn("lint-go", output)
                self.assertNotIn(f"{component}/justfile", output)
                self.assertNotIn("go-test-packages", output)

    def test_go_component_with_test_recipe_selects_component_test(self) -> None:
        output = self._select("op-node/pkg/example.go")

        self.assertIn("lint-go", output)
        self.assertIn("op-node/justfile", output)
        self.assertNotIn("go-test-packages", output)

    def test_go_files_without_component_recipe_select_package_tests(self) -> None:
        output = self._select("op-e2e/pkg/example.go", "root.go")

        self.assertIn("lint-go", output)
        self.assertIn("go-test-packages", output)
        self.assertIn("./op-e2e/pkg", output)
        self.assertRegex(output, r"(?:^| )\.(?: |$)")

    def test_rust_file_selects_checks_and_package_test(self) -> None:
        output = self._select("rust/example/src/lib.rs")

        self.assertIn("fmt-fix", output)
        self.assertIn(" lint", output)
        self.assertIn("test-unit", output)
        self.assertIn("example-crate", output)
        self.assertNotIn("check-no-std", output)

    def test_invalid_cargo_manifest_still_selects_general_rust_checks(self) -> None:
        self._git("checkout", "-q", "--", ".")
        self._write(
            "rust/example/Cargo.toml",
            '[package]\nname = "example-crate"\ninvalid = [\n',
        )

        output = self._run("HEAD").stdout

        self.assertIn("fmt-fix", output)
        self.assertIn(" lint", output)
        self.assertNotIn("test-unit", output)

    def test_kona_file_adds_no_std_check(self) -> None:
        output = self._select("rust/kona/example.txt")

        self.assertIn("fmt-fix", output)
        self.assertIn(" lint", output)
        self.assertIn("check-no-std", output)

    def test_special_components_select_their_checks(self) -> None:
        output = self._select(
            "cannon/example.txt",
            "packages/contracts-bedrock/src/Example.sol",
            ".circleci/continue/example.yml",
        )

        self.assertIn("cannon/justfile", output)
        self.assertIn("test-dev", output)
        self.assertIn(".circleci/scripts/merge-configs.sh", output)
        self.assertIn(".circleci/scripts/test-decision-tree.sh", output)

    def test_duplicate_files_do_not_duplicate_commands_or_packages(self) -> None:
        self._write("op-e2e/pkg/another.go", "package pkg\n")
        self._git("add", "op-e2e/pkg/another.go")
        self._git("commit", "-qm", "add another file")

        output = self._select("op-e2e/pkg/example.go", "op-e2e/pkg/another.go")

        self.assertEqual(1, output.count("lint-go"))
        self.assertEqual(1, output.count("./op-e2e/pkg"))

    def test_irrelevant_files_produce_no_commands(self) -> None:
        self.assertEqual("", self._select("docs/example.md"))

    def test_cached_diff_arguments_are_forwarded_to_git(self) -> None:
        self._append("op-node/pkg/example.go")
        self._git("add", "op-node/pkg/example.go")

        result = self._run("--cached")

        self.assertIn("op-node/justfile", result.stdout)

    def test_run_executes_selected_commands_in_order(self) -> None:
        fake_bin = self.repo / "fake-bin"
        fake_bin.mkdir()
        command_log = self.repo / "commands.log"
        mise = fake_bin / "mise"
        mise.write_text('#!/bin/sh\nprintf "%s\\n" "$*" >> "$COMMAND_LOG"\n')
        mise.chmod(0o755)
        self._append("op-node/pkg/example.go")
        env = os.environ | {
            "COMMAND_LOG": str(command_log),
            "PATH": f"{fake_bin}{os.pathsep}{os.environ['PATH']}",
        }

        self._run("--run", "HEAD", env=env)

        commands = command_log.read_text().splitlines()
        self.assertEqual(2, len(commands))
        self.assertIn("lint-go", commands[0])
        self.assertIn("op-node/justfile", commands[1])

    def test_help_documents_python_entrypoint(self) -> None:
        result = self._run("--help")

        self.assertIn("ops/scripts/precommit-targets.py", result.stdout)
        self.assertIn("--run", result.stdout)

    def _select(self, *files: str) -> str:
        self._git("checkout", "-q", "--", ".")
        for file in files:
            self._append(file)
        return self._run("HEAD").stdout.strip()

    def _run(
        self,
        *args: str,
        env: dict[str, str] | None = None,
    ) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(SCRIPT), *args],
            cwd=self.repo,
            check=True,
            text=True,
            capture_output=True,
            env=env,
        )

    def _append(self, relative_path: str) -> None:
        with (self.repo / relative_path).open("a") as file:
            file.write("\n")

    def _write(self, relative_path: str, content: str) -> None:
        path = self.repo / relative_path
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)

    def _git(self, *args: str) -> None:
        subprocess.run(["git", *args], cwd=self.repo, check=True)


if __name__ == "__main__":
    unittest.main()
