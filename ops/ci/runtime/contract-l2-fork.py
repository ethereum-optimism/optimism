#!/usr/bin/env python3
"""Run the OP Mainnet L2 fork tests as CircleCI's contracts-bedrock-tests-l2-fork job does."""
import argparse
import importlib.util
import os
from pathlib import Path
import sys

SPEC = importlib.util.spec_from_file_location('contract_upgrades', Path(__file__).with_name('contract-upgrades.py'))
UP = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(UP)
RPC = 'https://mainnet.optimism.io'
# Concurrent fork setup can burst requests at the public RPC before any assertion.
# Keep every test, but serialize the suites and bound forge's request rate.
TEST_ARGS = ['--threads', '1', '--compute-units-per-second', '100']


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
    os.environ.update(L2_FORK_RPC_URL=RPC, L2_FORK_BLOCK_NUMBER=block, JUNIT_TEST_PATH=str(directory / 'results.xml'))
    return (UP.stage(directory, 'nut-bundle-check', ['just', 'nut-bundle-check-no-build'])
            or UP.run_tests(directory, 'test-l2-fork-upgrade', TEST_ARGS))


if __name__ == '__main__': sys.exit(main())
