#!/usr/bin/env python3
"""Regenerate hack-weights.txt from the logs of green rust-cargo-hack CI jobs.

Usage: scripts/hack-weights-refresh.py <job number>... > scripts/hack-weights.txt

Sums each package's `Finished ... in Ns` lines across both hack steps, scales them
per node to the step's wall time (CircleCI keeps only the last 400 KB of a step's
log, so a noisy node can lose its first runs) and averages across the jobs.
"""

from __future__ import annotations

import json
import re
import sys
import urllib.request
from collections import Counter

JOB_URL = "https://circleci.com/api/v1.1/project/github/ethereum-optimism/optimism/{}"
STEPS = ("Run cargo hack", "Run cargo hack (test targets)")
ANSI = re.compile(r"\x1b\[[0-9;]*m")
RUN = re.compile(r"^info: running `cargo check[^`]*` on (\S+) \(")
FINISHED = re.compile(r"Finished `.* in (?:(\d+)m )?([\d.]+)s")

HEADER = """\
# Relative cost of each package's rust-cargo-hack checks, read by hack-partition.py.
# Columns: <package> <seconds in CI, both hack steps>
#
# Only balance depends on these: a package with no row gets a default weight and is
# still checked. Regenerate with hack-weights-refresh.py when the job's slowest node
# drifts well above the mean. This table: {}.
"""


def get_json(url: str):
    request = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(request, timeout=120) as response:
        return json.load(response)


def package_seconds(log: str) -> Counter[str]:
    """Seconds per package in one node's log of one step."""
    seconds: Counter[str] = Counter()
    package = None
    for line in ANSI.sub("", log).splitlines():
        if run := RUN.match(line):
            package = run[1]
        elif package and (finished := FINISHED.search(line)):
            seconds[package] += int(finished[1] or 0) * 60 + float(finished[2])
            package = None
    return seconds


def job_seconds(job: str) -> Counter[str]:
    totals: Counter[str] = Counter()
    for step in get_json(JOB_URL.format(job))["steps"]:
        if step["name"] not in STEPS:
            continue
        for action in step["actions"]:
            log = "".join(part.get("message", "") for part in get_json(action["output_url"]))
            seconds = package_seconds(log)
            scale = action["run_time_millis"] / 1000 / max(sum(seconds.values()), 1)
            for package, value in seconds.items():
                totals[package] += value * scale
    return totals


def main(jobs: list[str]) -> int:
    if not jobs:
        raise SystemExit(__doc__)
    totals: Counter[str] = Counter()
    for job in jobs:
        totals.update(job_seconds(job))
    if not totals:
        raise SystemExit("hack-weights-refresh: no cargo-hack runs found in those jobs")
    print(HEADER.format("mean of jobs " + ", ".join(jobs)))
    for package in sorted(totals):
        print(package, max(1, round(totals[package] / len(jobs))))
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
