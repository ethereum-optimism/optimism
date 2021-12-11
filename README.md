<!--![cannon](https://upload.wikimedia.org/wikipedia/commons/8/80/Cannon%2C_Château_du_Haut-Koenigsbourg%2C_France.jpg)-->
<!--![cannon](https://cdn1.epicgames.com/ue/product/Featured/SCIFIWEAPONBUNDLE_featured-894x488-83fbc936b6d86edcbbe892b1a6780224.png)-->
<!--![cannon](https://static.wikia.nocookie.net/ageofempires/images/8/80/Bombard_cannon_aoe2DE.png/revision/latest/top-crop/width/360/height/360?cb=20200331021834)-->
![cannon](https://paradacreativa.es/wp-content/uploads/2021/05/Canon-orbital-GTA-01.jpg)

The cannon (cannon cannon cannon) is an on chain interactive fraud prover

It's half geth, half of what I think truebit was supposed to be. It can prove L1 blocks aren't fraud.

* It's Go code
* ...that runs an EVM
* ...emulating a MIPS machine
* ...running compiled Go code
* ...that runs an EVM

## Directory Layout

```
contracts -- A Merkleized MIPS processor on chain + the challenge logic
minigeth -- A standalone "geth" capable of computing a block transition
mipigo -- minigeth compiled for MIPS. Outputs a MIPS binary that's run and mapped at 0x0
mipsevm -- A MIPS runtime in the EVM
```

## Usage
```
# build minigeth for MIPS
(cd mipigo && pip3 install -r requirements.txt && ./build.sh)

# build minigeth for PC
(cd minigeth/ && go build)
mkdir -p /tmp/cannon

# compute the transition from 13284469 -> 13284470 on PC
minigeth/go-ethereum 13284469

# write out the golden MIPS minigeth start state
mipsevm/mipsevm

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
rm -rf /tmp/cannon/*
mipsevm/mipsevm
npx hardhat run scripts/deploy.js --network hosthat

# compute the MIPS checkpoints
minigeth/go-ethereum 13284491 && mipsevm/mipsevm 13284491
minigeth/go-ethereum 13284469 && mipsevm/mipsevm 13284469
BLOCK=13284469 npx hardhat run scripts/challenge.js --network hosthat

# do binary search
for i in {1..23}
do
ID=0 BLOCK=13284491 CHALLENGER=1 npx hardhat run scripts/respond.js --network hosthat
ID=0 BLOCK=13284469 npx hardhat run scripts/respond.js --network hosthat
done

# assert as challenger (fails)
ID=0 BLOCK=13284491 CHALLENGER=1 npx hardhat run scripts/assert.js --network hosthat

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

All my code is MIT license, minigeth is LGPL3. Being developed under contract for @optimismPBC

Note: This code is unaudited. It in NO WAY should be used to secure any money until a lot more testing and auditing are done. I have deployed this nowhere, have advised against deploying it, and make no guarantees of security of ANY KIND.
