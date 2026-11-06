#!/usr/bin/env python3
"""Print the workspace packages one rust-cargo-hack node checks, one per line.

Usage: hack-partition.py <index>/<total>     (1-based, e.g. 3/10)

cargo-hack's own --partition cuts its run list by run count, which ignores what the
runs cost: a few reth crates take as long as dozens of leaf crates, so one node gets
several times another's work. This cuts the same workspace-member order by cost from
hack-weights.txt instead. Ranges stay contiguous because neighbouring crates share a
dependency tree, and each node that needs a tree pays to restore it.

The workspace comes from `cargo metadata`, never from the table, and the split is a
pure function of the member list, the table and <total>. So every package lands on
exactly one node, and a stale table costs balance but can't skip a check.
"""

from __future__ import annotations

import json
import re
import subprocess
import sys
from itertools import accumulate
from pathlib import Path

WEIGHTS_FILE = Path(__file__).with_name("hack-weights.txt")

# Roughly the median package. Only used for a package with no row in the table.
DEFAULT_WEIGHT = 20


def parse_partition(partition: str) -> tuple[int, int]:
    match = re.fullmatch(r"(\d+)/(\d+)", partition)
    if not match or not 1 <= int(match[1]) <= int(match[2]):
        raise ValueError(f"invalid partition {partition!r}: expected <index>/<total>")
    return int(match[1]), int(match[2])


def read_weights(text: str) -> dict[str, int]:
    rows = (line.split("#")[0].split() for line in text.splitlines())
    return {row[0]: int(row[1]) for row in rows if row}


def split(packages: list[str], weights: dict[str, int], nodes: int) -> list[list[str]]:
    """Cut packages into `nodes` non-empty contiguous ranges.

    Minimises the heaviest range, then the sum of squared range weights so the
    remaining ranges come out even. Ties go to the earliest cut, for determinism.
    """
    count = len(packages)
    if nodes > count:
        raise ValueError(f"{nodes} nodes for {count} packages: every node needs a package")
    prefix = [0, *accumulate(weights.get(package, DEFAULT_WEIGHT) for package in packages)]

    # best[k][i] is (heaviest, sum of squares, last cut) for the first i packages in k ranges.
    best: list[dict[int, tuple[int, int, int]]] = [{0: (0, 0, 0)}]
    for k in range(1, nodes + 1):
        best.append({})
        for i in range(k, count - (nodes - k) + 1):
            best[k][i] = min(
                (
                    max(best[k - 1][j][0], prefix[i] - prefix[j]),
                    best[k - 1][j][1] + (prefix[i] - prefix[j]) ** 2,
                    j,
                )
                for j in range(k - 1, i)
                if j in best[k - 1]
            )

    ranges = []
    end = count
    for k in range(nodes, 0, -1):
        start = best[k][end][2]
        ranges.append(packages[start:end])
        end = start
    return ranges[::-1]


def workspace_packages() -> list[str]:
    metadata = subprocess.run(
        ["cargo", "metadata", "--no-deps", "--format-version", "1"],
        check=True,
        stdout=subprocess.PIPE,
    ).stdout
    return [package["name"] for package in json.loads(metadata)["packages"]]


def main(argv: list[str]) -> int:
    if len(argv) != 1:
        raise SystemExit(__doc__)
    try:
        index, total = parse_partition(argv[0])
        ranges = split(workspace_packages(), read_weights(WEIGHTS_FILE.read_text()), total)
    except ValueError as error:
        raise SystemExit(f"hack-partition: {error}") from error
    print("\n".join(ranges[index - 1]))
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
