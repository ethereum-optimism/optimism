# Optimism Monorepo (VERY WIP)

## Quickstart

```
git clone git@github.com:ethereum-optimism/optimism.git
cd optimism
yarn
yarn lint
yarn test
```

## Taming the Monorepo

1. You solely use yarn workspaces for the Mono-Repo workflow.
1. You use lerna’s utility commands to optimize managing of multiple packages, e.g., selective execution of npm scripts for testing.
1. You use lerna for publishing packages since lerna provides sophisticated features with its version and publish commands.

## Incremental Tests

```
BRANCH_POINT="$(git merge-base $(git rev-parse --abbrev-ref HEAD) $(git describe origin/master))"
changedPackages="$(npx lerna ls -p --since $BRANCH_POINT --include-dependents)"
```
