# Kurtosis DevNet PR Checks

This document explains the PR check mechanisms implemented for the Kurtosis DevNet to ensure changes don't inadvertently break the deployment process.

## Overview

There are two types of checks:

1. **PR Sanity Check** - A lightweight check that runs on all PRs to verify that the kurtosis devnet can be deployed successfully.
2. **Merge Queue Check** - A more efficient check that runs only in the merge queue, using a persistent devnet that's updated rather than recreated.

## PR Sanity Check

The PR sanity check is a relatively lightweight workflow that:

1. Builds the minimal set of Docker images required for a simple devnet
2. Deploys a simple devnet configuration
3. Verifies that the devnet is running properly

This check runs on all PRs and serves as a quick validation that the PR doesn't break the basic functionality of the devnet.

## Merge Queue Check

The merge queue check is more efficient and is designed to run in the GitHub merge queue. It:

1. Checks for an existing persistent devnet
2. If one exists, it updates it with the new code
3. If one doesn't exist, it creates a new one
4. Verifies that the devnet is running properly

The merge queue workflow enforces the fact that no concurrency would disrupt the in-place update assumptions, making it an ideal place for this more efficient check.

## Benefits

These checks help to:

1. Catch inadvertent breakage of kurtosis deployments early in the development process
2. Provide clear feedback to developers about the impact of their changes
3. Minimize the risk of issues being merged into the main branches

## Troubleshooting

If a PR check fails, you should:

1. Check the CircleCI logs for detailed information
2. Look for specific error messages related to the kurtosis deployment
3. Test the deployment locally using the same configuration
4. Reach out to the DevNet team for assistance if needed
