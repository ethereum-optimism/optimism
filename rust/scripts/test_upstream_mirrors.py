#!/usr/bin/env python3
"""Contract tests for the upstream mirror checker."""

from __future__ import annotations

import importlib.util
import io
import sys
import subprocess
import tempfile
import textwrap
import unittest
from contextlib import redirect_stderr, redirect_stdout
from pathlib import Path
from types import SimpleNamespace
from unittest import mock

MODULE_PATH = Path(__file__).with_name("upstream-mirrors.py")
SPEC = importlib.util.spec_from_file_location("upstream_mirrors", MODULE_PATH)
assert SPEC and SPEC.loader
mirrors = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = mirrors
SPEC.loader.exec_module(mirrors)


class SemVerTests(unittest.TestCase):
    def test_rejects_invalid_semver(self) -> None:
        for value in ("banana", "41.0.0junk", "41.0", "v41.0.0", "1٢.0.0"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                mirrors.parse_semver(value)

    def test_orders_prereleases(self) -> None:
        self.assertLess(mirrors.parse_semver("41.0.0-alpha.1"), mirrors.parse_semver("41.0.0-alpha.2"))
        self.assertLess(mirrors.parse_semver("41.0.0-alpha.2"), mirrors.parse_semver("41.0.0"))

    def test_validates_reth_tokens(self) -> None:
        for value in ("rev:aef8d3e", "v2.4.0", "pre-24284"):
            with self.subTest(value=value):
                mirrors.parse_reth_token(value)
        for value in ("rev:aef8d", "v2.4", "pre-main", "pre-1٢", "garbage"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                mirrors.parse_reth_token(value)


class WorkspaceParsingTests(unittest.TestCase):
    def test_lockfile_preserves_duplicate_versions(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            lock = Path(temp) / "Cargo.lock"
            lock.write_text(textwrap.dedent("""
                [[package]]
                name = "alloy-consensus"
                version = "1.8.3"
                source = "registry+https://github.com/rust-lang/crates.io-index"

                [[package]]
                name = "alloy-consensus"
                version = "2.1.1"
                source = "registry+https://github.com/rust-lang/crates.io-index"
            """))
            self.assertEqual(
                mirrors.locked_versions(lock),
                {"alloy-consensus": {"1.8.3", "2.1.1"}},
            )

    def test_lockfile_ignores_non_cratesio_packages(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            lock = Path(temp) / "Cargo.lock"
            lock.write_text(textwrap.dedent("""
                [[package]]
                name = "example"
                version = "1.0.0"
                source = "registry+https://github.com/rust-lang/crates.io-index"

                [[package]]
                name = "example"
                version = "2.0.0"
                source = "git+https://github.com/example/example#deadbeef"
            """))
            self.assertEqual(mirrors.locked_versions(lock), {"example": {"1.0.0"}})

    def test_reth_pin_requires_one_consistent_source(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            manifest = Path(temp) / "Cargo.toml"
            manifest.write_text(textwrap.dedent("""
                [workspace.dependencies]
                reth-a = { git = "https://github.com/op-rs/reth", rev = "aef8d3ef92117f9" }
                reth-b = { git = "https://github.com/op-rs/reth", rev = "aef8d3ef92117f9" }
            """))
            self.assertEqual(mirrors.reth_pin(manifest), ("rev:aef8d3e", "op-rs/reth"))

            manifest.write_text(textwrap.dedent("""
                [workspace.dependencies]
                reth-a = { git = "https://github.com/op-rs/reth", rev = "aef8d3ef92117f9" }
                reth-b = { git = "https://github.com/op-rs/reth", tag = "v2.5.0" }
            """))
            with self.assertRaises(mirrors.ConfigError):
                mirrors.reth_pin(manifest)

    def test_reth_manifest_tag_requires_release_semver(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            manifest = Path(temp) / "Cargo.toml"
            for value in ("pre-123", "main"):
                with self.subTest(value=value):
                    manifest.write_text(textwrap.dedent(f"""
                        [workspace.dependencies]
                        reth = {{ git = "https://github.com/op-rs/reth", tag = "{value}" }}
                    """))
                    with self.assertRaises(mirrors.ConfigError):
                        mirrors.reth_pin(manifest)

    def test_reth_pin_compares_complete_revisions(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            manifest = Path(temp) / "Cargo.toml"
            manifest.write_text(textwrap.dedent("""
                [workspace.dependencies]
                reth-a = { git = "https://github.com/op-rs/reth", rev = "aef8d3e1111111" }
                reth-b = { git = "https://github.com/op-rs/reth", rev = "aef8d3e2222222" }
            """))
            with self.assertRaises(mirrors.ConfigError):
                mirrors.reth_pin(manifest)

    def test_reth_pin_rejects_missing_or_ambiguous_repo(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            manifest = Path(temp) / "Cargo.toml"
            manifest.write_text("[workspace.dependencies]\nfoo = \"1\"\n")
            with self.assertRaises(mirrors.ConfigError):
                mirrors.reth_pin(manifest)

            manifest.write_text(textwrap.dedent("""
                [workspace.dependencies]
                reth-a = { git = "https://github.com/op-rs/reth", rev = "aef8d3ef92117f9" }
                reth-b = { git = "https://github.com/paradigmxyz/reth", rev = "aef8d3ef92117f9" }
            """))
            with self.assertRaises(mirrors.ConfigError):
                mirrors.reth_pin(manifest)

    def test_find_tags_propagates_unreadable_file(self) -> None:
        with tempfile.TemporaryDirectory() as temp, mock.patch.object(
            mirrors.subprocess,
            "run",
            return_value=SimpleNamespace(stdout="missing.rs\0"),
        ):
            with self.assertRaises(OSError):
                mirrors.find_tags(Path(temp))

    def test_find_tags_parses_single_line_and_wrapped_tags(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            subprocess.run(["git", "init", "-q"], cwd=root, check=True)
            (root / "single.rs").write_text(
                "/// UPSTREAM-MIRROR(copy): revm-handler@41.0.0 `revm_handler::Handler::refund`\n"
                "fn single() {}\n"
            )
            (root / "wrapped.rs").write_text(
                "// file header\n"
                "/// UPSTREAM-MIRROR(override): revm-handler@41.0.0\n"
                "/// `revm_handler::Handler::catch_error`\n"
                "fn wrapped() {}\n"
            )
            subprocess.run(["git", "add", "single.rs", "wrapped.rs"], cwd=root, check=True)
            (root / "trailing.rs").write_text(
                "/// UPSTREAM-MIRROR(copy): revm-handler@41.0.0 "
                "`revm_handler::Handler::refund`junk\n"
            )
            (root / "unbalanced.rs").write_text(
                "/// UPSTREAM-MIRROR(copy): revm-handler@41.0.0 "
                "`revm_handler::Handler::refund\n"
            )
            subprocess.run(["git", "add", "trailing.rs", "unbalanced.rs"], cwd=root, check=True)

            found = mirrors.find_tags(root)
            self.assertEqual(
                [(entry.file, entry.line, entry.kind, entry.symbol, entry.status) for entry in found],
                [
                    ("single.rs", 1, "copy", "revm_handler::Handler::refund", None),
                    ("trailing.rs", 1, "?", "?", "malformed"),
                    ("unbalanced.rs", 1, "?", "?", "malformed"),
                    ("wrapped.rs", 2, "override", "revm_handler::Handler::catch_error", None),
                ],
            )

class ClassificationTests(unittest.TestCase):
    RETH_INFO = ("rev:aef8d3e", "op-rs/reth")

    @staticmethod
    def mirror(version: str, *, crate: str = "revm-handler", kind: str = "override"):
        return mirrors.Mirror(kind, crate, version, "upstream::symbol", "src/file.rs", 10)

    def classify(self, mirror, locked=None):
        mirrors.classify(
            [mirror],
            locked=locked or {"revm-handler": {"41.0.0"}},
            reth_info=self.RETH_INFO,
        )
        return mirror

    def test_invalid_cratesio_version_is_malformed(self) -> None:
        self.assertEqual(self.classify(self.mirror("banana")).status, "malformed")
        self.assertEqual(self.classify(self.mirror("41.0.0junk")).status, "malformed")

    def test_duplicate_locked_versions_are_ambiguous(self) -> None:
        mirror = self.mirror("1.8.3", crate="alloy-consensus")
        self.assertEqual(
            self.classify(mirror, {"alloy-consensus": {"1.8.3", "2.1.1"}}).status,
            "ambiguous-version",
        )

    def test_semver_statuses(self) -> None:
        for version, expected in (
            ("40.0.0", "stale"),
            ("41.0.0", "current"),
            ("42.0.0", "ahead"),
        ):
            with self.subTest(version=version):
                self.assertEqual(self.classify(self.mirror(version)).status, expected)

    def test_reth_statuses_and_token_validation(self) -> None:
        current = self.mirror("rev:aef8d3e", crate="reth")
        stale = self.mirror("v2.3.0", crate="reth")
        malformed = self.mirror("garbage", crate="reth")
        mirrors.classify(
            [current, stale, malformed],
            locked={},
            reth_info=self.RETH_INFO,
        )
        self.assertEqual([current.status, stale.status, malformed.status], ["current", "stale", "malformed"])

    def test_reth_release_build_metadata_is_not_ahead(self) -> None:
        mirror = self.mirror("v2.4.0+op.1", crate="reth")
        mirrors.classify(
            [mirror],
            locked={},
            reth_info=("v2.4.0+op.2", "op-rs/reth"),
        )
        self.assertEqual(mirror.status, "stale")

    def test_port_statuses_for_cratesio_and_reth(self) -> None:
        crates_current = self.mirror("41.0.0", kind="port")
        crates_frozen = self.mirror("40.0.0", kind="port")
        reth_current = self.mirror("rev:aef8d3e", crate="reth", kind="port")
        reth_frozen = self.mirror("v1.11.3", crate="reth", kind="port")
        entries = [crates_current, crates_frozen, reth_current, reth_frozen]
        mirrors.classify(
            entries,
            locked={"revm-handler": {"41.0.0"}},
            reth_info=self.RETH_INFO,
        )
        self.assertEqual(
            [entry.status for entry in entries],
            ["current", "frozen", "current", "frozen"],
        )
        self.assertEqual(mirrors.select_mirrors(entries, stale_only=True), [])

    def test_stale_only_excludes_frozen_and_errors(self) -> None:
        entries = []
        for status in ("current", "stale", "frozen", "malformed", "ahead", "unknown-crate"):
            mirror = self.mirror("41.0.0")
            mirror.status = status
            entries.append(mirror)
        self.assertEqual(
            [mirror.status for mirror in mirrors.select_mirrors(entries, stale_only=True)],
            ["stale"],
        )

    def test_check_output_names_bad_tag(self) -> None:
        bad = self.mirror("banana")
        stdout = io.StringIO()
        stderr = io.StringIO()
        with (
            mock.patch.object(mirrors, "find_tags", return_value=[bad]),
            mock.patch.object(mirrors, "locked_versions", return_value={"revm-handler": {"41.0.0"}}),
            mock.patch.object(mirrors, "reth_pin", return_value=self.RETH_INFO),
            redirect_stdout(stdout),
            redirect_stderr(stderr),
        ):
            self.assertEqual(mirrors.main(["--check"]), 1)
        self.assertIn("src/file.rs:10", stdout.getvalue())
        self.assertIn("tag(s) need fixing", stderr.getvalue())

    def test_check_status_policy(self) -> None:
        expectations = {
            "malformed": 1,
            "ahead": 1,
            "unknown-crate": 1,
            "ambiguous-version": 1,
            "stale": 0,
            "frozen": 0,
            "current": 0,
        }
        for status, expected in expectations.items():
            with self.subTest(status=status):
                entry = self.mirror("41.0.0")
                entry.status = mirrors.Status(status)
                with (
                    mock.patch.object(mirrors, "find_tags", return_value=[entry]),
                    mock.patch.object(mirrors, "classify", return_value=[entry]),
                    redirect_stdout(io.StringIO()),
                    redirect_stderr(io.StringIO()),
                ):
                    self.assertEqual(mirrors.main(["--check"]), expected)

    def test_stale_check_includes_bad_diagnostics(self) -> None:
        bad = self.mirror("41.0.0")
        bad.status = mirrors.Status.AHEAD
        with (
            mock.patch.object(mirrors, "find_tags", return_value=[bad]),
            mock.patch.object(mirrors, "classify", return_value=[bad]),
            redirect_stdout(stdout := io.StringIO()),
            redirect_stderr(io.StringIO()),
        ):
            self.assertEqual(mirrors.main(["--stale-only", "--check"]), 1)
        self.assertIn("src/file.rs:10", stdout.getvalue())

    def test_check_fails_closed_on_unknown_status(self) -> None:
        unknown = self.mirror("41.0.0")
        unknown.status = "future-status"
        with (
            mock.patch.object(mirrors, "find_tags", return_value=[unknown]),
            mock.patch.object(mirrors, "classify", return_value=[unknown]),
            redirect_stdout(io.StringIO()),
            redirect_stderr(io.StringIO()),
        ):
            self.assertEqual(mirrors.main(["--check"]), 1)


if __name__ == "__main__":
    unittest.main()
