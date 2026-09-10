#!/usr/bin/env python3
"""Report OP code that mirrors upstream reth/revm/alloy logic and has fallen behind the pin.

Reads `UPSTREAM-MIRROR` tags out of the Rust sources and compares each tag's
last-verified version against what the workspace currently resolves to, so a bump
gets a worklist proportional to what could have drifted rather than a checklist.

Tag grammar, in the doc comment of the mirroring item:

    /// UPSTREAM-MIRROR(<kind>): <crate>@<version> <upstream::symbol::path>

  <kind>     override | copy | delegate | set | port   (see docs/ai/reth-upstream-mirrors.md)
  <crate>    crates.io package name, or `reth` for the git-pinned family
  <version>  exact semver for crates.io; `rev:<sha7>`, `v<semver>` or `pre-<pr>` for reth
  <symbol>   full upstream path, used to locate the item in upstream sources

Usage:
  upstream-mirrors.py                 list every mirror with its status
  upstream-mirrors.py --stale-only    only the stale bump worklist
  upstream-mirrors.py --json          machine-readable, for CI comments
  upstream-mirrors.py --check         exit 1 on tag or workspace metadata errors
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
import tomllib
from dataclasses import asdict, dataclass
from enum import StrEnum
from pathlib import Path

RUST_ROOT = Path(__file__).resolve().parent.parent

RUST_PATH = r"[A-Za-z_][A-Za-z0-9_]*(?:::[A-Za-z_][A-Za-z0-9_]*)*"
TAG_RE = re.compile(
    r"^UPSTREAM-MIRROR\((?P<kind>[a-z]+)\):\s*"
    r"(?P<crate>[A-Za-z0-9_-]+)@(?P<version>[A-Za-z0-9_.:+-]+)\s+"
    rf"(?:`(?P<quoted_symbol>{RUST_PATH})`|(?P<bare_symbol>{RUST_PATH}))$"
)
DOC_RE = re.compile(r"^\s*(///|//!)\s?(.*)$")
KINDS = {"override", "copy", "delegate", "set", "port"}

SEMVER_RE = re.compile(
    r"^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)"
    r"(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?"
    r"(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$"
)


class ConfigError(RuntimeError):
    """Invalid or ambiguous workspace dependency metadata."""

class Status(StrEnum):
    MALFORMED = "malformed"
    UNKNOWN_CRATE = "unknown-crate"
    AMBIGUOUS_VERSION = "ambiguous-version"
    AHEAD = "ahead"
    STALE = "stale"
    FROZEN = "frozen"
    CURRENT = "current"


@dataclass(frozen=True)
class SemVer:
    major: int
    minor: int
    patch: int
    prerelease: tuple[tuple[int, int | str], ...] | None = None
    build: str | None = None

    def __lt__(self, other: SemVer) -> bool:
        return self.precedence_key() < other.precedence_key()

    def precedence_key(self) -> tuple[object, ...]:
        return (
            self.major,
            self.minor,
            self.patch,
            self.prerelease is None,
            self.prerelease or (),
        )


def parse_semver(value: str) -> SemVer:
    """Parse one complete SemVer 2.0 token."""
    match = SEMVER_RE.fullmatch(value)
    if match is None:
        raise ValueError(f"invalid semver {value!r}")

    prerelease = None
    if match.group(4) is not None:
        identifiers: list[tuple[int, int | str]] = []
        for identifier in match.group(4).split("."):
            if identifier.isdigit():
                if len(identifier) > 1 and identifier.startswith("0"):
                    raise ValueError(f"invalid semver {value!r}")
                identifiers.append((0, int(identifier)))
            else:
                identifiers.append((1, identifier))
        prerelease = tuple(identifiers)

    return SemVer(
        major=int(match.group(1)),
        minor=int(match.group(2)),
        patch=int(match.group(3)),
        prerelease=prerelease,
        build=match.group(5),
    )


def parse_reth_token(value: str) -> tuple[str, SemVer | str]:
    """Validate and classify a reth-family pin token."""
    if re.fullmatch(r"rev:[0-9a-f]{7}", value):
        return "rev", value[4:]
    if re.fullmatch(r"pre-[1-9][0-9]*", value):
        return "pre", value[4:]
    if value.startswith("v"):
        return "tag", parse_semver(value[1:])
    raise ValueError(f"invalid reth pin token {value!r}")


@dataclass
class Mirror:
    kind: str
    crate: str
    verified: str
    symbol: str
    file: str
    line: int
    pinned: str = ""
    repo: str = ""
    status: Status | None = None
    note: str = ""


def normalized_repo(url: str) -> str:
    match = re.fullmatch(r"https://github\.com/([\w.-]+/reth)(?:\.git)?/?", url)
    if match is None:
        raise ConfigError(f"unsupported reth git source {url!r}")
    return match.group(1)


def find_tags(root: Path = RUST_ROOT) -> list[Mirror]:
    files = subprocess.run(
        ["git", "ls-files", "-z", "*.rs"],
        cwd=root, stdout=subprocess.PIPE, text=True, check=True,
    ).stdout.split("\0")
    out: list[Mirror] = []
    for rel in filter(None, files):
        path = root / rel
        text = path.read_text(encoding="utf8", errors="replace")
        if "UPSTREAM-MIRROR" not in text:
            continue
        lines = text.splitlines()
        for n, line in enumerate(lines, 1):
            if "UPSTREAM-MIRROR" not in line:
                continue
            # rustfmt may wrap a tag across doc-comment lines. Stop as soon as
            # the complete tag matches, before maintenance prose begins.
            parts, i, match = [], n - 1, None
            while i < len(lines) and (doc := DOC_RE.match(lines[i])) and doc.group(2).strip():
                parts.append(doc.group(2).strip())
                if match := TAG_RE.fullmatch(" ".join(parts)):
                    break
                i += 1
            if match:
                out.append(Mirror(
                    kind=match.group("kind"),
                    crate=match.group("crate"),
                    verified=match.group("version"),
                    symbol=match.group("quoted_symbol") or match.group("bare_symbol"),
                    file=rel,
                    line=n,
                ))
            else:
                out.append(Mirror(kind="?", crate="?", verified="?", symbol="?",
                                  file=rel, line=n, status=Status.MALFORMED,
                                  note="does not match the tag grammar"))
    return out


def locked_versions(lock_path: Path | None = None) -> dict[str, set[str]]:
    """Every package version the main workspace resolves to, grouped by package name."""
    path = lock_path or RUST_ROOT / "Cargo.lock"
    with path.open("rb") as lock_file:
        packages = tomllib.load(lock_file).get("package", [])

    versions: dict[str, set[str]] = {}
    crates_io = "registry+https://github.com/rust-lang/crates.io-index"
    for package in packages:
        if package.get("source") == crates_io:
            versions.setdefault(package["name"], set()).add(package["version"])
    return versions


def reth_pin(manifest_path: Path | None = None) -> tuple[str, str]:
    """Return the one consistent (pin token, source repo) used by the reth family."""
    path = manifest_path or RUST_ROOT / "Cargo.toml"
    with path.open("rb") as manifest_file:
        manifest = tomllib.load(manifest_file)

    dependencies = manifest.get("workspace", {}).get("dependencies", {})
    sources: set[tuple[str, str, str]] = set()
    for dependency in dependencies.values():
        if not isinstance(dependency, dict) or "git" not in dependency:
            continue
        git = dependency["git"]
        if not re.search(r"/reth(?:\.git)?/?$", git):
            continue

        refs = [key for key in ("tag", "rev") if key in dependency]
        if len(refs) != 1:
            raise ConfigError(f"reth dependency must declare exactly one tag or rev: {dependency}")
        ref = refs[0]
        value = dependency[ref]
        try:
            if ref == "tag":
                kind, _ = parse_reth_token(value)
                if kind != "tag":
                    raise ValueError("manifest tag must be v<semver>")
            elif re.fullmatch(r"[0-9a-f]{7,40}", value) is None:
                raise ValueError(f"invalid reth revision {value!r}")
        except ValueError as error:
            raise ConfigError(f"invalid reth {ref} {value!r}: {error}") from error
        sources.add((ref, value, normalized_repo(git)))

    if len(sources) != 1:
        raise ConfigError(
            f"expected one consistent reth repository and pin, found {sorted(sources)!r}"
        )
    ref, value, repo = sources.pop()
    token = value if ref == "tag" else f"rev:{value[:7]}"
    return token, repo


def classify(
    mirrors: list[Mirror],
    *,
    locked: dict[str, set[str]] | None = None,
    reth_info: tuple[str, str] | None = None,
) -> list[Mirror]:
    locked = locked if locked is not None else locked_versions()
    reth, reth_repo = reth_info if reth_info is not None else reth_pin()
    for mirror in mirrors:
        if mirror.status == Status.MALFORMED:
            continue
        if mirror.kind not in KINDS:
            mirror.status, mirror.note = Status.MALFORMED, f"unknown kind {mirror.kind!r}"
            continue

        if mirror.crate == "reth":
            mirror.pinned, mirror.repo = reth, reth_repo
            try:
                verified_kind, verified = parse_reth_token(mirror.verified)
                pinned_kind, pinned = parse_reth_token(reth)
            except ValueError as error:
                mirror.status, mirror.note = Status.MALFORMED, str(error)
                continue

            if mirror.kind == "port":
                mirror.status = Status.FROZEN if mirror.verified != reth else Status.CURRENT
            elif mirror.verified == reth:
                mirror.status = Status.CURRENT
            elif verified_kind == pinned_kind == "tag":
                if verified < pinned:
                    mirror.status = Status.STALE
                elif pinned < verified:
                    mirror.status = Status.AHEAD
                    mirror.note = "verified version is newer than the pin"
                else:
                    mirror.status = Status.STALE
                    mirror.note = "version differs only by build metadata"
            else:
                # Git revisions and deleted pre-PR sources have no total ordering in the
                # manifest. A differing valid token is review work, never silently current.
                mirror.status = Status.STALE
            continue

        try:
            verified = parse_semver(mirror.verified)
        except ValueError as error:
            mirror.status, mirror.note = Status.MALFORMED, str(error)
            continue

        versions = locked.get(mirror.crate)
        if versions is None:
            mirror.status = Status.UNKNOWN_CRATE
            mirror.note = "not in rust/Cargo.lock — renamed, dropped, or a typo in the tag"
            continue
        if len(versions) != 1:
            mirror.status = Status.AMBIGUOUS_VERSION
            mirror.pinned = ",".join(sorted(versions))
            mirror.note = "multiple resolved versions; the tag does not identify which one it mirrors"
            continue

        mirror.pinned = next(iter(versions))
        try:
            pinned = parse_semver(mirror.pinned)
        except ValueError as error:
            raise ConfigError(f"invalid Cargo.lock version for {mirror.crate}: {error}") from error

        if mirror.kind == "port":
            mirror.status = Status.FROZEN if mirror.verified != mirror.pinned else Status.CURRENT
        elif mirror.verified == mirror.pinned:
            mirror.status = Status.CURRENT
        elif verified < pinned:
            mirror.status = Status.STALE
        elif pinned < verified:
            mirror.status = Status.AHEAD
            mirror.note = "verified version is newer than the pin"
        else:
            mirror.status = Status.STALE
            mirror.note = "version differs only by build metadata"
    return mirrors


ORDER = {
    Status.MALFORMED: 0,
    Status.UNKNOWN_CRATE: 1,
    Status.AMBIGUOUS_VERSION: 2,
    Status.AHEAD: 3,
    Status.STALE: 4,
    Status.FROZEN: 5,
    Status.CURRENT: 6,
}


def select_mirrors(mirrors: list[Mirror], *, stale_only: bool) -> list[Mirror]:
    return [mirror for mirror in mirrors if mirror.status == Status.STALE] if stale_only else mirrors


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description=__doc__,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--stale-only", action="store_true")
    parser.add_argument("--json", action="store_true")
    parser.add_argument(
        "--check",
        action="store_true",
        help="exit 1 on malformed/ahead/unknown/ambiguous tags (not on stale)",
    )
    args = parser.parse_args(argv)

    try:
        mirrors = classify(find_tags())
    except (ConfigError, OSError, subprocess.CalledProcessError, tomllib.TOMLDecodeError) as error:
        print(f"error: cannot inspect upstream mirrors: {error}", file=sys.stderr)
        return 1

    for mirror in mirrors:
        if not isinstance(mirror.status, Status):
            unknown = mirror.status
            mirror.status = Status.MALFORMED
            mirror.note = f"internal unknown status {unknown!r}"

    mirrors.sort(key=lambda mirror: (
        ORDER[mirror.status],
        mirror.crate,
        mirror.file,
        mirror.line,
    ))
    successful = {Status.CURRENT, Status.STALE, Status.FROZEN}
    bad = [mirror for mirror in mirrors if mirror.status not in successful]
    shown = select_mirrors(mirrors, stale_only=args.stale_only)
    if args.check and args.stale_only:
        shown = [
            mirror
            for mirror in mirrors
            if mirror.status == Status.STALE or mirror.status not in successful
        ]

    if args.json:
        print(json.dumps([asdict(mirror) for mirror in shown], indent=2))
    else:
        for mirror in shown:
            src = f" [{mirror.repo}]" if mirror.repo else ""
            print(
                f"{mirror.status:<18} {mirror.kind:<9} {mirror.crate}@{mirror.verified}{src}"
                f"{'' if mirror.status in (Status.CURRENT, Status.MALFORMED) else f' (pin {mirror.pinned})'}"
            )
            print(f"{'':<18} {mirror.symbol}")
            print(
                f"{'':<18} {mirror.file}:{mirror.line}"
                + (f"  -- {mirror.note}" if mirror.note else "")
            )
        counts: dict[Status, int] = {}
        for mirror in mirrors:
            counts[mirror.status] = counts.get(mirror.status, 0) + 1
        print(
            f"\n{len(mirrors)} mirrors: "
            + ", ".join(f"{count} {status}" for status, count in sorted(counts.items()))
        )

    if args.check:
        if bad:
            print(f"\nerror: {len(bad)} tag(s) need fixing (see above)", file=sys.stderr)
            return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
