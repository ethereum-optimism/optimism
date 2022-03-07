#!/bin/bash

# --------------------------------------------------------------------------------
#
# !! Make sure to run forked_node.sh in another terminal before running this.
#
# In this example, the challenger will challenge the transition from a block
# (`BLOCK`), but pretends that chain state before another block (`WRONG_BLOCK`)
# is the state before the challenged block. Consequently, the challenger will
# disagree with the defender on every single step of the challenge game, and the
# single step to execute will be the very first MIPS instruction executed. The
# reason is that the initial MIPS state Merkle root is stored on-chain, and
# immediately modified to reflect the fact that the input hash for the block is
# written at address 0x3000000.
#
# (The input hash is automatically validated against the blockhash, so note that
# in this demo the challenger has to provide the correct (`BLOCK`) input hash to
# the `initiateChallenge` function of `Challenge.sol`, but will execute as
# though the input hash was the one derived from `WRONG_BLOCK`.)
#
# Because the challenger uses the wrong inputs, it will assert a post-state
# (Merkle root) for the first MIPS instruction that has the wrong input hash at
# 0x3000000. Hence, the challenge will fail.
#
# --------------------------------------------------------------------------------

# Exit if any command fails.
set -e

shout() {
    echo ""
    echo "------------------------------------------------------------"
    echo "$1"
    echo "------------------------------------------------------------"
    echo ""
}

# Print an error if we exit before all commands have run.
exit_trap() {
    [[ $? == 0 ]] && return
    shout "EARLY EXIT: SCRIPT FAILED"
}
trap "exit_trap" EXIT

# chain ID, read by challenge.js, respond.js and assert.js
export ID=0

# block whose transition will be challenged
# this variable is read by challenge.js, respond.js and assert.js
export BLOCK=13284469

# block whose pre-state is used by the challenger instead of the challenged block's pre-state
WRONG_BLOCK=13284491

# clear data from previous runs
mkdir -p /tmp/cannon /tmp/cannon_fault && rm -rf /tmp/cannon/* /tmp/cannon_fault/*

# stored in /tmp/cannon/golden.json
shout "GENERATING INITIAL MEMORY STATE CHECKPOINT"
mipsevm/mipsevm

shout "DEPLOYING CONTRACTS"
npx hardhat run scripts/deploy.js --network hosthat

# challenger will use same initial memory checkpoint and deployed contracts
cp /tmp/cannon/{golden,deployed}.json /tmp/cannon_fault/

shout "FETCHING PREIMAGES FOR REAL BLOCK"
minigeth/go-ethereum $BLOCK

shout "COMPUTING REAL MIPS FINAL MEMORY CHECKPOINT"
mipsevm/mipsevm $BLOCK

shout "FETCHING PREIMAGES FOR WRONG BLOCK"
BASEDIR=/tmp/cannon_fault minigeth/go-ethereum $WRONG_BLOCK

shout "COMPUTING FAKE MIPS FINAL MEMORY CHECKPOINT"
BASEDIR=/tmp/cannon_fault mipsevm/mipsevm $WRONG_BLOCK

# pretend the wrong block's input, checkpoints and preimages are the right block's
ln -s /tmp/cannon_fault/0_$WRONG_BLOCK /tmp/cannon_fault/0_$BLOCK

shout "STARTING CHALLENGE"
BASEDIR=/tmp/cannon_fault npx hardhat run scripts/challenge.js --network hosthat

shout "BINARY SEARCH"
for i in {1..23}; do
    echo ""
    echo "--- STEP $i / 23 ---"
    echo ""
    BASEDIR=/tmp/cannon_fault CHALLENGER=1 npx hardhat run scripts/respond.js --network hosthat
    npx hardhat run scripts/respond.js --network hosthat
done

shout "ASSERTING AS CHALLENGER (should fail)"
set +e # this should fail!
BASEDIR=/tmp/cannon_fault CHALLENGER=1 npx hardhat run scripts/assert.js --network hosthat
set -e

shout "ASSERTING AS DEFENDER (should pass)"
npx hardhat run scripts/assert.js --network hosthat
