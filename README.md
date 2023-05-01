<!--![cannon](https://upload.wikimedia.org/wikipedia/commons/8/80/Cannon%2C_Château_du_Haut-Koenigsbourg%2C_France.jpg)-->
<!--![cannon](https://cdn1.epicgames.com/ue/product/Featured/SCIFIWEAPONBUNDLE_featured-894x488-83fbc936b6d86edcbbe892b1a6780224.png)-->
<!--![cannon](https://static.wikia.nocookie.net/ageofempires/images/8/80/Bombard_cannon_aoe2DE.png/revision/latest/top-crop/width/360/height/360?cb=20200331021834)-->
![cannon](https://paradacreativa.es/wp-content/uploads/2021/05/Canon-orbital-GTA-01.jpg)

---

Cannon *(cannon cannon cannon)* is an onchain MIPS instruction emulator.
Cannon supports EVM-equivalent fault proofs by enabling Geth to run onchain,
one instruction at a time, as part of an interactive dispute game.

* It's Go code
* ...that runs an EVM
* ...emulating a MIPS machine
* ...running compiled Go code
* ...that runs an EVM


## Directory Layout

```
contracts -- A MIPS emulator implementation, using merkleized state and a pre-image oracle.
example   -- Example programs that can be run and proven with Cannon.
extra     -- Extra scripts and legacy contracts, deprecated.
mipsevm   -- Go tooling to test the onchain MIPS implementation, and generate proof data.
unicorn   -- Sub-module, used by mipsevm for offchain MIPS emulation.
```

## Building

### `unicorn`

To build unicorn from source (git sub-module), run:
```
make libunicorn
```

### `contracts`

The contracts are compiled with [`forge`](https://github.com/foundry-rs/foundry).
```
make contracts
```

### `mipsevm`

This requires `unicorn` to be built, as well as the `contracts` for testing.

To test:
```
make test
```

Also see notes in `mipsevm/go.mod` about the Unicorn dependency, if you wish to use Cannon as a Go library.

## License

MIT, see [`LICENSE`](./LICENSE) file.

**Note: This code is unaudited.**
In NO WAY should it be used to secure any monetary value before testing and auditing.
This is experimental software, and should be treated as such.
The authors of this project make no guarantees of security of ANY KIND.
