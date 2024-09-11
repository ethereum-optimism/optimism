#!/usr/bin/env bash
set -euo pipefail

# This script is used to run extra fuzz test iterations on any fuzz tests that
# have been added or modified in a PR. We typically want to run extra fuzz
# iterations when new tests are added to make sure that they are not flaky with
# some small percentage of fuzz runs.
# NOTE: This script is NOT perfect and can only catch changes to fuzz tests
# that are made within the test file itself. It won't catch changes to
# dependencies or other external factors that might impact the behavior of the
# fuzz test. This script may also run fuzz tests that have not actually been
# modified.

# Set the number of fuzz runs to run.
# 350000 fuzz runs will guarantee that any test that fails 1% of the time with
# the default 512 fuzz runs will fail 99.9% of the time (on average) inside of
# this script.
FUZZ_RUNS=${1:-350000}

# Set the number of invariant runs to run.
# Invariant runs are generally slower than fuzz runs so we can't afford to run
# as many of them. 25000 is probably good enough for most cases.
INVARIANT_RUNS=${2:-25000}

# Verify that FUZZ_RUNS is a number.
if ! [[ "$FUZZ_RUNS" =~ ^[0-9]+$ ]]; then
    echo "Fuzz runs must be a number"
    exit 1
fi

# Trap any errors and exit.
trap 'echo "Script failed at line $LINENO"' ERR

# Get the various base directories.
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
CONTRACTS_BASE=$(dirname "$(dirname "$SCRIPT_DIR")")
ROOT_DIR=$(dirname "$(dirname "$CONTRACTS_BASE")")

# Change to the root directory.
cd "$ROOT_DIR"

# Get a list of changed Solidity test files relative to the project root.
CHANGED_FILES=$(git diff origin/develop...HEAD --name-only -- '*.sol')

# Exit if no changed Solidity files are found.
if [ -z "$CHANGED_FILES" ]; then
    echo "No changed Solidity files found"
    exit 0
fi

# Initialize an array to hold relevant function names.
NEW_OR_MODIFIED_TEST_NAMES=""

# Process each changed file.
for FILE in $CHANGED_FILES; do
    # Get the diff for the file.
    DIFF=$(git diff origin/develop...HEAD --unified=0 -- "$FILE")

    # Figure out every modified line.
    MODIFIED_LINES=$(echo "$DIFF" | \
    awk '/^@@/ {
        split($3, a, ",")
        start = substr(a[1], 2)
        if (length(a) > 1)
            count = a[2]
        else
            count = 1
        for (i = 0; i < count; i++)
            print start + i
    }' | sort -n | uniq | xargs)

    # Extract function names and their line numbers from the entire file
    FUNCTION_LINES=$(awk '/function testFuzz_|function invariant_/ {print FNR, $0}' "$FILE")

    # Reverse the function lines so we can match the last function modified.
    # We'd otherwise end up matching the first function with a line number less
    # than the modified line number which is not what we want.
    FUNCTION_LINES=$(echo "$FUNCTION_LINES" | sort -r)

    # Process each modified line.
    for MODIFIED_LINE_NUM in $MODIFIED_LINES; do
        # Check all functions to find the last one where the line number of the
        # function is less than or equal to the modified line number. This is
        # the function that was most likely modified.
        # NOTE: This is not perfect and may accidentally match a function that
        # was not actually modified but it works well enough and at least won't
        # accidentally miss any modified fuzz tests.
        while IFS= read -r func; do
            # Get the function line number and name.
            FUNC_LINE_NUM=$(echo "$func" | awk '{print $1}')
            FUNC_NAME=$(echo "$func" | awk '{print $3}' | sed 's/(.*//')

            # Check if the modified line number is greater than or equal to the
            # function line number. If it is, then we've found the closest fuzz
            # test that was modified. Again, this is not perfect and may lead
            # to false positives but won't lead to false negatives.
            if [ "$MODIFIED_LINE_NUM" -ge "$FUNC_LINE_NUM" ]; then
                NEW_OR_MODIFIED_TEST_NAMES+="$FUNC_NAME "
                break
            fi
        done <<< "$FUNCTION_LINES"
    done
done

# Remove duplicates and sort.
NEW_OR_MODIFIED_TEST_NAMES=$(echo "$NEW_OR_MODIFIED_TEST_NAMES" | xargs -n1 | sort -u | xargs)

# Exit if no new or modified fuzz tests are found.
if [ -z "$NEW_OR_MODIFIED_TEST_NAMES" ]; then
    echo "No new or modified fuzz tests found"
    exit 0
fi

# Print the detected tests on different lines.
echo "Detected new or modified fuzz tests:"
for TEST_NAME in $NEW_OR_MODIFIED_TEST_NAMES; do
    echo "  $TEST_NAME"
done

# Change to the contracts base directory.
cd "$CONTRACTS_BASE"

# Set the number of invariant runs.
export FOUNDRY_INVARIANT_RUNS="$INVARIANT_RUNS"

# Run the detected tests with extra fuzz runs
forge test --match-test "${NEW_OR_MODIFIED_TEST_NAMES// /|}" --fuzz-runs "$FUZZ_RUNS"
