#!/bin/bash

git clone https://github.com/ethereum-optimism/optimism-integration.git \
    $HOME/optimism-integration \
    --recurse-submodules

REPO=$(echo $GITHUB_REPOSITORY | cut -d '/' -f2)

cd $HOME/optimism-integration/$REPO

echo "GITHUB REF"
echo $GITHUB_REF

git checkout $GITHUB_REF

$HOME/optimism-integration/build.sh
$HOME/optimism-integration/test.sh
