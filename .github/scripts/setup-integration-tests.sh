#!/bin/bash

git clone https://github.com/ethereum-optimism/optimism-integration.git \
    $HOME/optimism-integration \
    --recurse-submodules

REPO=$(echo $GITHUB_REPOSITORY | cut -d '/' -f2)

cd $HOME/optimism-integration/$REPO

REMOTE="$GITHUB_SERVER_URL/$GITHUB_REPOSITORY.git"
git remote add gh $REMOTE

git fetch gh $GITHUB_SHA
git checkout $GITHUB_SHA

cd $HOME/optimism-integration

./build.sh
