#!/usr/bin/env python3
"""Run the OP Mainnet L2 fork tests as CircleCI's contracts-bedrock-tests-l2-fork job does."""
import argparse
import importlib.util
import os
from pathlib import Path
import sys


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


UP = helper('contract-upgrades')
PACER = helper('rpc-pacer')
RPC = 'https://mainnet.optimism.io'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('prepare', 'run')); parser.add_argument('--fork-block', default='latest')
    args = parser.parse_args()
    UP.configure('main', FOUNDRY_PROFILE='ci')
    directory = UP.report('contract-l2-fork', args.mode)
    if args.mode == 'prepare': return UP.build(directory)
    # Like the CircleCI job, `latest` forks the current head.
    block = UP.REPORT.command(['cast', 'block-number', '--rpc-url', RPC]) if args.fork_block == 'latest' else args.fork_block
    print('Forking OP Mainnet at block ' + block, flush=True)
    os.environ.update(L2_FORK_BLOCK_NUMBER=block, JUNIT_TEST_PATH=str(directory / 'results.xml'))
    status = UP.stage(directory, 'nut-bundle-check', ['just', 'nut-bundle-check-no-build'])
    if status: return status
    # CircleCI forks a private RPC. The public one needs paced requests: see rpc-pacer.py.
    with PACER.serve(RPC) as endpoint:
        os.environ['L2_FORK_RPC_URL'] = endpoint
        return UP.run_tests(directory, 'test-l2-fork-upgrade')


if __name__ == '__main__': sys.exit(main())
