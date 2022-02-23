#!/bin/bash

# --------------------------------------------------------------------------------
#
# !! Make sure to run forked_node.sh in another terminal before running this.
#
# Unlike the simple scenario (cf. challenge_simple.sh), in this
# challenge-response scenario we use the correct block data (preimages) and
# instead use the `OUTPUTFAULT` environment variable to request a fault in the
# challenger's execution, making his challenge invalid.
#
# The "fault" in question is a behaviour hardcoded in `mipsevm` (Unicorn mode
# only) which triggers when the `OUTPUTFAULT` env var is set: when writing to
# MIPS address 0x30000804 (address where the output hash is written at the end
# of execution), it will write a wrong value instead.
#
# Alternatively, if `REGFAULT` is set, it should contain a MIPS execution step
# number and causes the MIPS register V0 to be set to a bogus value at the given
# execution step. (Just like before, this behaviour is hardcoded in `mipsevm` in
# Unicorn mode and triggers when `REGFAULT` is set.)
#
# This is much slower than the previous scenario because:
#
# - Since we write to the output hash at the end of execution, we will execute ~
#   `log(n) * 3/4 * n` MIPS steps (where `n` = number of steps in full
#   execution) vs `log(n) * 1/4 * n`in the previous example. (This is the
#   difference of having the fault occur in the first vs (one of) the last
#   steps.)
#
# - The challenged block contains almost 4x as many transactions as the original
#   (8.5M vs 30M gas).
#
# --------------------------------------------------------------------------------

# Exit if any command fails.
set -e

# Print an error if we exit before all commands have run.
exit_trap() {
    [[ $? == 0 ]] && return
    echo "----------------------------------------"
    echo "EARLY EXIT: SCRIPT FAILED"
    echo "----------------------------------------"
}
trap "exit_trap" EXIT

# block whose transition will be challenged
# this variable is read by challenge.js, respond.js and assert.js
BLOCK=13284491

# chain ID, read by challenge.js, respond.js and assert.js
export ID=0

# clear data from previous runs
mkdir -p /tmp/cannon /tmp/cannon_fault && rm -rf /tmp/cannon/* /tmp/cannon_fault/*

# generate initial memory state checkpoint (in /tmp/cannon/golden.json)
mipsevm/mipsevm

# deploy contracts
npx hardhat run scripts/deploy.js --network hosthat

# challenger will use same initial memory checkpoint and deployed contracts
cp /tmp/cannon/{golden,deployed}.json /tmp/cannon_fault/

# fetch preimages for real block
minigeth/go-ethereum $BLOCK

# compute real MIPS checkpoint
mipsevm/mipsevm $BLOCK

# fetch preimages for fake block (real block modified with a fault)
# these are the same preimages as for the real block, but we're using a different basedir
BASEDIR=/tmp/cannon_fault minigeth/go-ethereum $BLOCK

# compute fake MIPS checkpoint (includes a fault)
# the output file will be different than for the real block
OUTPUTFAULT=1 BASEDIR=/tmp/cannon_fault mipsevm/mipsevm $BLOCK

# alternatively, to inject a fault in registers instead of memory
# REGFAULT=13240000 BASEDIR=/tmp/cannon_fault mipsevm/mipsevm $BLOCK

# start challenge
BASEDIR=/tmp/cannon_fault npx hardhat run scripts/challenge.js --network hosthat

# binary search
for i in {1..25};do
    OUTPUTFAULT=1 BASEDIR=/tmp/cannon_fault CHALLENGER=1 npx hardhat run scripts/respond.js --network hosthat
    npx hardhat run scripts/respond.js --network hosthat
done

# assert as challenger (fails)
set +e # this should fail!
BASEDIR=/tmp/cannon_fault CHALLENGER=1 npx hardhat run scripts/assert.js --network hosthat
set -e

# assert as defender (passes)
npx hardhat run scripts/assert.js --network hosthat
