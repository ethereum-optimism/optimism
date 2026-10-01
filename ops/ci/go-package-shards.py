#!/usr/bin/env python3
"""Build and validate exhaustive, provider-neutral Go package partitions."""

import argparse
import json
import re
import sys
from pathlib import Path


MAX_SHARDS = 256


def shard_number(value, *, total=False):
    if not re.fullmatch(r"0|[1-9][0-9]*", str(value)):
        raise ValueError("shard numbers must be decimal integers")
    number = int(value)
    if total and not 1 <= number <= MAX_SHARDS:
        raise ValueError(f"shard total must be between 1 and {MAX_SHARDS}")
    return number


def validate_packages(packages, prefix):
    if not isinstance(packages, list) or not packages:
        raise ValueError("package manifest must contain at least one package")
    for package in packages:
        if not isinstance(package, str) or not re.fullmatch(r"[A-Za-z0-9_./-]+", package):
            raise ValueError("invalid Go import path in manifest")
        if any(segment in ("", ".", "..") for segment in package.split("/")):
            raise ValueError("invalid Go import path segments in manifest")
        if package != prefix and not package.startswith(prefix + "/"):
            raise ValueError(f"package is outside the selected scope: {package}")
    if len(set(packages)) != len(packages):
        raise ValueError("duplicate package in manifest")


def read_go_packages(source, prefix):
    decoder = json.JSONDecoder()
    packages = []
    offset = 0
    while offset < len(source):
        while offset < len(source) and source[offset].isspace():
            offset += 1
        if offset == len(source):
            break
        package, offset = decoder.raw_decode(source, offset)
        if not isinstance(package, dict):
            raise ValueError("go list must emit JSON package objects")
        # go list -e exits successfully even for missing embeds or dependencies.
        if package.get("Error") or package.get("DepsErrors"):
            raise ValueError(f"go list reported an error for {package.get('ImportPath', '<unknown>')}")
        packages.append(package.get("ImportPath"))
    validate_packages(packages, prefix)
    return sorted(packages)


def create_manifest(packages, prefix, total):
    total = shard_number(total, total=True)
    validate_packages(packages, prefix)
    packages = sorted(packages)
    return {
        "version": 1,
        "prefix": prefix,
        "total": total,
        "packages": packages,
        "shards": [packages[index::total] for index in range(total)],
    }


def select_packages(manifest, prefix, index, total):
    total = shard_number(total, total=True)
    index = shard_number(index)
    if index >= total:
        raise ValueError("shard index must be smaller than shard total")
    if not isinstance(manifest, dict) or type(manifest.get("version")) is not int or manifest["version"] != 1:
        raise ValueError("unsupported package manifest")
    if manifest.get("prefix") != prefix or type(manifest.get("total")) is not int or manifest["total"] != total:
        raise ValueError("manifest scope or shard total does not match this task")
    packages = manifest.get("packages")
    validate_packages(packages, prefix)
    expected = create_manifest(packages, prefix, total)
    # Equality checks the exact union, duplicate-free assignment, and stable order.
    if manifest.get("shards") != expected["shards"] or packages != expected["packages"]:
        raise ValueError("shards must partition the complete sorted package manifest exactly once")
    return manifest["shards"][index]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    create = commands.add_parser("create")
    create.add_argument("--prefix", required=True)
    create.add_argument("--total", required=True)
    create.add_argument("--output", type=Path, required=True)
    select = commands.add_parser("select")
    select.add_argument("--prefix", required=True)
    select.add_argument("--total", required=True)
    select.add_argument("--index", required=True)
    select.add_argument("--manifest", type=Path, required=True)
    args = parser.parse_args()
    try:
        if args.command == "create":
            packages = read_go_packages(sys.stdin.read(), args.prefix)
            manifest = create_manifest(packages, args.prefix, args.total)
            args.output.write_text(json.dumps(manifest, indent=2) + "\n")
        else:
            manifest = json.loads(args.manifest.read_text())
            packages = select_packages(manifest, args.prefix, args.index, args.total)
            if packages:
                print("\n".join(packages))
    except (ValueError, OSError) as error:
        print(f"ERROR: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
