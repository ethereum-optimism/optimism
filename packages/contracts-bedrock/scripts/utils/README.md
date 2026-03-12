# Test Utility Scripts

This directory contains utility scripts for managing and fixing test files.

## add-skip-l2-fork-test.py

Automatically adds `skipIfL2ForkTest("not an L2 fork test");` to test files that should not run during L2 fork tests.

### Background

When `L2_FORK_RPC_URL` is set, the test framework activates L2 fork mode. Tests that are not designed for L2 fork testing will fail because:
- They expect L1 contracts to be available
- They try to deploy contracts that already exist on the fork
- They make assumptions about the chain state that don't hold on a forked L2

### Usage

```bash
# From the contracts-bedrock directory

# Process all failing tests from dev_test.log
python3 scripts/utils/add-skip-l2-fork-test.py

# Add skip to specific files
python3 scripts/utils/add-skip-l2-fork-test.py test/L1/ProxyAdminOwnedBase.t.sol test/L2/BaseFeeVault.t.sol

# Use a custom log file
python3 scripts/utils/add-skip-l2-fork-test.py --log my_test_output.log
```

### How it works

The script:
1. Finds test files that are failing (from log file or command line)
2. Locates the `function setUp()` with `super.setUp();`
3. Inserts `skipIfL2ForkTest("not an L2 fork test");` after `super.setUp();`
4. Skips files that already have the skip call

### When to use

Use this script when:
- You add new L1-specific tests that shouldn't run on L2 forks
- You encounter test failures when `L2_FORK_RPC_URL` is set
- You need to bulk-update multiple test files

### When NOT to use

DO NOT use this script for:
- L2 fork upgrade tests (they SHOULD run with `L2_FORK_RPC_URL`)
- Tests that are already designed to work with L2 forks
- Tests that you want to make fork-compatible instead of skipping

### Examples

#### Example 1: Fix failing tests after setting L2_FORK_RPC_URL

```bash
# Run tests and save output
forge test > dev_test.log 2>&1

# Fix all failing tests
python3 scripts/utils/add-skip-l2-fork-test.py

# Verify the fixes
forge test
```

#### Example 2: Add skip to a specific new test

```bash
python3 scripts/utils/add-skip-l2-fork-test.py test/L1/MyNewTest.t.sol
```

### Output example

```
Processing 12 files from dev_test.log...
  UPDATED: test/L1/ProxyAdminOwnedBase.t.sol
  SKIP: test/L1/OPContractsManager.t.sol - already has skipIfL2ForkTest
  UPDATED: test/L2/BaseFeeVault.t.sol
  NOT FOUND: test/L2/NonexistentTest.t.sol
  SKIP: test/L2/L2ForkUpgrade.t.sol - no setUp with super.setUp() found

Summary: Modified 2 / 5 files
```

## Future scripts

Additional utility scripts can be added here for:
- Automatically running gas benchmarks
- Generating test coverage reports
- Validating test naming conventions
- etc.
