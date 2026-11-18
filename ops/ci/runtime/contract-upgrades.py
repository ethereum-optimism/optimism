#!/usr/bin/env python3
"""Run the L1 contract upgrade tests as CircleCI's contracts-bedrock-tests-upgrade job does.

The contract suite and L2 fork runners import the shared steps of this module.
"""
import argparse
import html
import importlib.util
import os
from pathlib import Path
import shutil
import sys
import urllib.parse

SPEC = importlib.util.spec_from_file_location('ci_report', Path(__file__).with_name('ci-report.py'))
REPORT = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(REPORT)
ROOT = Path(__file__).resolve().parents[3]
CONTRACTS = ROOT / 'packages/contracts-bedrock'
# CircleCI's features_matrix, with the variable that its setup-features command exports.
FEATURES = {'main': None,
            'CUSTOM_GAS_TOKEN': 'SYS_FEATURE__CUSTOM_GAS_TOKEN',
            'INTEROP': 'SYS_FEATURE__INTEROP',
            'BRIDGE_HOOK': 'DEV_FEATURE__BRIDGE_HOOK',
            'OP_PRIVATE': 'DEV_FEATURE__OP_PRIVATE'}
# Variant to (FORK_OP_CHAIN, feature): CircleCI's feature matrix on OP Mainnet, then its chain matrix.
VARIANTS = {**{'feature-' + feature: ('op', feature) for feature in FEATURES},
            **{'chain-' + chain: (chain, 'main') for chain in ('op', 'ink', 'unichain')}}


class Redactor:
    """Mask authentication data before streaming or saving RPC-dependent output."""
    def __init__(self, url):
        parsed = urllib.parse.urlsplit(url)
        if parsed.scheme not in ('http', 'https') or not parsed.hostname: raise ValueError('Invalid test-only RPC configuration')
        parts = [url, html.escape(url, quote=True), urllib.parse.quote(url, safe="")]
        parts += [v for _, v in urllib.parse.parse_qsl(parsed.query) if v]
        parts += [v for v in (parsed.username, parsed.password) if v]
        path_parts = [v for v in parsed.path.split('/') if v]
        # Hosted RPC keys commonly occupy the final path segment. Error pages
        # can echo that key alone, including keys shorter than sixteen bytes.
        parts += [v for v in path_parts if len(v) >= 16] + path_parts[-1:]
        parts += [html.escape(v, quote=True) for v in parts] + [urllib.parse.quote(v, safe="") for v in parts]
        self.values = sorted({v.encode() for v in parts}, key=len, reverse=True)
    def __call__(self, data, replacement=b'<test-only-rpc>'):
        for value in self.values: data = data.replace(value, replacement)
        return data
    def xml(self, data):
        # XML text and attributes must remain parseable after authentication is
        # removed. The plain-log marker would introduce an unclosed XML tag.
        return self(data, b'&lt;test-only-rpc&gt;')


def profile():
    """Return the Foundry profile that CircleCI uses on this branch."""
    return 'ci' if os.environ['CI_BRANCH'] == 'develop' else 'liteci'


def configure(feature, **environment):
    """Export the CircleCI job's `environment:` values and the flag of its feature."""
    os.environ.update(environment, CI='true')
    if FEATURES[feature]: os.environ[FEATURES[feature]] = 'true'


def report(*names):
    """Return a new, empty directory under .ci for the logs and reports of this task's stages."""
    directory = ROOT.joinpath('.ci', *names)
    shutil.rmtree(directory, ignore_errors=True); directory.mkdir(parents=True)
    return directory


def stage(directory, name, argv, redactor=lambda data: data):
    return REPORT.stage(directory, name, argv, cwd=CONTRACTS, layout='split', redactor=redactor)


def build(directory):
    """Compile the Go FFI helper and the contracts with the exported profile."""
    return stage(directory, 'go-ffi', ['just', 'build-go-ffi']) or stage(directory, 'contracts-build', ['forge', 'build'])


def run_tests(directory, recipe, arguments=(), redactor=lambda data: data):
    """Run a test recipe. After a test failure, run its rerun recipe, which prints the traces."""
    status = stage(directory, 'tests', ['just', recipe, *arguments], redactor)
    if status and status < 128:
        # CircleCI's trace step sets no JUnit path. With one set, the recipe sends
        # forge's output, traces included, to the report file.
        for name in ('JUNIT_TEST_PATH', 'JUNIT_TEST_DIR'): os.environ.pop(name, None)
        stage(directory, 'rerun', ['just', recipe + '-rerun', *arguments], redactor)
    return status


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('prepare', 'run')); parser.add_argument('--variant', choices=VARIANTS, required=True)
    args = parser.parse_args()
    chain, feature = VARIANTS[args.variant]
    configure(feature, FOUNDRY_PROFILE=profile(), FOUNDRY_FUZZ_SEED='42424242', FOUNDRY_FUZZ_RUNS='1',
              FORK_OP_CHAIN=chain, FORK_BASE_CHAIN='mainnet')
    directory = report('contract-upgrades', args.variant, args.mode)
    if args.mode == 'prepare': return build(directory)
    # Like CircleCI, the recipe gets the archive URL itself: it pins the block and retries fork reads.
    url = os.environ.get('OP_CI_MAINNET_L1_ARCHIVE_RPC_URL', '')
    try: redactor = Redactor(url)
    except ValueError: sys.exit('OP_CI_MAINNET_L1_ARCHIVE_RPC_URL is not an http(s) URL. Does this branch have a grant on the RPC vault?')
    junit = directory / 'results.xml'
    os.environ.update(ETH_RPC_URL=url, JUNIT_TEST_PATH=str(junit))
    try: return run_tests(directory, 'test-upgrade', redactor=redactor)
    finally:
        # Failure reasons in the report can quote the URL.
        if junit.exists(): junit.write_bytes(redactor.xml(junit.read_bytes()))


if __name__ == '__main__': sys.exit(main())
