#!/usr/bin/env python3
"""Prepare an optional ancestor compiler cache; never provide a HEAD verdict."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys


def prepare(job, root):
    if job not in ('tests-build', 'features'):
        raise ValueError('Expected tests-build or features')
    index = os.environ.get('CI_RUST_PARTITION_INDEX', '0')
    name = 'features-' + index if job == 'features' else job
    report = root / '.ci/rust-workspace' / name
    result = subprocess.run(['bash', str(root / 'ops/ci/runtime/rust-workspace.sh'), job], cwd=root)
    # An intermediate ancestor can be broken even though HEAD fixes it. Retain
    # its original failure and discard partial targets, then let fresh HEAD
    # workers compile normally. A fallback is never reported as a warm cache.
    evidence = root / '.ci/rust-workspace/compiler-baseline' / name
    evidence.mkdir(parents=True, exist_ok=True)
    if report.exists():
        for path in report.iterdir():
            if path.is_file() and path.suffix in ('.json', '.log', '.xml'):
                shutil.copyfile(path, evidence / path.name)
    target = root / 'rust/target'
    if result.returncode and target.exists():
        # Retain originals even if cleanup fails, but never claim a clean
        # fallback or allow downstream compilation with failed cleanup.
        shutil.rmtree(target)
    (evidence / 'compiler-baseline.json').write_text(json.dumps({
        'source_sha': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(),
        'build_exit_code': result.returncode, 'usable': result.returncode == 0,
        'fallback': 'none' if result.returncode == 0 else 'clean HEAD compilation',
    }, indent=2) + '\n')


if __name__ == '__main__':
    prepare(sys.argv[1], Path.cwd())
