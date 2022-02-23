#!/bin/bash

# This runs a hardhat node forked from mainnet at the specified block.
# You need to run this in a separate terminal (or in the background)
# before running challenge_simple.sh or challenge_fault.sh.

RPC_URL=https://mainnet.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161

# block at which to fork mainnet
FORK_BLOCK=13284495

# testing on hardhat (forked mainnet, a few blocks ahead of challenges in
# challenge_simple.sh and challenge_fault.sh)
npx hardhat node --fork $RPC_URL --fork-block-number $FORK_BLOCK
