#!/usr/bin/env python3
"""Check whether a released component's build inputs changed since its last release.

This script uses component-specific build closures instead of broad CI path globs.

- Go components: resolves the transitive package dependency graph of the main
  package with `go list -deps -json` and tracks the exact local source files
  used by those packages for Linux amd64/arm64 builds.
- Rust components: resolves the transitive workspace crate dependency graph of
  the package with `cargo metadata` and tracks local crate manifests, build
  scripts, and source trees for the crates in the closure.

The output answers a narrow question:
  "Did any tracked build input for this component change since the last stable
   release tag that is an ancestor of the target ref?"
"""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from collections import deque
from pathlib import Path
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[2]
COMMIT_MARKER = "__COMMIT__"
GO_SOURCE_FIELDS = (
    "GoFiles",
    "CgoFiles",
    "CFiles",
    "CXXFiles",
    "MFiles",
    "HFiles",
    "FFiles",
    "SFiles",
    "SwigFiles",
    "SwigCXXFiles",
    "SysoFiles",
    "EmbedFiles",
)
GO_TARGETS = (("linux", "amd64"), ("linux", "arm64"))
RUST_RELEASE_PLATFORM = "x86_64-unknown-linux-gnu"
RUST_LIBRARY_KINDS = {"lib", "rlib", "dylib", "cdylib", "staticlib", "proc-macro"}
RUST_ROOT = REPO_ROOT / "rust"
SUPPORTED_COMPONENTS: dict[str, dict[str, Any]] = {
    "op-node": {
        "kind": "go",
        "main": "./op-node/cmd",
    },
    "op-batcher": {
        "kind": "go",
        "main": "./op-batcher/cmd",
    },
    "kona-node": {
        "kind": "rust",
        "package": "kona-node",
        "manifest": "rust/kona/bin/node/Cargo.toml",
        "platform": RUST_RELEASE_PLATFORM,
    },
}


class ScriptError(RuntimeError):
    pass


def run(
    *args: str,
    cwd: Path = REPO_ROOT,
    env: dict[str, str] | None = None,
    check: bool = True,
) -> str:
    proc = subprocess.run(
        list(args),
        cwd=cwd,
        env=env,
        text=True,
        capture_output=True,
    )
    if check and proc.returncode != 0:
        raise ScriptError(proc.stderr.strip() or f"command failed: {' '.join(args)}")
    return proc.stdout


def parse_json_stream(text: str) -> list[dict[str, Any]]:
    decoder = json.JSONDecoder()
    idx = 0
    items: list[dict[str, Any]] = []
    length = len(text)
    while idx < length:
        while idx < length and text[idx].isspace():
            idx += 1
        if idx >= length:
            break
        item, end = decoder.raw_decode(text, idx)
        items.append(item)
        idx = end
    return items


def repo_relative(path: str | Path) -> str | None:
    resolved = Path(path).resolve()
    try:
        return resolved.relative_to(REPO_ROOT).as_posix()
    except ValueError:
        return None


def git_changed_files(range_spec: str) -> list[str]:
    return [line for line in run("git", "diff", "--name-only", range_spec).splitlines() if line]


def find_last_stable_tag(component: str, ref: str) -> str:
    stable_tag_re = re.compile(rf"^{re.escape(component)}/v\d+\.\d+\.\d+$")
    tags = run("git", "tag", "--list", f"{component}/v*", "--sort=-version:refname").splitlines()
    for tag in tags:
        if not stable_tag_re.fullmatch(tag):
            continue
        proc = subprocess.run(
            ["git", "merge-base", "--is-ancestor", tag, ref],
            cwd=REPO_ROOT,
            capture_output=True,
            text=True,
        )
        if proc.returncode == 0:
            return tag
    raise ScriptError(
        f"could not find a stable {component} release tag that is an ancestor of {ref}"
    )


def collect_go_inputs(component_cfg: dict[str, Any]) -> tuple[list[str], list[dict[str, Any]], dict[str, Any]]:
    files: set[str] = set()
    packages: dict[str, dict[str, Any]] = {}

    for goos, goarch in GO_TARGETS:
        env = dict(os.environ)
        env.update({"GOOS": goos, "GOARCH": goarch})
        output = run("go", "list", "-deps", "-json", component_cfg["main"], env=env)
        for pkg in parse_json_stream(output):
            pkg_dir = pkg.get("Dir")
            if not pkg_dir:
                continue
            rel_dir = repo_relative(pkg_dir)
            if rel_dir is None:
                continue

            import_path = pkg.get("ImportPath", rel_dir)
            pkg_info = packages.setdefault(
                import_path,
                {
                    "import_path": import_path,
                    "dir": rel_dir,
                    "targets": set(),
                },
            )
            pkg_info["targets"].add(f"{goos}/{goarch}")

            for field in GO_SOURCE_FIELDS:
                for name in pkg.get(field) or []:
                    rel_path = repo_relative(Path(pkg_dir) / name)
                    if rel_path is not None:
                        files.add(rel_path)

    for extra in ("go.mod", "go.sum"):
        if (REPO_ROOT / extra).exists():
            files.add(extra)

    package_list = [
        {
            "import_path": info["import_path"],
            "dir": info["dir"],
            "targets": sorted(info["targets"]),
        }
        for info in sorted(packages.values(), key=lambda item: item["import_path"])
    ]
    metadata = {"targets": [f"{goos}/{goarch}" for goos, goarch in GO_TARGETS]}
    return sorted(files), package_list, metadata


