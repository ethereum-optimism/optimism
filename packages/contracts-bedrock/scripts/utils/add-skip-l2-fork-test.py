#!/usr/bin/env python3
"""
Script to add skipIfL2ForkTest to test files that fail when L2_FORK_RPC_URL is set.

This script automatically adds `skipIfL2ForkTest("not an L2 fork test");` after `super.setUp();`
in test files to prevent them from running during L2 fork tests.

Usage:
    python3 scripts/utils/add-skip-l2-fork-test.py [test_file1.t.sol] [test_file2.t.sol] ...

    If no files are specified, it will process all files listed in dev_test.log

Examples:
    # Add skip to specific files
    python3 scripts/utils/add-skip-l2-fork-test.py test/L1/ProxyAdminOwnedBase.t.sol

    # Process all failing tests from dev_test.log
    python3 scripts/utils/add-skip-l2-fork-test.py

    # Extract failing tests from a custom log file
    python3 scripts/utils/add-skip-l2-fork-test.py --log custom_test.log
"""

import os
import re
import sys
import argparse
from pathlib import Path
from typing import List, Set


def extract_failing_tests_from_log(log_file: str) -> Set[str]:
    """Extract unique failing test files from a log file."""
    failing_tests = set()

    try:
        with open(log_file, 'r') as f:
            for line in f:
                # Match lines like "Ran X tests for test/path/File.t.sol:TestContract"
                match = re.search(r'Ran \d+ tests? for (test/[^:]+\.t\.sol)', line)
                if match:
                    test_file = match.group(1)
                    failing_tests.add(test_file)
    except FileNotFoundError:
        print(f"Warning: Log file '{log_file}' not found")
        return set()

    return failing_tests


def find_setup_line(lines: List[str]) -> int:
    """Find the line number (0-indexed) containing super.setUp();"""
    for i, line in enumerate(lines):
        if 'function setUp()' in line:
            # Find the next super.setUp() line
            for j in range(i, min(i + 10, len(lines))):
                if 'super.setUp();' in lines[j]:
                    return j
    return -1


def add_skip_to_file(file_path: str, base_path: str = ".") -> bool:
    """
    Add skipIfL2ForkTest after super.setUp() in the given test file.

    Returns:
        True if the file was modified, False otherwise
    """
    full_path = os.path.join(base_path, file_path)

    if not os.path.exists(full_path):
        print(f"  NOT FOUND: {file_path}")
        return False

    with open(full_path, 'r') as f:
        lines = f.readlines()

    # Find super.setUp() line
    setup_line = find_setup_line(lines)

    if setup_line == -1:
        print(f"  SKIP: {file_path} - no setUp with super.setUp() found")
        return False

    # Check if skip is already present
    if setup_line + 1 < len(lines) and 'skipIfL2ForkTest' in lines[setup_line + 1]:
        print(f"  SKIP: {file_path} - already has skipIfL2ForkTest")
        return False

    # Get indentation from super.setUp() line
    indent = len(lines[setup_line]) - len(lines[setup_line].lstrip())
    skip_line = ' ' * indent + 'skipIfL2ForkTest("not an L2 fork test");\n'

    # Insert after super.setUp()
    lines.insert(setup_line + 1, skip_line)

    # Write back
    with open(full_path, 'w') as f:
        f.writelines(lines)

    print(f"  UPDATED: {file_path}")
    return True


def main():
    parser = argparse.ArgumentParser(
        description='Add skipIfL2ForkTest to test files',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__
    )
    parser.add_argument(
        'files',
        nargs='*',
        help='Test files to process (relative to contracts-bedrock directory)'
    )
    parser.add_argument(
        '--log',
        default='dev_test.log',
        help='Log file to extract failing tests from (default: dev_test.log)'
    )
    parser.add_argument(
        '--base-path',
        default='.',
        help='Base path for test files (default: current directory)'
    )

    args = parser.parse_args()

    # Determine which files to process
    if args.files:
        files_to_process = args.files
        print(f"Processing {len(files_to_process)} specified files...")
    else:
        # Extract from log file
        log_path = os.path.join(args.base_path, args.log)
        files_to_process = sorted(extract_failing_tests_from_log(log_path))
        if not files_to_process:
            print(f"No failing tests found in {args.log}")
            return 1
        print(f"Found {len(files_to_process)} unique failing test files in {args.log}")

    # Process each file
    modified_count = 0
    for file in files_to_process:
        if add_skip_to_file(file, args.base_path):
            modified_count += 1

    print(f"\nSummary: Modified {modified_count} / {len(files_to_process)} files")
    return 0


if __name__ == '__main__':
    sys.exit(main())
