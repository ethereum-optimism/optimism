#!/usr/bin/env bash

# The following variables can be overridden as environment variables:
# * BLOCK (block whose transition will be challenged)
# * SKIP_NODE (skip forking a node, useful if you've already forked a node)
#
# Example usage:
# SKIP_NODE=1 BLOCK=13284469 ./demo/challenge_fault.sh

# --- DOC ----------------------------------------------------------------------

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


# --- SCRIPT SETUP -------------------------------------------------------------

shout() {
    echo ""
    echo "----------------------------------------"
    echo "$1"
    echo "----------------------------------------"
    echo ""
}

# Exit if any command fails.
set -e

exit_trap() {
    # Print an error if the last command failed
    # (in which case the script is exiting because of set -e).
    [[ $? == 0 ]] && return
    echo "----------------------------------------"
    echo "EARLY EXIT: SCRIPT FAILED"
    echo "----------------------------------------"

    # Kill (send SIGTERM) to the whole process group, also killing
    # any background processes.
    # I think the trap command resets SIGTERM before resending it to the whole
    # group. (cf. https://stackoverflow.com/a/2173421)
    trap - SIGTERM && kill -- -$$
}
trap "exit_trap" SIGINT SIGTERM EXIT

# --- BOOT MAINNET FORK --------------------------------------------------------

if [[ ! "$SKIP_NODE" ]]; then
    NODE_LOG="challenge_fault_node.log"

    shout "BOOTING MAINNET FORK NODE IN BACKGROUND (LOG: $NODE_LOG)"

    # get directory containing this file
    SCRIPT_DIR=$(dirname "$(readlink -f "$0")")

    # run a hardhat mainnet fork node
    "$SCRIPT_DIR/forked_node.sh" > "$NODE_LOG" 2>&1 &

    # give the node some time to boot up
    sleep 10
fi

# --- CHALLENGE SETUP ----------------------------------------------------------

# hardhat network to use
NETWORK=${NETWORK:-l1}
export NETWORK

# block whose transition will be challenged
# this variable is read by challenge.js, respond.js and assert.js
BLOCK=${BLOCK:-13284491}
export BLOCK

# challenge ID, read by respond.js and assert.js
export ID=0

# clear data from previous runs
mkdir -p /tmp/cannon /tmp/cannon_fault && rm -rf /tmp/cannon/* /tmp/cannon_fault/*

# stored in /tmp/cannon/golden.json
shout "GENERATING INITIAL MEMORY STATE CHECKPOINT"
mipsevm/mipsevm

shout "DEPLOYING CONTRACTS"
npx hardhat run scripts/deploy.js --network $NETWORK

# challenger will use same initial memory checkpoint and deployed contracts
cp /tmp/cannon/{golden,deployed}.json /tmp/cannon_fault/

shout "FETCHING PREIMAGES FOR REAL BLOCK"
minigeth/go-ethereum $BLOCK

shout "COMPUTING REAL MIPS FINAL MEMORY CHECKPOINT"
mipsevm/mipsevm $BLOCK

# these are the preimages for the real block (but go into a different basedir)
shout "FETCHING PREIMAGES FOR FAULTY BLOCK"
BASEDIR=/tmp/cannon_fault minigeth/go-ethereum $BLOCK

# since the computation includes a fault, the output file will be different than
# for the real block
shout "COMPUTE FAKE MIPS CHECKPOINT"
OUTPUTFAULT=1 BASEDIR=/tmp/cannon_fault mipsevm/mipsevm $BLOCK

# alternatively, to inject a fault in registers instead of memory
# REGFAULT=13240000 BASEDIR=/tmp/cannon_fault mipsevm/mipsevm $BLOCK

# --- BINARY SEARCH ------------------------------------------------------------

shout "STARTING CHALLENGE"
BASEDIR=/tmp/cannon_fault npx hardhat run scripts/challenge.js --network $NETWORK

shout "BINARY SEARCH"
for i in {1..25}; do
    echo ""
    echo "--- STEP $i / 25 --"
    echo ""
    OUTPUTFAULT=1 BASEDIR=/tmp/cannon_fault CHALLENGER=1 npx hardhat run scripts/respond.js --network $NETWORK
    npx hardhat run scripts/respond.js --network $NETWORK
done

# --- SINGLE STEP EXECUTION ----------------------------------------------------

shout "ASSERTING AS CHALLENGER (should fail)"
set +e # this should fail!
BASEDIR=/tmp/cannon_fault CHALLENGER=1 npx hardhat run scripts/assert.js --network $NETWORK
set -e

shout "ASSERTING AS DEFENDER (should pass)"
npx hardhat run scripts/assert.js --network $NETWORK
