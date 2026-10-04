#!/usr/bin/env python3
"""Hash, bind and restore Go runtime dependencies without accepting stale inputs."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tarfile
import sys
import time

ROOT = Path(__file__).resolve().parents[2]
SETTINGS = {"go": {"hello_toolchain": "go1.24.13"}, "contracts": {"profile": "ci"},
            "contracts-e2e": {"profile": "ci", "build_args": ["--skip", "test"]},
            "rust-e2e-release": {"profile": "release", "features": ["default"], "scope": "workspace"},
            "kona": {"profile": "release", "features": ["default"], "packages": ["kona-host", "kona-client", "kona-node", "op-zk-proposer"]},
            "op-reth": {"profile": "release", "features": ["default"], "packages": ["op-reth"]},
            "prestate": {"recipe": "just reproducible-prestate"},
            "sp1-executor": {"profile": "release", "features": ["all"], "packages": ["kona-sp1-super-range-executor"]}}

def digest(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()

def provenance(kind):
    return {"version": 1, "kind": kind, "commit_sha": os.environ['CI_COMMIT_SHA'],
            "settings": SETTINGS[kind], "mise_sha256": digest(ROOT / 'mise.toml')}

def pack(kind, paths):
    if subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip() != os.environ['CI_COMMIT_SHA']:
        raise ValueError('Dependency revision differs from the requested source')
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
    metadata = provenance(kind)
    metadata.update(files=files, archive_sha256=digest(archive),
                    tool_versions={tool: subprocess.check_output([tool, 'version' if tool == 'go' else '--version'], text=True).strip()
                                   for tool in {'go': ['go'], 'contracts': ['go', 'forge'], 'contracts-e2e': ['go', 'forge'],
                                                'rust-e2e-release': ['rustc', 'cargo'], 'kona': ['rustc', 'cargo'],
                                                'op-reth': ['rustc', 'cargo'], 'sp1-executor': ['rustc', 'cargo'], 'prestate': ['docker']}[kind]})
    (output / 'metadata.json').write_text(json.dumps(metadata, indent=2) + '\n')

def restore(kind, artifact):
    started = time.monotonic()
    metadata = json.loads((artifact / 'metadata.json').read_text())
    if any(metadata.get(key) != value for key, value in provenance(kind).items()):
        raise ValueError('Dependency revision, toolchain pins or settings mismatch')
    archive = artifact / 'files.tar.gz'
    if metadata.get('archive_sha256') != digest(archive):
        raise ValueError('Dependency archive changed')
    files = metadata.get('files')
    if not isinstance(files, dict) or not files:
        raise ValueError('Missing dependency file manifest')
    with tarfile.open(archive) as stream:
        stream.extractall(ROOT, filter='data')
    for name, expected in files.items():
        path = ROOT / name
        if Path(name).is_absolute() or '..' in Path(name).parts or not path.is_file() or digest(path) != expected:
            raise ValueError('Missing or corrupt dependency file')
    output = ROOT / 'tmp/testlogs/dependencies'
    output.mkdir(parents=True, exist_ok=True)
    metadata['restore_seconds'] = time.monotonic() - started
    (output / f'{kind}.json').write_text(json.dumps(metadata, indent=2) + '\n')

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['pack', 'restore'])
    parser.add_argument('kind', choices=SETTINGS)
    parser.add_argument('paths', nargs='+')
    args = parser.parse_args()
    try:
        if args.command == 'pack': pack(args.kind, args.paths)
        elif len(args.paths) == 1: restore(args.kind, Path(args.paths[0]))
        else: raise ValueError('Restore requires exactly one artifact directory')
    except (ValueError, OSError, KeyError, subprocess.CalledProcessError) as error:
        print(f'ERROR: {error}', file=sys.stderr)
        sys.exit(1)
