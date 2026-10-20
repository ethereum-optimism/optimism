#!/usr/bin/env python3
"""Bind reusable Go helpers to all tracked build inputs, independently of PR SHA."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[3]
COMMON = ['mise.toml', '.gitmodules', 'superchain-registry', 'justfile', 'justfiles', 'ops/ci/runtime', '.rwx',
          'go.mod', 'go.sum', 'package.json', 'pnpm-lock.yaml', 'pnpm-workspace.yaml', '.npmrc']
RUST = ['rust', 'op-core/nuts/bundles', 'op-core/superchain']
SCOPES = {
    'go': COMMON + [':(glob)**/*.go', 'cannon', 'op-core/superchain'],
    'contracts': COMMON + ['packages/contracts-bedrock', 'op-deployer/justfile',
                            'op-deployer/pkg/deployer/artifacts', 'op-service/ioutil'],
    'kona': COMMON + RUST,
    'op-reth': COMMON + RUST,
    'prestate': COMMON + RUST + ['cannon', 'op-service', 'op-preimage', 'ops/scripts',
                                 'Dockerfile*', '.dockerignore'],
    'compiler': COMMON + ['packages/contracts-bedrock', 'op-deployer/justfile',
                           'op-deployer/pkg/deployer/artifacts', 'op-service/ioutil',
                           ':(glob)**/go.mod', ':(glob)**/go.sum'],
}

def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT)

def fingerprint(kind, revision='HEAD'):
    # Tree diff includes names, modes, blob IDs, and submodule commits. A changed
    # helper/configuration must invalidate reuse just like changed source does.
    empty = subprocess.check_output(['git', 'hash-object', '-t', 'tree', '--stdin'], input=b'', cwd=ROOT).decode().strip()
    return hashlib.sha256(git('diff-tree', '--no-commit-id', '--raw', '--no-abbrev', '-r', '-z', empty, revision, '--', *SCOPES[kind])).hexdigest()

def ancestor(kind):
    sha = git('log', '-1', '--format=%H', 'HEAD', '--', *SCOPES[kind]).decode().strip()
    if not sha or fingerprint(kind, sha) != fingerprint(kind):
        sha = git('rev-parse', 'HEAD').decode().strip()
    return sha

def verify(kind, metadata):
    sha = metadata.get('commit_sha', '')
    if len(sha) != 40 or any(c not in '0123456789abcdef' for c in sha):
        raise ValueError('Invalid dependency source revision')
    subprocess.run(['git', 'merge-base', '--is-ancestor', sha, 'HEAD'], cwd=ROOT, check=True)
    if git('status', '--porcelain', '--untracked-files=no', '--', *SCOPES[kind]).strip():
        raise ValueError('Modified dependency build inputs')
    expected = fingerprint(kind)
    if metadata.get('build_inputs_sha256') != expected or fingerprint(kind, sha) != expected:
        raise ValueError('Dependency build inputs differ from current source')

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['plan', 'fingerprint'])
    parser.add_argument('kind', choices=SCOPES, nargs='?')
    args = parser.parse_args()
    if args.command == 'plan':
        values = Path(os.environ['RWX_VALUES'])
        for kind in SCOPES:
            (values / kind).write_text(ancestor(kind))
    else:
        if not args.kind: parser.error('fingerprint requires a kind')
        print(fingerprint(args.kind))
