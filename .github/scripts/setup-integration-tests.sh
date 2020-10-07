#!/bin/bash

git clone https://github.com/ethereum-optimism/optimism-integration.git \
    $HOME/optimism-integration \
    --recurse-submodules

REPO=$(echo $GITHUB_REPOSITORY | cut -d '/' -f2)

cd $HOME/optimism-integration/$REPO

# TODO: this will not work for outside contributors
git fetch origin $GITHUB_SHA
git checkout $GITHUB_SHA

cd $HOME/optimism-integration

./build.sh
./test.sh
