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

# compute the transition from 13284469 -> 13284470 on PC
mkdir -p /tmp/cannon
minigeth/go-ethereum 13284469

# write out the golden MIPS minigeth start state
yarn
(cd mipsevm && ./evm.sh)

# generate MIPS checkpoints for 13284469 -> 13284470
mipsevm/mipsevm 13284469

# deploy the MIPS and challenge contracts
npx hardhat run scripts/deploy.js
```

## Full Challenge / Response

```
# testing on hardhat (forked mainnet, a few blocks ahead of challenge)
npx hardhat node --fork https://mainnet.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161 --fork-block-number 13284495

# challenger is pretending the block 13284491 transition is the transition for 13284469
# this will conflict at the first step
mkdir -p /tmp/cannon /tmp/cannon_fault && rm -rf /tmp/cannon/* /tmp/cannon_fault/*
mipsevm/mipsevm
npx hardhat run scripts/deploy.js --network hosthat
cp /tmp/cannon/*.json /tmp/cannon_fault/

# compute real MIPS checkpoint
minigeth/go-ethereum 13284469 && mipsevm/mipsevm 13284469

# compute fake MIPS checkpoint (use a symlink to pretend)
BASEDIR=/tmp/cannon_fault minigeth/go-ethereum 13284491 && BASEDIR=/tmp/cannon_fault mipsevm/mipsevm 13284491
ln -s /tmp/cannon_fault/0_13284491 /tmp/cannon_fault/0_13284469

BASEDIR=/tmp/cannon_fault BLOCK=13284469 npx hardhat run scripts/challenge.js --network hosthat

# do binary search
for i in {1..23}
do
BASEDIR=/tmp/cannon_fault ID=0 BLOCK=13284469 CHALLENGER=1 npx hardhat run scripts/respond.js --network hosthat
ID=0 BLOCK=13284469 npx hardhat run scripts/respond.js --network hosthat
done

# assert as challenger (fails)
BASEDIR=/tmp/cannon_fault ID=0 BLOCK=13284469 CHALLENGER=1 npx hardhat run scripts/assert.js --network hosthat

# assert as defender (passes)
ID=0 BLOCK=13284469 npx hardhat run scripts/assert.js --network hosthat
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

Note: This code is unaudited. It in NO WAY should be used to secure any money until a lot more testing and auditing are done. I have deployed this nowhere, have advised against deploying it, and make no guarantees of security of ANY KIND.
