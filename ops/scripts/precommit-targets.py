#!/usr/bin/env python3

from __future__ import annotations

import os
import re
import shlex
import subprocess
import sys
import tomllib
from dataclasses import dataclass, field
from pathlib import Path, PurePosixPath
from typing import Sequence


USAGE = """Usage: ops/scripts/precommit-targets.py [--run] [git diff args...]

Print the commands for the precommit checks relevant to the changed files.
With no git diff args, changes on this branch are detected with origin/HEAD...HEAD.

Examples:
  ops/scripts/precommit-targets.py
  ops/scripts/precommit-targets.py --run
  ops/scripts/precommit-targets.py origin/develop...HEAD
  ops/scripts/precommit-targets.py --cached
  ops/scripts/precommit-targets.py HEAD -- op-node/

Options:
  --run       Run the selected commands instead of printing them.
  -h, --help  Show this help.
"""

GO_TEST_EXCLUDED_COMPONENTS = {"op-acceptance-tests", "op-deployer"}
RUST_NO_STD_PREFIXES = (
    "rust/kona/",
    "rust/op-alloy/",
    "rust/alloy-op-evm/",
    "rust/alloy-op-hardforks/",
    "rust/op-revm/",
)


@dataclass(frozen=True)
class Command:
    key: str
    argv: tuple[str, ...]

    def render(self) -> str:
        return shlex.join(self.argv)


@dataclass
class Selection:
    go_lint: bool = False
    cannon_tests: bool = False
    rust_checks: bool = False
    rust_no_std: bool = False
    contracts_checks: bool = False
    circleci_checks: bool = False
    go_component_test_dirs: list[str] = field(default_factory=list)
    go_package_tests: list[str] = field(default_factory=list)
    rust_package_tests: list[str] = field(default_factory=list)


def append_unique(values: list[str], value: str) -> None:
    if value not in values:
        values.append(value)


def go_package_selector(file: str) -> str:
    directory = PurePosixPath(file).parent.as_posix()
    return "." if directory == "." else f"./{directory}"


def rust_package_for_file(repo_root: Path, file: str) -> str | None:
    rust_root = repo_root / "rust"
    directory = repo_root / PurePosixPath(file).parent

    while directory != rust_root and directory.is_relative_to(rust_root):
        manifest = directory / "Cargo.toml"
        if manifest.is_file():
            try:
                package = tomllib.loads(manifest.read_text()).get("package", {})
            except tomllib.TOMLDecodeError:
                return None
            name = package.get("name")
            return name if isinstance(name, str) else None
        directory = directory.parent
    return None


def component_has_test_recipe(repo_root: Path, component: str) -> bool:
    justfile = repo_root / component / "justfile"
    if not justfile.is_file():
        return False
    return re.search(r"^test(?:[\s:*]|$)", justfile.read_text(), re.MULTILINE) is not None


def select(repo_root: Path, changed_files: Sequence[str]) -> Selection:
    selection = Selection()

    for file in changed_files:
        top_dir = file.partition("/")[0]

        if file in {"go.mod", "go.sum"} or file == ".golangci.yaml" or file.startswith("linter/") or file == "justfiles/go.just":
            selection.go_lint = True

        if file.endswith(".go"):
            selection.go_lint = True
            if file.startswith("cannon/"):
                selection.cannon_tests = True
            elif file.startswith("rust/"):
                selection.rust_checks = True
                package = rust_package_for_file(repo_root, file)
                if package is not None:
                    append_unique(selection.rust_package_tests, package)
                append_unique(selection.go_package_tests, go_package_selector(file))
            elif file.startswith("op-e2e/"):
                append_unique(selection.go_package_tests, go_package_selector(file))
            elif top_dir not in GO_TEST_EXCLUDED_COMPONENTS:
                if component_has_test_recipe(repo_root, top_dir):
                    append_unique(selection.go_component_test_dirs, top_dir)
                else:
                    append_unique(selection.go_package_tests, go_package_selector(file))

        if file.startswith("cannon/"):
            selection.cannon_tests = True

        if file.startswith("rust/"):
            selection.rust_checks = True
            package = rust_package_for_file(repo_root, file)
            if package is not None:
                append_unique(selection.rust_package_tests, package)
            if file in {"rust/Cargo.toml", "rust/Cargo.lock"} or file.startswith(RUST_NO_STD_PREFIXES):
                selection.rust_no_std = True

        if (
            file.startswith("packages/contracts-bedrock/")
            or file.startswith("op-core/forks/")
            or file.startswith("op-core/nuts/")
            or file.startswith(".semgrep/")
        ):
            selection.contracts_checks = True

        if file.startswith(".circleci/"):
            selection.circleci_checks = True

    return selection


