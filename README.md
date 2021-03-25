# Optimism Monorepo (VERY WIP)

## Taming the Monorepo

1. You solely use yarn workspaces for the Mono-Repo workflow.
1. You use lerna’s utility commands to optimize managing of multiple packages, e.g., selective execution of npm scripts for testing.
1. You use lerna for publishing packages since lerna provides sophisticated features with its version and publish commands.

## Incremental Tests

```
BRANCH_POINT="$(git merge-base $(git rev-parse --abbrev-ref HEAD) $(git describe origin/master))"
changedPackages="$(npx lerna ls -p --since $BRANCH_POINT --include-dependents)"
```

## Goals


## Ops

https://github.com/connext/vector/tree/main/ops
https://github.com/connext/vector/blob/main/Makefile
https://github.com/connext/vector/blob/main/.github/workflows/prod.yml
https://github.com/connext/vector/blob/main/.github/workflows/feature.yml

https://www.npmjs.com/package/depcheck


## Lerna import
https://medium.com/zocdoc-engineering/lerna-you-a-monorepo-the-nuts-and-bolts-of-building-a-ci-pipeline-with-lerna-850e6a290bb2
