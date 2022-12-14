#!/bin/bash

# This file exists because as of yarn@1.12.3, --frozen-lockfile is completely
# broken when combined with Yarn workspaces. See https://github.com/yarnpkg/yarn/issues/6291
# This runs on CI to make sure yarn lockfile is up to date before merging a pr

CKSUM_BEFORE=$(cksum yarn.lock)
yarn install
CKSUM_AFTER=$(cksum yarn.lock)


if [[ $CKSUM_BEFORE != $CKSUM_AFTER ]]; then
  echo "yarn_frozen_lockfile.sh: yarn.lock was modified unexpectedly - terminating"
  exit 1
fi