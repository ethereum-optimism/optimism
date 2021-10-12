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
mipigo -- minigeth compiled for MIPS. Outputs a binary that's run and mapped at 0x0
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

## WINNER

The first block transition has been computed "on chain" in 14 minutes

```
  89100000 pc: 1086f8 steps per s 106820.330833 ram entries 1395627
  89200000 pc: 107588 steps per s 106824.461317 ram entries 1395672
  89300000 pc: 11e4a0 steps per s 106827.491612 ram entries 1395780
  89400000 pc: 107568 steps per s 106830.298848 ram entries 1395830
  89500000 pc: 107c50 steps per s 106833.903685 ram entries 1395881
new Root 0x9e0261efe4509912b8862f3d45a0cb8404b99b239247df9c55871bd3844cebbd
  89600000 pc: a6cf8 steps per s 106835.300542 ram entries 1396355
  89700000 pc: a80c8 steps per s 106834.819235 ram entries 1396601
  89800000 pc: 8aec0 steps per s 106837.707869 ram entries 1396753
  89900000 pc: 1086d0 steps per s 106840.803960 ram entries 1396829
  90000000 pc: 11e454 steps per s 106843.869077 ram entries 1396985
  90100000 pc: 107dcc steps per s 106847.450632 ram entries 1396943
receipt count 5 hash 0xa2947195971207f3654f635af06f2ab5d3a57af7a834ac88446afd3e8105e57c
process done with hash 0x5c45998dfbf9ce70bcbb80574ed7a622922d2c775e0a2331fe5a8b8dcc99f490 -> 0x9e0261efe4509912b8862f3d45a0cb8404b99b239247df9c55871bd3844cebbd
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