def rust_dep_is_build_relevant(dep: dict[str, Any]) -> bool:
    kinds = dep.get("dep_kinds") or []
    if not kinds:
        return True
    return any(kind.get("kind") in (None, "build") for kind in kinds)


def include_rust_target(target: dict[str, Any], *, is_root: bool, root_name: str) -> bool:
    kinds = set(target.get("kind") or [])
    if "custom-build" in kinds:
        return True
    if kinds & RUST_LIBRARY_KINDS:
        return True
    if is_root and "bin" in kinds and target.get("name") == root_name:
        return True
    return False


def add_rust_source_tree(path: Path, files: set[str]) -> None:
    rel = repo_relative(path)
    if rel is not None:
        files.add(rel)

    if not path.exists():
        return

    # For standard Cargo layouts, include the whole src tree rooted at the target's src dir.
    if path.parent.name == "src":
        for child in path.parent.rglob("*"):
            if child.is_file():
                rel_child = repo_relative(child)
                if rel_child is not None:
                    files.add(rel_child)


def collect_rust_inputs(component_cfg: dict[str, Any]) -> tuple[list[str], list[dict[str, Any]], dict[str, Any]]:
    platform = component_cfg.get("platform", RUST_RELEASE_PLATFORM)
    metadata = json.loads(
        run(
            "cargo",
            "metadata",
            "--format-version",
            "1",
            "--filter-platform",
            platform,
            cwd=RUST_ROOT,
        )
    )

    packages_by_id = {pkg["id"]: pkg for pkg in metadata["packages"]}
    nodes_by_id = {node["id"]: node for node in metadata["resolve"]["nodes"]}

    manifest_rel = component_cfg["manifest"]
    root_manifest = (REPO_ROOT / manifest_rel).resolve()
    root_pkg = None
    for pkg in metadata["packages"]:
        if Path(pkg["manifest_path"]).resolve() == root_manifest:
            root_pkg = pkg
            break
    if root_pkg is None:
        raise ScriptError(f"could not find Rust package for manifest {manifest_rel}")

    reachable: set[str] = set()
    queue: deque[str] = deque([root_pkg["id"]])
    while queue:
        pkg_id = queue.popleft()
        if pkg_id in reachable:
            continue
        reachable.add(pkg_id)
        node = nodes_by_id.get(pkg_id)
        if node is None:
            continue
        for dep in node.get("deps", []):
            if rust_dep_is_build_relevant(dep):
                queue.append(dep["pkg"])

    files: set[str] = set()
    crates: list[dict[str, Any]] = []

    if (RUST_ROOT / "Cargo.toml").exists():
        files.add("rust/Cargo.toml")
    if (RUST_ROOT / "Cargo.lock").exists():
        files.add("rust/Cargo.lock")

    for pkg_id in sorted(reachable):
        pkg = packages_by_id[pkg_id]
        manifest_path = Path(pkg["manifest_path"]).resolve()
        rel_manifest = repo_relative(manifest_path)
        if rel_manifest is None:
            continue

        crate_dir = manifest_path.parent
        files.add(rel_manifest)
        crates.append(
            {
                "name": pkg["name"],
                "dir": crate_dir.relative_to(REPO_ROOT).as_posix(),
                "manifest": rel_manifest,
            }
        )

        is_root = pkg_id == root_pkg["id"]
        for target in pkg.get("targets", []):
            if not include_rust_target(target, is_root=is_root, root_name=root_pkg["name"]):
                continue
            add_rust_source_tree(Path(target["src_path"]), files)

    crate_list = sorted(crates, key=lambda item: (item["name"], item["dir"]))
    return sorted(files), crate_list, {"platform": platform}


