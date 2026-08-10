#!/usr/bin/env python3
"""Report OP code that mirrors upstream reth/revm/alloy logic and has fallen behind the pin.

Reads `UPSTREAM-MIRROR` tags out of the Rust sources and compares each tag's
last-verified version against what the workspace currently resolves to, so a bump
gets a worklist proportional to what actually drifted rather than a checklist.

Tag grammar, in the doc comment of the mirroring item:

    /// UPSTREAM-MIRROR(<kind>): <crate>@<version> <upstream::symbol::path>

  <kind>     override | copy | delegate | set | port   (see docs/ai/reth-upstream-mirrors.md)
  <crate>    crates.io package name, or `reth` for the git-pinned family
  <version>  exact semver for crates.io; `rev:<sha7>`, `v<tag>` or `pre-<pr>` for reth
  <symbol>   full upstream path, used to locate the item in upstream sources

Usage:
  upstream-mirrors.py                 list every mirror with its status
  upstream-mirrors.py --stale-only    only the ones behind the pin (the bump worklist)
  upstream-mirrors.py --json          machine-readable, for CI comments
  upstream-mirrors.py --check         exit 1 on `ahead` or `unknown-crate` (never on `stale`)
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from dataclasses import asdict, dataclass
from pathlib import Path

RUST_ROOT = Path(__file__).resolve().parent.parent

TAG_RE = re.compile(
    r"UPSTREAM-MIRROR\((?P<kind>[a-z]+)\):\s*"
    r"(?P<crate>[A-Za-z0-9_-]+)@(?P<version>[A-Za-z0-9_.:-]+)\s+"
    r"(?P<symbol>[A-Za-z0-9_:]+)"
)
DOC_RE = re.compile(r"^\s*(///|//!)\s?(.*)$")
KINDS = {"override", "copy", "delegate", "set", "port"}


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
    status: str = ""
    note: str = ""


def semver(v: str) -> tuple[int, ...]:
    return tuple(int(p) for p in re.findall(r"\d+", v)[:3])


def find_tags() -> list[Mirror]:
    files = subprocess.run(
        ["git", "ls-files", "-z", "*.rs"],
        cwd=RUST_ROOT, capture_output=True, text=True, check=True,
    ).stdout.split("\0")
    out: list[Mirror] = []
    for rel in filter(None, files):
        path = RUST_ROOT / rel
        try:
            text = path.read_text(encoding="utf8", errors="replace")
        except OSError:
            continue
        if "UPSTREAM-MIRROR" not in text:
            continue
        lines = text.splitlines()
        for n, line in enumerate(lines, 1):
            if "UPSTREAM-MIRROR" not in line:
                continue
            # rustfmt owns comment wrapping (`wrap_comments`), so a tag may be split across
            # several `///` lines. Rejoin the paragraph before parsing rather than requiring
            # a layout rustfmt would undo.
            para, i = [], n - 1
            while i < len(lines) and (dm := DOC_RE.match(lines[i])) and dm.group(2).strip():
                para.append(dm.group(2).strip())
                i += 1
            if m := TAG_RE.search(" ".join(para)):
                out.append(Mirror(file=rel, line=n, verified=m.group("version"),
                                  **{k: m.group(k) for k in ("kind", "crate", "symbol")}))
            else:
                out.append(Mirror(kind="?", crate="?", verified="?", symbol="?",
                                  file=rel, line=n, status="malformed",
                                  note="does not match the tag grammar"))
    return out


def locked_versions() -> dict[str, str]:
    """Every crates.io package version the main workspace resolves to."""
    text = (RUST_ROOT / "Cargo.lock").read_text()
    versions: dict[str, str] = {}
    for name, ver in re.findall(
        r'\[\[package\]\]\nname = "([^"]+)"\nversion = "([^"]+)"', text
    ):
        # A crate can appear at several majors; keep the highest.
        if name not in versions or semver(ver) > semver(versions[name]):
            versions[name] = ver
    return versions


def reth_pin() -> tuple[str, str]:
    """Returns (pin token, source repo) for the reth family.

    The repo is read from the manifest rather than assumed: the pin has moved between
    `paradigmxyz/reth` and OP's `op-rs/reth` fork, and a hardcoded repo turns that move
    into a silent "unknown" pin rather than a visible one. Tags name the family (`reth`),
    not the repo, so a move like that does not invalidate every tag.
    """
    text = (RUST_ROOT / "Cargo.toml").read_text()
    repos = set(re.findall(r'git = "https://github\.com/([\w.-]+/reth)"', text))
    repo = repos.pop() if len(repos) == 1 else ("/".join(sorted(repos)) or "unknown")
    if m := re.search(r'/reth"[^\n]*\btag = "([^"]+)"', text):
        return m.group(1), repo
    if m := re.search(r'/reth"[^\n]*\brev = "([0-9a-f]+)"', text):
        return f"rev:{m.group(1)[:7]}", repo
    return "unknown", repo


def classify(mirrors: list[Mirror]) -> list[Mirror]:
    locked, (reth, reth_repo) = locked_versions(), reth_pin()
    for m in mirrors:
        if m.status == "malformed":
            continue
        if m.kind not in KINDS:
            m.status, m.note = "malformed", f"unknown kind {m.kind!r}"
            continue

        if m.crate == "reth":
            m.pinned, m.repo = reth, reth_repo
            # Ports are pinned to an old upstream on purpose; report the distance, never
            # treat it as a bump obligation.
            if m.kind == "port":
                m.status = "frozen" if m.verified != reth else "current"
            else:
                m.status = "current" if m.verified == reth else "stale"
            continue

        if m.crate not in locked:
            m.status = "unknown-crate"
            m.note = "not in rust/Cargo.lock — renamed, dropped, or a typo in the tag"
            continue

        m.pinned = locked[m.crate]
        if m.kind == "port":
            m.status = "frozen" if m.verified != m.pinned else "current"
        elif semver(m.verified) == semver(m.pinned):
            m.status = "current"
        elif semver(m.verified) < semver(m.pinned):
            m.status = "stale"
        else:
            m.status = "ahead"
            m.note = "verified version is newer than the pin — likely a bad find/replace"
    return mirrors


ORDER = {"malformed": 0, "unknown-crate": 1, "ahead": 2, "stale": 3, "frozen": 4, "current": 5}


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--stale-only", action="store_true")
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--check", action="store_true",
                    help="exit 1 on malformed/ahead/unknown-crate tags (not on stale)")
    args = ap.parse_args()

    mirrors = classify(find_tags())
    mirrors.sort(key=lambda m: (ORDER.get(m.status, 9), m.crate, m.file, m.line))
    shown = [m for m in mirrors if m.status != "current"] if args.stale_only else mirrors

    if args.json:
        print(json.dumps([asdict(m) for m in shown], indent=2))
    else:
        for m in shown:
            src = f" [{m.repo}]" if m.repo else ""
            print(f"{m.status:<14} {m.kind:<9} {m.crate}@{m.verified}{src}"
                  f"{'' if m.status in ('current', 'malformed') else f' (pin {m.pinned})'}")
            print(f"{'':<14} {m.symbol}")
            print(f"{'':<14} {m.file}:{m.line}" + (f"  -- {m.note}" if m.note else ""))
        counts: dict[str, int] = {}
        for m in mirrors:
            counts[m.status] = counts.get(m.status, 0) + 1
        print(f"\n{len(mirrors)} mirrors: "
              + ", ".join(f"{n} {s}" for s, n in sorted(counts.items())))

    if args.check:
        bad = [m for m in mirrors if m.status in ("malformed", "ahead", "unknown-crate")]
        if bad:
            print(f"\nerror: {len(bad)} tag(s) need fixing (see above)", file=sys.stderr)
            return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
