#!/bin/bash
#
#
# This script runs the benchmarks and prepares a customized report.
#
# Primary use-case is running in (GitHub) CI, but can also run locally.
#
# Benchmarks can be run in two modes:
# 1. Basic mode: Run the benchmarks for the current commit only (used if HEAD_SHA == BASE_SHA)
# 2. Comparison mode: Run the benchmarks for the current commit and a target commit, and compare them (to see the specific improvements)
#
# Important variables are sourced from ./env-vars.sh, which combines local git information with Github specific variables and methods.
#
# Useful environment variables (to possibly override):
# - X_HEAD_SHA: custom head sha (for dev purposes)
# - X_BASE_SHA: custom base sha (which is run first, and the head benchmark compared against)
# - SKIP_BENCH: skip running `cargo bench` (just generates the HTML based on the previous benchmark).
#
set -e

script_dir=$( dirname -- "$0"; )

echo "Running benchmarks ..."

# Gather env vars
# https://docs.github.com/en/actions/learn-github-actions/variables
source "${script_dir}/env-vars.sh"

echo "DATE_UTC:         ${DATE}"
echo "HEAD_SHA:         ${HEAD_SHA}"
echo "HEAD_BRANCH:      ${HEAD_BRANCH}"
echo "BASE_SHA:         ${BASE_SHA}"
echo "BASE_REF:         ${BASE_REF}"
echo "GITHUB_SHA:       ${GITHUB_SHA}"
echo "GITHUB_REF_NAME:  ${GITHUB_REF_NAME}"
echo "GITHUB_REF_TYPE:  ${GITHUB_REF_TYPE}"
echo "GITHUB_HEAD_REF:  ${GITHUB_HEAD_REF}"
echo "GITHUB_ACTOR:     ${GITHUB_ACTOR}"
echo "PR_NUMBER:        ${PR_NUMBER}"
echo "RUNNER_ARCH:      ${RUNNER_ARCH}"
echo "RUNNER_OS:        ${RUNNER_OS}"

#
# RUN THE BENCHMARKS
#
cd $script_dir
cd ../..

function run_benchmark() {
    if [ "$HEAD_SHA" == "$BASE_SHA" ]; then
        # Benchmark only current commit, no comparison
        echo "Running cargo bench ..."
        cargo bench --workspace --features optimism
    else
        # Benchmark target commit first, and then benchmark current commit against that baseline
        echo "Benchmarking ${HEAD_SHA_SHORT} against the target ${BASE_SHA_SHORT} ..."

        # Switch to target commit and run benchmarks
        echo "Switching to $BASE_SHA_SHORT and starting benchmarks ..."
        git checkout $BASE_SHA
        cargo bench --workspace --features optimism

        # Switch back to current commit and run benchmarks again
        echo "Switching back to $HEAD_SHA_SHORT and running benchmarks ..."
        # Reset to ensure any changes (e.g. to Cargo.lock) are discarded before attempting to checkout.
        git reset --hard
        git checkout $HEAD_SHA
        cargo bench --workspace --features optimism
    fi
}

# Run benchmarks now (if env var SKIP_BENCH is not set)
if [ -n "$SKIP_BENCH" ]; then
    echo "SKIP_BENCH is set, skipping cargo bench."
else
    run_benchmark
fi

#
# Build the summary
#
# Grab the changes as markdown (used in templates for PR and job-summary comment)
export BENCH_CHANGES_MD=$( ./scripts/ci/criterion-get-changes.py target/criterion --output-format md )
export BENCH_CHANGES_MD_ONLYSIGNIFICANT=$( ./scripts/ci/criterion-get-changes.py target/criterion --output-format md --only-significant )
if [ -z "$BENCH_CHANGES_MD_ONLYSIGNIFICANT" ]; then
    export BENCH_CHANGES_MD_ONLYSIGNIFICANT="None"
fi

# Prettify criterion report
mkdir -p target/benchmark-in-ci
./scripts/ci/criterion-prettify-report.sh target/criterion target/benchmark-in-ci/benchmark-report
echo "Saved report: target/benchmark-in-ci/benchmark-report/report/index.html"

# Create summary markdown
fn="target/benchmark-in-ci/benchmark-report/benchmark-summary.md"
envsubst < scripts/ci/templates/benchmark-summary.md > $fn
echo "Wrote summary: $fn"

# Create summary pr comment
fn="target/benchmark-in-ci/benchmark-report/benchmark-pr-comment.md"
envsubst < scripts/ci/templates/benchmark-pr-comment.md > $fn
echo "Wrote PR comment: $fn"
