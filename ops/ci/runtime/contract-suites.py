#!/usr/bin/env python3
"""Run a contract test suite as CircleCI's contracts-bedrock-tests job does.

The `standard` suite is contracts-bedrock-tests. The `modified` suite is
contracts-bedrock-tests-heavy-fuzz-modified.
"""
import argparse
import importlib.util
import os
from pathlib import Path
import sys

SPEC = importlib.util.spec_from_file_location('contract_upgrades', Path(__file__).with_name('contract-upgrades.py'))
UP = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(UP)
# Each job's CircleCI `test_list` parameter. It prints the test files to run, one per line.
# The `modified-baseline` task fetches origin/develop once, so every task of a run diffs the same commit.
FILE_COMMANDS = {
    'standard': ['find', 'test', '-name', '*.t.sol'],
    'modified': ['bash', '-eo', 'pipefail', '-c',
                 "git diff origin/develop...HEAD --name-only --no-renames --diff-filter=AM -- './test/**/*.t.sol' | sed 's|packages/contracts-bedrock/||'"],
}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('prepare', 'run')); parser.add_argument('--suite', choices=FILE_COMMANDS, required=True)
    parser.add_argument('--feature', choices=UP.FEATURES, required=True)
    args = parser.parse_args()
    UP.configure(args.feature, FOUNDRY_PROFILE='ciheavy' if args.suite == 'modified' else UP.profile())
    directory = UP.report('contract-suites', args.suite + '-' + args.feature, args.mode)
    status = UP.stage(directory, 'files', FILE_COMMANDS[args.suite])
    if status: return status
    files = (directory / 'files.log').read_text().strip()
    if not files:
        # Like the CircleCI job, stop with success. With no TEST_FILES, `just test` runs every test.
        print('No test files to run.', flush=True); return 0
    if args.mode == 'prepare': return UP.build(directory)
    os.environ.update(TEST_FILES=files, JUNIT_TEST_DIR=str(directory))
    return UP.run_tests(directory, 'test') or UP.stage(directory, 'lint-test-names', ['just', 'lint-forge-tests-check-no-build'])


if __name__ == '__main__': sys.exit(main())
