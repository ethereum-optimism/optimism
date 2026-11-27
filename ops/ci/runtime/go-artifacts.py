#!/usr/bin/env python3
"""Pack the outputs of a dependency build, and restore them in a task that consumes them.

pack <kind> <path>...     Write files.tar.gz and metadata.json to .ci/go-tests/dependencies/<kind>.
restore <kind> <directory>  Extract the artifact in <directory> if it fits this checkout.
"""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tarfile
import sys

ROOT = Path(__file__).resolve().parents[3]
KINDS = ('go', 'go-modules', 'contracts', 'contracts-e2e', 'rust-e2e-release', 'kona', 'op-reth', 'prestate', 'sp1-executor')

def dependency_source():
    spec = importlib.util.spec_from_file_location('go_dependency_source', ROOT / 'ops/ci/runtime/go-dependency-source.py')
    helper = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(helper)
    return helper


def digest(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()

def pack(kind, paths):
    actual_sha = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    if actual_sha != os.environ['CI_COMMIT_SHA']:
        raise ValueError(f"Dependency revision {actual_sha} differs from requested {os.environ['CI_COMMIT_SHA']}")
    output = ROOT / '.ci/go-tests/dependencies' / kind
    output.mkdir(parents=True, exist_ok=True)
    files = {}
    for name in paths:
        path = ROOT / name
        if not path.exists():
            raise ValueError(f'Missing dependency output: {name}')
        for item in sorted(path.rglob('*')) if path.is_dir() else [path]:
            if item.is_file():
                files[str(item.relative_to(ROOT))] = digest(item)
    if not files:
        raise ValueError('Empty dependency artifact')
    archive = output / 'files.tar.gz'
    with tarfile.open(archive, 'w:gz') as stream:
        for name in paths:
            stream.add(ROOT / name, arcname=name)
    metadata = {'kind': kind, 'commit_sha': actual_sha, 'files': files, 'archive_sha256': digest(archive)}
    if os.environ.get('CI_GO_REUSABLE_DEPENDENCY') == '1':
        source = dependency_source()
        metadata['build_inputs_sha256'] = source.fingerprint(kind)
        source.verify(kind, metadata)
    (output / 'metadata.json').write_text(json.dumps(metadata, indent=2) + '\n')

def restore(kind, artifact):
    metadata = json.loads((artifact / 'metadata.json').read_text())
    expected = {'kind': kind, 'commit_sha': os.environ['CI_COMMIT_SHA']}
    if 'build_inputs_sha256' in metadata:
        actual = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
        if actual != os.environ['CI_COMMIT_SHA']:
            raise ValueError(f"Consumer checkout {actual} differs from requested {os.environ['CI_COMMIT_SHA']}")
        # Same-commit consumers may intentionally project the workspace (e.g.
        # acceptance tests). Their artifact still has the strict source binding.
        # Relax that binding only after verifying an ancestor's complete inputs.
        if metadata.get('commit_sha') != expected['commit_sha']:
            dependency_source().verify(kind, metadata)
            expected.pop('commit_sha')
    if any(metadata.get(key) != value for key, value in expected.items()):
        raise ValueError('Dependency kind or revision mismatch')
    archive = artifact / 'files.tar.gz'
    if metadata.get('archive_sha256') != digest(archive):
        raise ValueError('Dependency archive changed')
    files = metadata['files']
    with tarfile.open(archive) as stream:
        def member_to_extract(member, destination):
            member = tarfile.data_filter(member, destination)
            if member is None: return None
            path = ROOT / member.name
            # The producer's layer can already hold these files, and Go module
            # directories are read-only. Do not write a file that is identical.
            if (member.isfile() and path.is_file() and not path.is_symlink()
                    and digest(path) == files.get(member.name)
                    and path.stat().st_mode & 0o111 == member.mode & 0o111):
                return None
            return member
        stream.extractall(ROOT, filter=member_to_extract)

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('command', choices=['pack', 'restore'])
    parser.add_argument('kind', choices=KINDS)
    parser.add_argument('paths', nargs='+')
    args = parser.parse_args()
    try:
        if args.command == 'pack': pack(args.kind, args.paths)
        elif len(args.paths) == 1: restore(args.kind, Path(args.paths[0]))
        else: raise ValueError('Restore requires exactly one artifact directory')
    except (ValueError, OSError, KeyError, subprocess.CalledProcessError) as error:
        print(f'ERROR: {error}', file=sys.stderr)
        sys.exit(1)
