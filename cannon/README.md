<!--![cannon](https://upload.wikimedia.org/wikipedia/commons/8/80/Cannon%2C_Château_du_Haut-Koenigsbourg%2C_France.jpg)-->
<!--![cannon](https://cdn1.epicgames.com/ue/product/Featured/SCIFIWEAPONBUNDLE_featured-894x488-83fbc936b6d86edcbbe892b1a6780224.png)-->
<!--![cannon](https://static.wikia.nocookie.net/ageofempires/images/8/80/Bombard_cannon_aoe2DE.png/revision/latest/top-crop/width/360/height/360?cb=20200331021834)-->
<!--![cannon](https://paradacreativa.es/wp-content/uploads/2021/05/Canon-orbital-GTA-01.jpg)-->

---

Cannon *(cannon cannon cannon)* is an onchain MIPS instruction emulator.
Cannon supports EVM-equivalent fault proofs by enabling Geth to run onchain,
one instruction at a time, as part of an interactive dispute game.

* It's Go code
* ...that runs an EVM
* ...emulating a MIPS machine
* ...running compiled Go code
* ...that runs an EVM

For more information, see [Docs](./docs/README.md).

## Usage

The fault proof program is [kona](../rust/kona) (client + host). The kona client
is compiled to a MIPS ELF and loaded into cannon; the kona host runs as a
sub-process to serve pre-image data.

```shell
# Build the kona client MIPS ELF (and the kona prestate artifacts).
cd ../rust
just build-kona-prestates

# Build the kona host binary.
cargo build --release --bin kona-host

# Switch back to cannon, and build the CLI
cd ../cannon
just cannon

# Transform the MIPS kona client binary into the first VM state.
# This outputs state.bin.gz (VM state) and meta.json (for debug symbols).
./bin/cannon load-elf --type singlethreaded-2 --path=../rust/kona/prestate-artifacts-cannon/kona-client-elf

# Run cannon emulator (with example inputs).
# The kona host command is passed into cannon (after the --) and runs as a
# sub-process to provide the pre-image data.
#
# Note:
#  - The L2 RPC is an archive L2 node on OP MAINNET.
#  - The L1 RPC is a non-archive RPC.
#  - Use --rollup-config-path / --l1-config-path instead of --l2-chain-id when
#    running against a devnet rather than a known network.
./bin/cannon run \
    --pprof.cpu \
    --info-at '%10000000' \
    --proof-at '=<TRACE_INDEX>' \
    --stop-at '=<STOP_INDEX>' \
    --snapshot-at '%1000000000' \
    --input ./state.bin.gz \
    -- \
    ../rust/target/release/kona-host \
    single \
    --l1-node-address <L1_URL> \
    --l1-beacon-address <L1_BEACON_URL> \
    --l2-node-address <L2_URL> \
    --l1-head <L1_HEAD> \
    --agreed-l2-head-hash <L2_HEAD> \
    --agreed-l2-output-root <L2_OUTPUT_ROOT> \
    --claimed-l2-output-root <L2_CLAIM> \
    --claimed-l2-block-number <L2_BLOCK_NUMBER> \
    --l2-chain-id <L2_CHAIN_ID> \
    --data-dir /tmp/fpp-database \
    --server

# Add --proof-at '=12345' (or pick other pattern, see --help)
# to pick a step to build a proof for (e.g. exact step, every N steps, etc.)

# Also see `./bin/cannon run --help` for more options
```

## Contracts

The Cannon contracts:
- `MIPS64.sol`: A MIPS emulator implementation, to run a single instruction onchain, with merkleized VM memory.
- `PreimageOracle.sol`: implements the pre-image oracle ABI, to support the instruction execution pre-image requests.

The smart-contracts are integrated into the Optimism monorepo contracts:
[`../packages/contracts-bedrock/src/cannon`](../packages/contracts-bedrock/src/cannon)

## `mipsevm`

`mipsevm` is Go tooling to test the onchain MIPS implementation, and generate proof data.

## `testdata`

Example programs that can be run and proven with Cannon.
Optional dependency, but required for `mipsevm` Go tests.
See [`testdata/Makefile`](testdata/Makefile) for building these MIPS binaries.

## License

MIT, see [`LICENSE`](./LICENSE) file.

