#!/usr/bin/env python3
"""Prefilters upstream reth fixes newer than a fork pin for a forward sweep.

Run inside a reth clone that has the fork pin, upstream main and the upstream
release tags (v*) fetched:

    reth-forward-prefilter.py <pin> <upstream-main>

It prints the upstream tip as a `#` comment line, then a tab-separated table
with one row per fix, perf, security or revert commit that is newer than the
pin and not patch-equivalent to a commit on it. Candidates are the commits on
<upstream-main> after the pin's base and the commits of newer release tags,
deduplicated by patch id. Columns:

    release  first release tag newer than the pin's base that contains the
             commit or a patch-equivalent backport, else main-only
    source   main, or release-only for a commit no main commit matches
    picked   picked if a pin commit names it in a `(cherry picked from
             commit ...)` line (a hand-port), else -
    files    touched source files that exist at the pin, out of all touched
    apply    clean or conflict: whether the patch applies to the pin as is
             (- for a picked commit)
    sha, date, subject
"""

import os
import re
import subprocess
import sys
import tempfile

FIX_SUBJECT = re.compile(r"^(fix|perf|security|revert)", re.IGNORECASE)
NON_SOURCE = re.compile(r"^(\.github|docs|book)/|Cargo\.lock$|\.mdx?$")
PICKED_LINE = re.compile(r"^\(cherry picked from commit ([0-9a-f]{40})\)$", re.MULTILINE)


def git(*args, input=None, env=None, check=True):
    return subprocess.run(
        ["git", *args], input=input, env=env, check=check, capture_output=True, text=True
    )


def patch_ids(*rev_args):
    """Maps patch id -> commit for the non-merge commits of a rev range."""
    log = git("log", "--no-merges", "-p", "--format=commit %H", *rev_args).stdout
    if not log:
        return {}
    out = git("patch-id", "--stable", input=log).stdout
    ids = {}
    for line in out.splitlines():
        pid, sha = line.split()
        ids.setdefault(pid, sha)
    return ids


def newer_releases(base, pin):
    tags = git("tag", "--list", "v[0-9]*", "--contains", base, "--sort=v:refname").stdout.split()
    return [t for t in tags if git("merge-base", "--is-ancestor", t, pin, check=False).returncode]


def commit_info(sha):
    date, subject = git("log", "-1", "--format=%cs%x09%s", sha).stdout.rstrip("\n").split("\t", 1)
    return date, subject


def main(argv):
    if len(argv) != 3:
        print(f"usage: {argv[0]} <pin> <upstream-main>", file=sys.stderr)
        return 2
    pin = git("rev-parse", "--verify", f"{argv[1]}^{{commit}}").stdout.strip()
    upstream = git("rev-parse", "--verify", f"{argv[2]}^{{commit}}").stdout.strip()
    base = git("merge-base", pin, upstream).stdout.strip()
    releases = newer_releases(base, pin)

    carried = set(patch_ids(f"{base}..{pin}"))
    picked = set(PICKED_LINE.findall(git("log", "--format=%B", f"{base}..{pin}").stdout))

    first_release = {}
    release_commits = {}
    seen = [base]
    for tag in releases:
        for pid, sha in patch_ids(tag, "--not", *seen).items():
            first_release.setdefault(pid, tag)
            release_commits.setdefault(pid, sha)
        seen.append(tag)

    main_commits = patch_ids(f"{base}..{upstream}")
    candidates = [(pid, sha, "main") for pid, sha in main_commits.items()]
    candidates += [
        (pid, sha, "release-only")
        for pid, sha in release_commits.items()
        if pid not in main_commits
    ]

    pin_files = set(git("ls-tree", "-r", "--name-only", pin).stdout.splitlines())
    rows = []
    with tempfile.TemporaryDirectory() as tmp:
        env = {**os.environ, "GIT_INDEX_FILE": os.path.join(tmp, "index")}
        git("read-tree", pin, env=env)
        for pid, sha, source in candidates:
            if pid in carried:
                continue
            date, subject = commit_info(sha)
            if not FIX_SUBJECT.match(subject):
                continue
            files = [
                f
                for f in git("show", "--format=", "--name-only", sha).stdout.splitlines()
                if f and not NON_SOURCE.search(f)
            ]
            present = sum(f in pin_files for f in files)
            if sha in picked:
                apply = "-"
            else:
                patch = git("format-patch", "-1", "--stdout", sha).stdout
                ok = git("apply", "--cached", "--check", "-", input=patch, env=env, check=False)
                apply = "clean" if ok.returncode == 0 else "conflict"
            rows.append(
                (
                    date,
                    first_release.get(pid, "main-only"),
                    source,
                    "picked" if sha in picked else "-",
                    f"{present}/{len(files)}",
                    apply,
                    sha[:10],
                    subject,
                )
            )

    tip = git("log", "-1", "--format=%H %cI", upstream).stdout.strip()
    print(f"# upstream tip: {tip}")
    print("release\tsource\tpicked\tfiles\tapply\tsha\tdate\tsubject")
    for date, *cols, sha, subject in sorted(rows, reverse=True):
        print("\t".join([*cols, sha, date, subject]))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
