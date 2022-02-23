<!--![cannon](https://upload.wikimedia.org/wikipedia/commons/8/80/Cannon%2C_Château_du_Haut-Koenigsbourg%2C_France.jpg)-->
<!--![cannon](https://cdn1.epicgames.com/ue/product/Featured/SCIFIWEAPONBUNDLE_featured-894x488-83fbc936b6d86edcbbe892b1a6780224.png)-->
<!--![cannon](https://static.wikia.nocookie.net/ageofempires/images/8/80/Bombard_cannon_aoe2DE.png/revision/latest/top-crop/width/360/height/360?cb=20200331021834)-->
![cannon](https://paradacreativa.es/wp-content/uploads/2021/05/Canon-orbital-GTA-01.jpg)

The cannon (cannon cannon cannon) is an on chain interactive dispute engine implementing EVM-equivalent fault proofs.

It's half geth, half MIPS, and whole awesome.

* It's Go code
* ...that runs an EVM
* ...emulating a MIPS machine
* ...running compiled Go code
* ...that runs an EVM

For more information on Cannon's inner workings, check [this overview][overview].

[overview]: https://github.com/ethereum-optimism/optimistic-specs/wiki/Cannon-Overview

## Directory Layout

```
minigeth -- A standalone "geth" capable of computing a block transition
mipigo -- minigeth compiled for MIPS. Outputs a MIPS binary that's run and mapped at 0x0
mipsevm -- A MIPS runtime in the EVM (works with contracts)
contracts -- A Merkleized MIPS processor on chain + the challenge logic
```

## Usage

The following commands should be run from the root directory unless otherwise specified:

```
./build_unicorn.sh

# build minigeth for MIPS
(cd mipigo && ./build.sh)

# build minigeth for PC
(cd minigeth/ && go build)

# compute the transition from 13284469 -> 13284470 on PC
TRANSITION_BLOCK=13284469
mkdir -p /tmp/cannon
minigeth/go-ethereum $TRANSITION_BLOCK

# write out the golden MIPS minigeth start state
yarn
(cd mipsevm && ./evm.sh)

# if you run into "digital envelope routines::unsupported", rerun after this:
# export NODE_OPTIONS=--openssl-legacy-provider

# generate MIPS checkpoints
mipsevm/mipsevm $TRANSITION_BLOCK

# deploy the MIPS and challenge contracts
npx hardhat run scripts/deploy.js
```

## Full Challenge / Response

In this example, the challenger will challenge the transition from a block (`BLOCK`), but pretends
that chain state before another block (`WRONG_BLOCK`) is the state before the challenged block.
Consequently, the challenger will disagree with the defender on every single step of the challenge
game, and the single step to execute will be the very first MIPS instruction executed. The reason is
that the initial MIPS state Merkle root is stored on-chain, and immediately modified to reflect the
fact that the input hash for the block is written at address 0x3000000.

(The input hash is automatically validated against the blockhash, so note that in this demo the
challenger has to provide the correct (`BLOCK`) input hash to the `InitiateChallenge` function of
`Challenge.sol`, but will execute as though the input hash was the one derived from `WRONG_BLOCK`.)

Because the challenger uses the wrong inputs, it will assert a post-state (Merkle root) for the
first MIPS instruction that has the wrong input hash at 0x3000000. Hence, the challenge will fail.

```
RPC_URL=https://mainnet.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161

# block at which to fork mainnet
FORK_BLOCK=13284495

# testing on hardhat (forked mainnet, a few blocks ahead of challenge)
npx hardhat node --fork $RPC_URL --fork-block-number $FORK_BLOCK

# open a new terminal for the following commands

# chain ID, read by challenge.js, respond.js and assert.js
export ID=0

# block whose transition will be challenged
# this variable is read by challenge.js, respond.js and assert.js
export BLOCK=13284469

# block whose pre-state is used by the challenger instead of the challenged block's pre-state
WRONG_BLOCK=13284491

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

# compute real MIPS final memory checkpoint
mipsevm/mipsevm $BLOCK

# fetch preimages for wrong block
BASEDIR=/tmp/cannon_fault minigeth/go-ethereum $WRONG_BLOCK

# compute fake MIPS final memory checkpoint
BASEDIR=/tmp/cannon_fault mipsevm/mipsevm $WRONG_BLOCK

# pretend the wrong block's input, checkpoints and preimages are the right block's
ln -s /tmp/cannon_fault/0_$WRONG_BLOCK /tmp/cannon_fault/0_$BLOCK

# start challenge
BASEDIR=/tmp/cannon_fault npx hardhat run scripts/challenge.js --network hosthat

# binary search
for i in {1..23}; do
    BASEDIR=/tmp/cannon_fault CHALLENGER=1 npx hardhat run scripts/respond.js --network hosthat
    npx hardhat run scripts/respond.js --network hosthat
done

# assert as challenger (fails)
BASEDIR=/tmp/cannon_fault CHALLENGER=1 npx hardhat run scripts/assert.js --network hosthat

# assert as defender (passes)
npx hardhat run scripts/assert.js --network hosthat
```

## Alternate challenge with output fault (much slower)

```
# START setup (same as previous example)

RPC_URL=https://mainnet.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161

# block at which to fork mainnet
FORK_BLOCK=13284495

# testing on hardhat (forked mainnet, a few blocks ahead of challenge)
npx hardhat node --fork $RPC_URL --fork-block-number $FORK_BLOCK

# open a new terminal for the following commands

# END setup

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
BASEDIR=/tmp/cannon_fault CHALLENGER=1 npx hardhat run scripts/assert.js --network hosthat

# assert as defender (passes)
npx hardhat run scripts/assert.js --network hosthat
```

## State Oracle API

On chain / in MIPS, we have two simple oracles

* InputHash() -> hash
* Preimage(hash) -> value

We generate the Preimages in x86 using geth RPC

* PrefetchAccount
* PrefetchStorage
* PrefetchCode
* PrefetchBlock

These are NOP in the VM

## License

Most of this code is MIT licensed, minigeth is LGPL3.

Note: This code is unaudited. It in NO WAY should be used to secure any money until a lot more
testing and auditing are done. I have deployed this nowhere, have advised against deploying it, and
make no guarantees of security of ANY KIND.
