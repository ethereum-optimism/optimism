<!--![cannon](https://upload.wikimedia.org/wikipedia/commons/8/80/Cannon%2C_Château_du_Haut-Koenigsbourg%2C_France.jpg)-->
![cannon](https://cdn1.epicgames.com/ue/product/Featured/SCIFIWEAPONBUNDLE_featured-894x488-83fbc936b6d86edcbbe892b1a6780224.png)
<!--![cannon](https://static.wikia.nocookie.net/ageofempires/images/8/80/Bombard_cannon_aoe2DE.png/revision/latest/top-crop/width/360/height/360?cb=20200331021834)-->

The cannon (cannon cannon cannon) is an on chain interactive fraud prover

It's half geth, half of what I think truebit was supposed to be. When it's done, we'll be able to prove L1 blocks aren't fraud

* It's code in Go
* ...running an EVM
* ...emulating a MIPS machine
* ...running an EVM

## Directory Layout

```
minigeth -- A standalone "geth" capable of computing a block transition
mipigeth -- minigeth compiled for MIPS. Outputs a binary that's run and mapped at 0x0
mipsevm -- A MIPS runtime in the EVM (see also contracts/)
```

## Steps

1. Get minigeth to verify a block locally paying attention to oracle (done)
2. Compile embedded minigeth to MIPS (done)
3. Get embedded minigeth to verify a block using the oracle (done)
4. Merkleize the state of the embedded machine
5. Write Solidity code to verify any MIPS/oracle transitions (done)
6. Write binary search engine to play on chain game (done)

The system is checking an embedded block in CI now

## TODO

* Get minigeth running in Solidity MIPS emulator with reasonable performance (Go code using EVM with native memory)
* Add merkleization for MIPS ReadMemory and WriteMemory

## Usage
```
# verify the transition from 13284469 -> 13284470
./run.sh
```

## State Oracle API

On chain / in MIPS, we have two oracles

* Input(index) -> value
* Preimage(hash) -> value    # hash(value) == hash

We generate the Preimages in x86 using geth RPC

* PrefetchAccount
* PrefetchStorage
* PrefetchCode
* PrefetchBlock

These are NOP in the VM

## License

All my code is MIT license, minigeth is LGPL3. Being developed under contract for @optimismPBC

# Very important TODO

TODO: update to picture of increasingly futuristic cannon as it starts to work
