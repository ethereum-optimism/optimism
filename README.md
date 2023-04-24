<!--![cannon](https://upload.wikimedia.org/wikipedia/commons/8/80/Cannon%2C_Château_du_Haut-Koenigsbourg%2C_France.jpg)-->
<!--![cannon](https://cdn1.epicgames.com/ue/product/Featured/SCIFIWEAPONBUNDLE_featured-894x488-83fbc936b6d86edcbbe892b1a6780224.png)-->
<!--![cannon](https://static.wikia.nocookie.net/ageofempires/images/8/80/Bombard_cannon_aoe2DE.png/revision/latest/top-crop/width/360/height/360?cb=20200331021834)-->
![cannon](https://paradacreativa.es/wp-content/uploads/2021/05/Canon-orbital-GTA-01.jpg)

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
contracts -- A MIPS emulator implementation, using merkleized state and a pre-image oracle.
example   -- Example programs that can be run and proven with Cannon.
extra     -- Extra scripts and legacy contracts, deprecated.
mipsevm   -- Go tooling to test the onchain MIPS implementation, and generate proof data.
unicorn   -- Sub-module, used by mipsevm for offchain MIPS emulation.
```

## Building

Pre-requisites: Go, Node.js, Make, and CMake.

```
make build
make test # verify everything works correctly
```

## License

MIT, see [`LICENSE`](./LICENSE) file.

Note: This code is unaudited. It in NO WAY should be used to secure any money until a lot more
testing and auditing are done. I have deployed this nowhere, have advised against deploying it, and
make no guarantees of security of ANY KIND.
