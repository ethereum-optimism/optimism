#!/bin/bash

if ! command -v forge &> /dev/null
then
  echo "Is Foundry not installed? Consider installing via `curl -L https://foundry.paradigm.xyz | bash` and then running `foundryup` on a new terminal. For more context, check the installation instructions in the book: https://book.getfoundry.sh/getting-started/installation.html."
  exit 1
fi

VERSION=$(forge --version)
echo "Using foundry version: $VERSION"
