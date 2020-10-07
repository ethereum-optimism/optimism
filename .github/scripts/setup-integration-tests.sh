#!/bin/bash

git clone https://github.com/ethereum-optimism/optimism-integration.git \
    $HOME/optimism-integration \
    --recurse-submodules

REPO=$(echo $GITHUB_REPOSITORY | cut -d '/' -f2)

cd $HOME/optimism-integration/$REPO

echo "GITHUB_SHA"
echo $GITHUB_SHA

echo "pwd $PWD"

git fetch
git checkout $GITHUB_SHA

$HOME/optimism-integration/build.sh
$HOME/optimism-integration/test.sh
