#!/usr/bin/env python3
"""Hash, bind and restore Go runtime dependencies without accepting stale inputs."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import tarfile
import sys
import time

ROOT = Path(__file__).resolve().parents[3]
SETTINGS = {"go": {"hello_toolchain": "go1.24.13"}, "go-modules": {"verified": True},
            "contracts": {"profile": "ci", "build_args": ["--deny-warnings", "--skip", "test"]},
            "contracts-e2e": {"profile": "ci", "build_args": ["--skip", "test"]},
            "contracts-kontrol": {"profile": "ci", "build_args": ["--skip", "test"],
                                  "legacy_graph": {"source": "scripts/Artifacts.s.sol", "compiler": "0.8.28"}},
            "rust-e2e-release": {"profile": "release", "features": ["default"], "scope": "workspace"},
            "kona": {"profile": "release", "features": ["default"], "packages": ["kona-host", "kona-client", "kona-node", "op-zk-proposer"]},
            "op-reth": {"profile": "release", "features": ["default"], "packages": ["op-reth"]},
            "prestate": {"recipe": "just reproducible-prestate"},
            "sp1-executor": {"profile": "release", "features": ["all"], "packages": ["kona-sp1-super-range-executor"]}}

def dependency_source():
    spec = importlib.util.spec_from_file_location('go_dependency_source', ROOT / 'ops/ci/runtime/go-dependency-source.py')
    helper = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(helper)
    return helper


def digest(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()

def provenance(kind):
    return {"version": 1, "kind": kind, "commit_sha": os.environ['CI_COMMIT_SHA'],
            "settings": SETTINGS[kind], "mise_sha256": digest(ROOT / 'mise.toml')}

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
    metadata = provenance(kind)
    metadata.update(files=files, archive_sha256=digest(archive),
                    tool_versions={tool: subprocess.check_output([tool, 'version' if tool == 'go' else '--version'], text=True).strip()
                                   for tool in {'go': ['go'], 'go-modules': ['go'], 'contracts': ['go', 'forge'], 'contracts-e2e': ['go', 'forge'], 'contracts-kontrol': ['go', 'forge'],
                                                'rust-e2e-release': ['rustc', 'cargo'], 'kona': ['rustc', 'cargo'],
                                                'op-reth': ['rustc', 'cargo'], 'sp1-executor': ['rustc', 'cargo'], 'prestate': ['docker']}[kind]})
    if os.environ.get('CI_GO_REUSABLE_DEPENDENCY') == '1':
        metadata['build_inputs_sha256'] = dependency_source().fingerprint(kind)
    (output / 'metadata.json').write_text(json.dumps(metadata, indent=2) + '\n')

def restore(kind, artifact):
    started = time.monotonic()
    metadata = json.loads((artifact / 'metadata.json').read_text())
    expected = provenance(kind)
    if 'build_inputs_sha256' in metadata:
        actual = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
        if actual != os.environ['CI_COMMIT_SHA']:
            raise ValueError(f"Consumer checkout {actual} differs from requested {os.environ['CI_COMMIT_SHA']}")
        dependency_source().verify(kind, metadata)
        expected.pop('commit_sha')
    if any(metadata.get(key) != value for key, value in expected.items()):
        raise ValueError('Dependency revision, toolchain pins or settings mismatch')
    archive = artifact / 'files.tar.gz'
    if metadata.get('archive_sha256') != digest(archive):
        raise ValueError('Dependency archive changed')
    files = metadata.get('files')
    if not isinstance(files, dict) or not files:
        raise ValueError('Missing dependency file manifest')
    for name, expected in files.items():
        if (not isinstance(name, str) or Path(name).is_absolute() or '..' in Path(name).parts
                or not isinstance(expected, str) or not re.fullmatch('[0-9a-f]{64}', expected)):
            raise ValueError('Invalid dependency file manifest')
    with tarfile.open(archive) as stream:
        def restore_member(member, destination):
            member = tarfile.data_filter(member, destination)
            if member is None: return None
            path = ROOT / member.name
            expected = files.get(member.name)
            if expected and member.isfile() and path.is_file() and not path.is_symlink() and digest(path) == expected and path.stat().st_mode & 0o111 == member.mode & 0o111:
                # Artifact dependencies can arrive with producer files already
                # materialized. Go module directories are read-only. Reuse
                # exact bytes, but still validate the archived file itself.
                with stream.extractfile(member) as source:
                    if hashlib.file_digest(source, 'sha256').hexdigest() != expected:
                        raise ValueError('Corrupt archived dependency file')
                return None
            return member
        stream.extractall(ROOT, filter=restore_member)
    for name, expected in files.items():
        path = ROOT / name
        if Path(name).is_absolute() or '..' in Path(name).parts or not path.is_file() or digest(path) != expected:
            raise ValueError('Missing or corrupt dependency file')
    output = ROOT / 'tmp/testlogs/dependencies'
    output.mkdir(parents=True, exist_ok=True)
    metadata['consumer_commit_sha'] = os.environ['CI_COMMIT_SHA']
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