def just_command(key: str, directory: Path, *args: str, install_tools: bool = False) -> Command:
    mise_args = ("mise", "x", "--") if install_tools else ("mise", "exec", "--")
    return Command(
        key,
        (*mise_args, "just", "-f", str(directory / "justfile"), "-d", str(directory), *args),
    )


def build_commands(repo_root: Path, selection: Selection) -> list[Command]:
    commands: dict[str, Command] = {}

    def add(command: Command) -> None:
        commands.setdefault(command.key, command)

    if selection.go_lint:
        add(just_command("go-lint", repo_root, "lint-go"))
    if selection.cannon_tests:
        add(just_command("cannon-tests", repo_root / "cannon", "test"))
    for directory in selection.go_component_test_dirs:
        add(just_command(f"go-component-test:{directory}", repo_root / directory, "test"))
    if selection.go_package_tests:
        add(
            Command(
                "go-package-tests",
                (
                    "mise",
                    "exec",
                    "--",
                    "just",
                    "-f",
                    str(repo_root / "justfile"),
                    "-d",
                    str(repo_root),
                    "go-test-packages",
                    *selection.go_package_tests,
                ),
            )
        )
    if selection.rust_checks:
        add(just_command("rust-fmt", repo_root / "rust", "fmt-fix"))
        add(just_command("rust-lint", repo_root / "rust", "lint"))
    if selection.rust_package_tests:
        package_args = tuple(arg for package in selection.rust_package_tests for arg in ("-p", package))
        add(
            just_command(
                "rust-test-unit-packages",
                repo_root / "rust",
                "test-unit",
                "-E",
                "!test(test_online)",
                *package_args,
            )
        )
    if selection.rust_no_std:
        add(just_command("rust-check-no-std", repo_root / "rust", "check-no-std"))
    if selection.contracts_checks:
        contracts = repo_root / "packages/contracts-bedrock"
        add(just_command("contracts-lint", contracts, "lint", install_tools=True))
        add(just_command("contracts-test-dev", contracts, "test-dev", install_tools=True))
    if selection.circleci_checks:
        add(
            Command(
                "circleci-merge",
                ("mise", "exec", "--", "bash", str(repo_root / ".circleci/scripts/merge-configs.sh")),
            )
        )
        add(
            Command(
                "circleci-decision-tree",
                ("mise", "exec", "--", "bash", str(repo_root / ".circleci/scripts/test-decision-tree.sh")),
            )
        )

    return list(commands.values())


def parse_args(args: Sequence[str]) -> tuple[bool, bool, list[str]]:
    run = False
    show_help = False
    diff_args: list[str] = []
    index = 0

    while index < len(args):
        arg = args[index]
        if arg == "--run":
            run = True
        elif arg in {"-h", "--help"}:
            show_help = True
        elif arg == "--":
            diff_args.extend(args[index:])
            break
        else:
            diff_args.append(arg)
        index += 1

    return run, show_help, diff_args


def capture(*args: str, check: bool = True) -> subprocess.CompletedProcess[bytes]:
    return subprocess.run(args, stdout=subprocess.PIPE, check=check)


def repo_root() -> Path:
    result = capture("git", "rev-parse", "--show-toplevel")
    return Path(os.fsdecode(result.stdout).strip())


def default_diff_args(root: Path) -> list[str]:
    base = capture(
        "git",
        "-C",
        str(root),
        "symbolic-ref",
        "--quiet",
        "--short",
        "refs/remotes/origin/HEAD",
        check=False,
    )
    base_ref = os.fsdecode(base.stdout).strip() or "origin/develop"
    merge_base = capture("git", "-C", str(root), "merge-base", base_ref, "HEAD")
    return [f"{os.fsdecode(merge_base.stdout).strip()}...HEAD"]


def changed_files(root: Path, diff_args: Sequence[str]) -> list[str]:
    result = capture("git", "-C", str(root), "diff", "--name-only", "-z", *diff_args)
    return [os.fsdecode(file) for file in result.stdout.split(b"\0") if file]


def main(args: Sequence[str]) -> int:
    run, show_help, diff_args = parse_args(args)
    if show_help:
        print(USAGE, end="")
        return 0

    try:
        root = repo_root()
        if not diff_args:
            diff_args = default_diff_args(root)
        commands = build_commands(root, select(root, changed_files(root, diff_args)))
    except subprocess.CalledProcessError as error:
        return error.returncode

    if not run:
        for command in commands:
            print(command.render())
        return 0

    for command in commands:
        result = subprocess.run(command.argv)
        if result.returncode != 0:
            return result.returncode
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
