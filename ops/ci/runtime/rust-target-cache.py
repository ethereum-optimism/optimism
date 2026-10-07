#!/usr/bin/env python3
"""Keep Cargo source freshness correct when restoring targets across checkouts."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
import sys
import tempfile
import time

GENERATED_SOURCES = ('rust/op-reth/crates/chainspec/res/superchain-configs.tar',)


def generated_sources(root):
    # prepare-superchain/build.rs own checksum verification and materialization.
    # Cargo declares this ignored archive as a build-script input. Its timestamp
    # must survive a cache restore just like a tracked source file's timestamp.
    return {name: {'sha256': hashlib.sha256((root / name).read_bytes()).hexdigest()}
            for name in GENERATED_SOURCES if (root / name).is_file()}


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


def namespace(root):
    """Keep source edits incremental; reset large targets when build inputs change.

    RWX retains the initial tool-cache layer across incremental writes. A new
    dependency graph can otherwise add a second full build to that old layer.
    This identity covers manifests, compiler configuration and actual tools.
    The caller adds the profile mode to its task-specific cache name.
    """
    names = subprocess.check_output(['git', 'ls-files', '-z', '--', 'rust',
                                    'ops/ci/runtime/rust-workspace.sh',
                                    'ops/ci/runtime/rust-workspace-report.py',
                                    'ops/ci/runtime/rust-target-cache.py',
                                    'rust/op-reth/crates/chainspec/res/superchain-configs.tar.sha256'], cwd=root).decode().split('\0')
    names = sorted(name for name in names if name and
                   (name.endswith('/Cargo.toml') or name in ('rust/Cargo.lock', 'rust/.cargo/config.toml',
                    'rust/rust-toolchain.toml', 'ops/ci/runtime/rust-workspace.sh',
                    'ops/ci/runtime/rust-workspace-report.py',
                    'rust/op-reth/crates/chainspec/res/superchain-configs.tar.sha256',
                    'ops/ci/runtime/rust-target-cache.py')))
    if not {'rust/Cargo.lock', 'rust/Cargo.toml'} <= set(names):
        raise ValueError('Missing tracked Rust cache configuration')
    binding = {'input_sha256': {name: hashlib.sha256((root / name).read_bytes()).hexdigest() for name in names},
               'tools': {name: subprocess.check_output(argv, cwd=root, text=True).strip() for name, argv in
                         (('rustc', ['rustc', '-Vv']), ('cargo', ['cargo', '-V']), ('sccache', ['sccache', '--version']))},
               'flags': {name: value for name, value in sorted(os.environ.items()) if name in
                         ('RUSTFLAGS', 'CARGO_ENCODED_RUSTFLAGS', 'CARGO_BUILD_RUSTFLAGS') or
                         name.startswith('CARGO_PROFILE_')}}
    digest = hashlib.sha256(json.dumps(binding, sort_keys=True).encode()).hexdigest()
    return {'namespace': digest, **binding,
            'source_sha': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()}


def detach_metadata(target):
    """Publish independent host metadata files with identical bytes and mtimes.

    Rust hardlinks deps/*.rmeta to incremental session metadata. A restored
    shared entry returned ESTALE during a hosted rebuild. Keep incremental
    data, but do not publish that shared inode as a writable compiler output.
    """
    count = size = 0
    for path in target.glob('*/deps/*.rmeta'):
        entry = path.lstat()
        if entry.st_nlink <= 1 or not stat.S_ISREG(entry.st_mode):
            continue
        fd, name = tempfile.mkstemp(prefix='.rwx-metadata-', dir=path.parent)
        os.close(fd)
        temporary = Path(name)
        try:
            shutil.copyfile(path, temporary)
            shutil.copystat(path, temporary)
            temporary.replace(path)
        finally:
            temporary.unlink(missing_ok=True)
        count += 1
        size += entry.st_size
    return {'detached_metadata_files': count, 'detached_metadata_bytes': size}


def snapshot(phase, target, archive):
    """Pack compiler output; restore its original permissions and nanosecond mtimes.

    RWX owns cache byte transport. Cargo owns source/compiler invalidation.
    The caller removes test reports before packing. A failed pack leaves the
    previous snapshot intact; an unreadable snapshot fails before compilation.
    """
    if phase == 'restore':
        if not archive.exists():
            print(json.dumps({'restored': False}))
            return
        if target.exists():
            shutil.rmtree(target)
        target.mkdir(parents=True, exist_ok=True)
        subprocess.run(['tar', '--zstd', '--extract', '--file', str(archive),
                        '--directory', str(target)], check=True)
        print(json.dumps({'restored': True, 'archive_bytes': archive.stat().st_size}))
    else:
        archive.parent.mkdir(parents=True, exist_ok=True)
        fd, name = tempfile.mkstemp(prefix='.target-cache-', dir=archive.parent)
        os.close(fd)
        temporary = Path(name)
        try:
            subprocess.run(['tar', '--use-compress-program=zstd -T0 -3', '--sort=name',
                            '--format=pax', '--pax-option=delete=atime,delete=ctime',
                            '--create', '--file', str(temporary), '--directory', str(target),
                            '.'], check=True)
            temporary.replace(archive)
        finally:
            temporary.unlink(missing_ok=True)
        print(json.dumps({'archive_bytes': archive.stat().st_size}))


def manage(phase, root, target):
    target.mkdir(parents=True, exist_ok=True)
    state = target / '.rwx-source-fingerprint.json'
    pending = target / '.rwx-source-pending.json'
    entries, digest = sources(root)
    generated = generated_sources(root)
    if not entries:
        raise ValueError('No Rust build source inputs')
    if phase == 'prepare':
        previous = json.loads(state.read_text()) if state.exists() else {}
        # A pending manifest means a prior build did not commit. Its partial
        # targets may be newer than the last successful source, even after a
        # source rollback. Refresh all inputs before reusing those targets.
        reusable = previous.get('version') == 3 and not pending.exists()
        old_entries = previous.get('files', {}) if reusable else {}
        old_generated = previous.get('generated_files', {}) if reusable else {}
        changed = previous.get('source_sha256') != digest or not reusable
        changed |= {name: row['sha256'] for name, row in old_generated.items()} != {
            name: row['sha256'] for name, row in generated.items()}
        stamp = time.time_ns()
        restored = 0
        restored_generated = 0
        refreshed_generated = 0
        recorded_generated = 0
        for current, old_files in ((entries, old_entries), (generated, old_generated)):
            for name, entry in current.items():
                old = old_files.get(name, {})
                if old.get('sha256') == entry['sha256'] and type(old.get('mtime_ns')) is int:
                    entry['mtime_ns'] = old['mtime_ns']
                    if current is entries: restored += 1
                    else: restored_generated += 1
                elif current is generated and not old:
                    # The producer owns a newly materialized asset's timestamp.
                    # Record it without changing a checksum-matching bundle.
                    entry['mtime_ns'] = (root / name).stat().st_mtime_ns
                    recorded_generated += 1
                    continue
                else:
                    entry['mtime_ns'] = stamp
                    if current is generated: refreshed_generated += 1
                os.utime(root / name, ns=(entry['mtime_ns'], entry['mtime_ns']))
        if changed:
            state.unlink(missing_ok=True)
        # Commit this per-file map only after the caller completes its build.
        pending.write_text(json.dumps({'version': 3, 'source_sha256': digest,
                                       'files': entries, 'generated_files': generated}) + '\n')
        print(json.dumps({'source_sha256': digest, 'source_changed': changed,
                          'restored_source_files': restored,
                          'refreshed_source_files': len(entries) - restored,
                          'restored_generated_files': restored_generated,
                          'refreshed_generated_files': refreshed_generated,
                          'recorded_generated_files': recorded_generated}))
    elif phase == 'commit':
        if not pending.exists():
            raise ValueError('Rust source changed during build or preparation is missing')
        prepared = json.loads(pending.read_text())
        if prepared.get('source_sha256') != digest:
            raise ValueError('Rust source changed during build or preparation is missing')
        for name, entry in prepared.get('generated_files', {}).items():
            if generated.get(name, {}).get('sha256') != entry['sha256']:
                raise ValueError('Generated Rust input changed during build: ' + name)
        # A cold build may create the checksum-verified archive after prepare.
        # Record the timestamp Cargo actually used, without refreshing it now.
        for name, entry in generated.items():
            entry['mtime_ns'] = (root / name).stat().st_mtime_ns
        prepared['generated_files'] = generated
        publication = detach_metadata(target)
        pending.write_text(json.dumps(prepared) + '\n')
        # A failed build never reaches commit, so partial targets stay untrusted.
        pending.replace(state)
        print(json.dumps(publication))
    else:
        raise ValueError('Expected prepare or commit')


if __name__ == '__main__':
    if sys.argv[1] == 'namespace':
        print(json.dumps(namespace(Path.cwd()), sort_keys=True))
    elif sys.argv[1] in ('pack', 'restore'):
        snapshot(sys.argv[1], Path(os.environ['CARGO_TARGET_DIR']), Path(sys.argv[2]))
    else:
        manage(sys.argv[1], Path.cwd(), Path(os.environ['CARGO_TARGET_DIR']))