def relevant_commits(range_spec: str, tracked_files: set[str], include_merges: bool) -> list[dict[str, Any]]:
    cmd = [
        "git",
        "log",
        "--reverse",
        f"--format={COMMIT_MARKER}%x09%H%x09%s",
        "--name-only",
        "--find-renames",
    ]
    if not include_merges:
        cmd.insert(2, "--no-merges")
    cmd.append(range_spec)

    output = run(*cmd)
    commits: list[dict[str, Any]] = []
    current: dict[str, Any] | None = None

    def flush_current() -> None:
        nonlocal current
        if current is None:
            return
        matching = [path for path in current["files"] if path in tracked_files]
        if matching:
            commits.append(
                {
                    "sha": current["sha"],
                    "short_sha": current["sha"][:12],
                    "subject": current["subject"],
                    "files": matching,
                }
            )
        current = None

    for raw_line in output.splitlines():
        line = raw_line.strip()
        if line.startswith(COMMIT_MARKER + "\t"):
            flush_current()
            _, sha, subject = line.split("\t", 2)
            current = {"sha": sha, "subject": subject, "files": []}
            continue
        if not line:
            continue
        if current is not None:
            current["files"].append(line)

    flush_current()
    return commits


def maybe_fetch(fetch: bool) -> None:
    if fetch:
        subprocess.run(["git", "fetch", "--tags", "origin"], cwd=REPO_ROOT, check=True)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Check whether a component's tracked build inputs changed since its last stable release."
    )
    parser.add_argument(
        "component",
        help="supported components: " + ", ".join(sorted(SUPPORTED_COMPONENTS)),
    )
    parser.add_argument("--ref", default="develop", help="git ref to compare against (default: develop)")
    parser.add_argument("--fetch", action="store_true", help="run 'git fetch --tags origin' first")
    parser.add_argument(
        "--include-merges",
        action="store_true",
        help="include merge commits in the reported commit list",
    )
    parser.add_argument(
        "-v",
        "--verbose",
        action="count",
        default=0,
        help="increase text verbosity; use -vvv to show full package/crate and file detail",
    )
    parser.add_argument("--json", action="store_true", help="emit machine-readable JSON")
    return parser


def main() -> int:
    args = build_parser().parse_args()
    maybe_fetch(args.fetch)

    component_cfg = SUPPORTED_COMPONENTS.get(args.component)
    if component_cfg is None:
        raise ScriptError(
            f"unsupported component '{args.component}'. Supported: {', '.join(sorted(SUPPORTED_COMPONENTS))}"
        )

    last_tag = find_last_stable_tag(args.component, args.ref)
    range_spec = f"{last_tag}..{args.ref}"

    if component_cfg["kind"] == "go":
        tracked_files, build_units, build_metadata = collect_go_inputs(component_cfg)
        build_unit_label = "packages"
    elif component_cfg["kind"] == "rust":
        tracked_files, build_units, build_metadata = collect_rust_inputs(component_cfg)
        build_unit_label = "crates"
    else:
        raise ScriptError(f"unsupported component kind: {component_cfg['kind']}")

    tracked_set = set(tracked_files)
    changed_files = [path for path in git_changed_files(range_spec) if path in tracked_set]
    matching_commits = relevant_commits(range_spec, tracked_set, include_merges=args.include_merges)

    result = {
        "component": args.component,
        "kind": component_cfg["kind"],
        "ref": args.ref,
        "last_release": last_tag,
        "range": range_spec,
        "artifact_changed": bool(changed_files),
        "tracked_files": tracked_files,
        "changed_files": changed_files,
        "matching_commits": matching_commits,
        build_unit_label: build_units,
        "build_metadata": build_metadata,
    }

    if args.json:
        print(json.dumps(result, indent=2))
        return 0

    very_verbose = args.verbose >= 3

    print(f"kind:             {result['kind']}")
    print(f"ref:              {result['ref']}")
    print(f"last_release:     {result['last_release']}")
    print(f"range:            {result['range']}")
    print(f"artifact_changed: {'yes' if result['artifact_changed'] else 'no'}")
    print(f"tracked_files:    {len(result['tracked_files'])}")
    print(f"changed_files:    {len(result['changed_files'])}")
    print(f"matching_commits: {len(result['matching_commits'])}")

    if very_verbose:
        print(f"\n{build_unit_label}:")
        for unit in build_units:
            if component_cfg["kind"] == "go":
                targets = ", ".join(unit["targets"])
                print(f"- {unit['import_path']}  [{targets}]")
                print(f"    dir: {unit['dir']}")
            else:
                print(f"- {unit['name']}")
                print(f"    dir: {unit['dir']}")
                print(f"    manifest: {unit['manifest']}")

    if matching_commits:
        print("\ncommits:")
        for commit in matching_commits:
            print(f"- {commit['short_sha']} {commit['subject']}")
            if very_verbose:
                for path in commit["files"]:
                    print(f"    - {path}")
    else:
        print("\ncommits:\n- none")

    if very_verbose:
        if changed_files:
            print("\nchanged tracked files:")
            for path in changed_files:
                print(f"- {path}")
        else:
            print("\nchanged tracked files:\n- none")

    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ScriptError as exc:
        print(f"error: {exc}", file=sys.stderr)
        raise SystemExit(1)
