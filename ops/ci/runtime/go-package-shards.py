#!/usr/bin/env python3
"""Read `go list -json` output and place Go packages on shards."""

import json
import statistics


def read_go_packages(source):
    """Return the package objects in `go list -e -json` output. Fail if a package has an error."""
    decoder, offset, packages = json.JSONDecoder(), 0, []
    # go list prints concatenated JSON objects, not a JSON array.
    while offset < len(source):
        if source[offset].isspace():
            offset += 1
            continue
        package, offset = decoder.raw_decode(source, offset)
        # go list -e exits successfully even for missing embeds or dependencies.
        if package.get("Error") or package.get("DepsErrors"):
            raise ValueError(f"go list reported an error for {package.get('ImportPath', '<unknown>')}")
        packages.append(package)
    return packages


def assign(packages, total, durations):
    """Return `total` sorted package lists. Each package is in exactly one list.

    `durations` maps a package to its seconds. A package without a duration gets
    the median of the known ones. The longest package goes first, onto the shard
    with the smallest load. Durations change the placement only.
    """
    known = [durations[package] for package in packages if package in durations]
    fallback = max(0.001, statistics.median(known)) if known else 1.0
    weights = {package: durations.get(package, fallback) for package in packages}
    shards, loads = [[] for _ in range(total)], [0.0] * total
    for package in sorted(packages, key=lambda package: (-weights[package], package)):
        index = min(range(total), key=lambda index: (loads[index], len(shards[index]), index))
        shards[index].append(package)
        loads[index] += weights[package]
    return [sorted(shard) for shard in shards]
