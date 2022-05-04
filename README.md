<!--![cannon](https://upload.wikimedia.org/wikipedia/commons/8/80/Cannon%2C_Château_du_Haut-Koenigsbourg%2C_France.jpg)-->
<!--![cannon](https://cdn1.epicgames.com/ue/product/Featured/SCIFIWEAPONBUNDLE_featured-894x488-83fbc936b6d86edcbbe892b1a6780224.png)-->
<!--![cannon](https://static.wikia.nocookie.net/ageofempires/images/8/80/Bombard_cannon_aoe2DE.png/revision/latest/top-crop/width/360/height/360?cb=20200331021834)-->
![cannon](https://paradacreativa.es/wp-content/uploads/2021/05/Canon-orbital-GTA-01.jpg)

---

**NEW: Cannon is currently the object of [a bug bounty on Immunefi](https://immunefi.com/bounty/optimismcannon/). Find vulnerabilities
in Cannon for up to a $50.000 payout.**

- Please take note of [the honest defender assumption](https://github.com/ethereum-optimism/cannon/issues/63)

---

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

## Building

Pre-requisites: Go, Node.js, and Make.

```
make build
make test # verify everything works correctly
```

## Usage

The following commands should be run from the root directory unless otherwise specified:

```
# compute the transition from 13284469 -> 13284470 on PC
TRANSITION_BLOCK=13284469
mkdir -p /tmp/cannon
minigeth/go-ethereum $TRANSITION_BLOCK

# write out the golden MIPS minigeth start state
mipsevm/mipsevm

# if you run into "digital envelope routines::unsupported", rerun after this:
# export NODE_OPTIONS=--openssl-legacy-provider

# generate MIPS checkpoints
mipsevm/mipsevm $TRANSITION_BLOCK

# deploy the MIPS and challenge contracts
npx hardhat run scripts/deploy.js
```

## Examples

The script files [`demo/challenge_simple.sh`](demo/challenge_simple.sh) and
[`demo/challenge_fault.sh`](demo/challenge_fault.sh) present two example scenarios demonstrating the
whole process of a fault proof, including the challenge game and single step verification.

- In the `simple` challenge, the challenger uses the wrong block data in his challenge.
- In the `fault` scenario, fault injection is used to alter the challenger's memory at a specific
  step of the execution.

In both cases, the challenger fails to challenge the block. Refer to the documentation string at the
top of these file for more details regarding the scenario.

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
