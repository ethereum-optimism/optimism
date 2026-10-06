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
    entries = {}
    digest = hashlib.sha256(b'rwx-cargo-source-v3\0')
    for path in files:
        name = str(path.relative_to(root))
        content = hashlib.sha256(path.read_bytes()).hexdigest()
        entries[name] = {'sha256': content}
        digest.update(name.encode() + b'\0')
        digest.update(bytes.fromhex(content))
    return entries, digest.hexdigest()


def manage(phase, root, target):
    target.mkdir(parents=True, exist_ok=True)
    state = target / '.rwx-source-fingerprint.json'
    pending = target / '.rwx-source-pending.json'
    entries, digest = sources(root)
    if not entries:
        raise ValueError('No Rust build source inputs')
    if phase == 'prepare':
        previous = json.loads(state.read_text()) if state.exists() else {}
        # A pending manifest means a prior build did not commit. Its partial
        # targets may be newer than the last successful source, even after a
        # source rollback. Refresh all inputs before reusing those targets.
        reusable = previous.get('version') == 3 and not pending.exists()
        old_entries = previous.get('files', {}) if reusable else {}
        changed = previous.get('source_sha256') != digest or not reusable
        stamp = time.time_ns()
        restored = 0
        for name, entry in entries.items():
            old = old_entries.get(name, {})
            if old.get('sha256') == entry['sha256'] and type(old.get('mtime_ns')) is int:
                entry['mtime_ns'] = old['mtime_ns']
                restored += 1
            else:
                entry['mtime_ns'] = stamp
            os.utime(root / name, ns=(entry['mtime_ns'], entry['mtime_ns']))
        if changed:
            state.unlink(missing_ok=True)
        # Commit this per-file map only after the caller completes its build.
        pending.write_text(json.dumps({'version': 3, 'source_sha256': digest,
                                       'files': entries}) + '\n')
        print(json.dumps({'source_sha256': digest, 'source_changed': changed,
                          'restored_source_files': restored,
                          'refreshed_source_files': len(entries) - restored}))
    elif phase == 'commit':
        if not pending.exists() or json.loads(pending.read_text()).get('source_sha256') != digest:
            raise ValueError('Rust source changed during build or preparation is missing')
        # A failed build never reaches commit, so partial targets stay untrusted.
        pending.replace(state)
    else:
        raise ValueError('Expected prepare or commit')


if __name__ == '__main__':
    manage(sys.argv[1], Path.cwd(), Path(os.environ['CARGO_TARGET_DIR']))
