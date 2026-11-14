#!/usr/bin/env python3
"""Fetch immutable ancestor dependencies as an optional compiler-cache hint."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time


def fetch(root):
    report = root / '.ci/rust-dependencies'
    report.mkdir(parents=True, exist_ok=True)
    home = root / '.ci/rust-cache/cargo'
    home.mkdir(parents=True, exist_ok=True)
    attempts = []
    with (report / 'fetch.log').open('wb') as log:
        for attempt in range(1, 6):
            started = time.monotonic()
            result = subprocess.run(['cargo', 'fetch', '--locked'], cwd=root / 'rust',
                                    env={**os.environ, 'CARGO_HOME': str(home)},
                                    stdout=log, stderr=subprocess.STDOUT)
            attempts.append({'attempt': attempt, 'exit_code': result.returncode,
                             'elapsed_seconds': time.monotonic() - started})
            if result.returncode == 0:
                break
            if attempt < 5:
                time.sleep(2 ** attempt)
    # A broken intermediate manifest must not block a valid HEAD. Cargo's fresh
    # HEAD commands still fetch/verify missing dependencies and own the verdict.
    usable = attempts[-1]['exit_code'] == 0
    evidence = {'source_sha': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(),
                'lock_sha256': hashlib.sha256((root / 'rust/Cargo.lock').read_bytes()).hexdigest(),
                'usable': usable, 'attempts': attempts,
                'original_log_sha256': hashlib.sha256((report / 'fetch.log').read_bytes()).hexdigest()}
    (report / 'result.json').write_text(json.dumps(evidence, indent=2) + '\n')
    print(json.dumps(evidence))
    if not usable:
        print('Ancestor dependency prefetch unavailable; fresh HEAD Cargo commands must fetch dependencies.')


if __name__ == '__main__':
    fetch(Path.cwd())
