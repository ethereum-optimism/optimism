#!/usr/bin/env python3
"""Keep Cargo source freshness correct when restoring targets across checkouts."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time


def sources(root):
    listing = subprocess.check_output(['git', 'ls-files', '-z', '--cached', '--others',
                                      '--exclude-standard', '--', 'rust', 'mise.toml',
                                      'op-core/nuts/bundles'], cwd=root)
    paths = sorted(set(name.decode() for name in listing.split(b'\0') if name))
    files = [root / name for name in paths if (root / name).is_file()
             and not name.startswith('rust/target/')]
    digest = hashlib.sha256(b'rwx-cargo-source-v2\0')
    for path in files:
        digest.update(str(path.relative_to(root)).encode() + b'\0')
        digest.update(hashlib.sha256(path.read_bytes()).digest())
    return files, digest.hexdigest()


def manage(phase, root, target):
    target.mkdir(parents=True, exist_ok=True)
    state = target / '.rwx-source-fingerprint.json'
    pending = target / '.rwx-source-pending.json'
    files, digest = sources(root)
    if not files:
        raise ValueError('No Rust build source inputs')
    if phase == 'prepare':
        previous = json.loads(state.read_text()) if state.exists() else {}
        changed = previous.get('source_sha256') != digest or type(previous.get('source_mtime_ns')) is not int
        stamp = time.time_ns() if changed else previous['source_mtime_ns']
        if changed:
            state.unlink(missing_ok=True)
        # Restore a stable content-version timestamp across normalized checkouts.
        # Older workspace outputs become dirty; unchanged compiled targets stay fresh.
        for path in files:
            os.utime(path, ns=(stamp, stamp))
        pending.write_text(json.dumps({'source_sha256': digest, 'source_mtime_ns': stamp}) + '\n')
        print(json.dumps({'source_sha256': digest, 'source_changed': changed,
                          'restored_source_files': len(files)}))
    elif phase == 'commit':
        if not pending.exists() or json.loads(pending.read_text()).get('source_sha256') != digest:
            raise ValueError('Rust source changed during build or preparation is missing')
        # A failed build never reaches commit, so partial targets stay untrusted.
        pending.replace(state)
    else:
        raise ValueError('Expected prepare or commit')


if __name__ == '__main__':
    manage(sys.argv[1], Path.cwd(), Path(os.environ['CARGO_TARGET_DIR']))
