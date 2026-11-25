## `kona-registry`

[`kona-registry`][sc] is a `no_std` crate that exports rust type definitions for chains
in the [`superchain-registry`][osr]. Since it reads static files to read configurations for
various chains into instantiated objects, the [`kona-registry`][sc] crate requires
[`serde`][serde] as a dependency. To use the [`kona-registry`][sc] crate, add the crate
as a dependency to a `Cargo.toml`.

```toml
kona-registry = "0.1.0"
```

[`kona-registry`][sc] declares lazy evaluated statics that expose `ChainConfig`s, `RollupConfig`s,
and `Chain` objects for all chains with static definitions in the superchain registry. The way this works
is the golang side of the superchain registry contains an "internal code generation" script that has
been modified to output configuration files to this crate's [`etc`][s] folder
that are read by the [`kona-registry`][sc] rust crate. These static config files
contain an up-to-date list of all superchain configurations with their chain configs. It is expected
that if the commit hash of the [`superchain-registry`][osr] pulled in as a git submodule has breaking
changes, the tests in this crate (`kona-registry`) will break and updates will need to be made.

There are three core statics exposed by the [`kona-registry`][sc].
- `CHAINS`: A list of chain objects containing the superchain metadata for this chain.
- `OPCHAINS`: A map from chain id to `ChainConfig`.
- `ROLLUP_CONFIGS`: A map from chain id to `RollupConfig`.

[`kona-registry`][sc] exports the _complete_ list of chains within the superchain, as well as each
chain's `RollupConfig`s and `ChainConfig`s.

### Custom chain configurations

`kona-registry` embeds a frozen snapshot of the upstream superchain registry, but downstream
users can extend that snapshot at build time. This is useful when you need bespoke test chains or
partner networks that are not yet part of the public registry but still want to rely on the crate's
lazy statics.

1. Produce JSON files that follow the same schema as the generated artifacts in `etc/`:
   - `chainList.json` containing additional [`Chain`][chains] entries.
   - `configs.json` containing [`Superchain`][superchains] structures with matching `ChainConfig`s and
     `RollupConfig`s for the new chain ids.
   - `depsets.json` containing a JSON array of dependency sets that covers every custom chain. A
     chain outside an interop cluster uses a self-only set:
     ```json
     [{"dependencies": {"<L2_CHAIN_ID>": {}}, "overrideMessageExpiryWindow": null}]
     ```
     Chains in an interop cluster use one entry that lists every member. Each entry must be
     identical to the JSON object in the cluster's dependency-set file: op-supernode's
     `--dependency-set` if set, otherwise op-node's `--interop.dependency-set`. For every chain,
     op-challenger's depset config (`--depset-config`, or `--super-cannon-kona-depset-config` /
     `--cannon-kona-depset-config`) must hold the same object.
2. Point the build to those files by setting the following environment variables during `cargo build`
   (or `cargo test`):
   ```sh
   export KONA_CUSTOM_CONFIGS=true
   export KONA_CUSTOM_CONFIGS_DIR=/absolute/path/to/custom-configs
   cargo build -p kona-registry
   ```
3. The build script copies the committed registry snapshot into Cargo's build output and merges the
   custom files there before compiling the crate. The committed files under `etc/` remain unchanged.
   Attempting to override existing chain ids will result in build failures.

All three JSON files must stay in lockstep: every chain listed in `configs.json` must also appear in
`chainList.json`, chain identifiers must map to a single chain id, every chain in the merged
`chainList.json` must appear in a dependency set, and every cluster that the chains' `interop`
sections declare must be one of the dependency sets. A custom chain's group must also be on Ethereum
mainnet, Sepolia or Holesky, the only L1s whose configs `kona-registry` embeds: on any other L1 the
fault proof programs read the L1 config from a local preimage key that dispute games cannot load.
The build script validates these invariants and fails if one does not hold or if `depsets.json` is
missing. When publishing another crate that depends on `kona-registry`, you can check the custom
artifacts into your workspace and set `KONA_CUSTOM_CONFIGS_DIR` via a build script or `just` recipe
so that consumers automatically embed the additional definitions.

### Usage

Add the following to your `Cargo.toml`.

```toml
[dependencies]
kona-registry = "0.1.0"
```

To make `kona-registry` `no_std`, toggle `default-features` off like so.

```toml
[dependencies]
kona-registry = { version = "0.1.0", default-features = false }
```

Below demonstrates getting the `RollupConfig` for OP Mainnet (Chain ID `10`).

```rust
use kona_registry::ROLLUP_CONFIGS;

let op_chain_id = 10;
let op_rollup_config = ROLLUP_CONFIGS.get(&op_chain_id);
println!("OP Mainnet Rollup Config: {:?}", op_rollup_config);
```

A mapping from chain id to `ChainConfig` is also available.

```rust
use kona_registry::OPCHAINS;

let op_chain_id = 10;
let op_chain_config = OPCHAINS.get(&op_chain_id);
println!("OP Mainnet Chain Config: {:?}", op_chain_config);
```


### Feature Flags

- `std`: Uses the standard library to pull in environment variables.


### Credits

[superchain-registry][osr] contributors for building and maintaining superchain types.

[alloy] and [op-alloy] for creating and maintaining high quality Ethereum and Optimism types in rust.


<!-- Hyperlinks -->

[serde]: https://crates.io/crates/serde
[alloy]: https://github.com/alloy-rs/alloy
[op-alloy]: https://github.com/ethereum-optimism/optimism/tree/develop/rust/op-alloy
[op-superchain]: https://docs.optimism.io/stack/explainer
[osr]: https://github.com/ethereum-optimism/superchain-registry

[s]: ./etc
[sc]: https://crates.io/crates/kona-registry
[g]: https://crates.io/crates/kona-genesis

[chains]: https://docs.rs/kona-registry/latest/kona_registry/struct.CHAINS.html
[opchains]: https://docs.rs/kona-registry/latest/kona_registry/struct.OPCHAINS.html
[rollups]: https://docs.rs/kona-registry/latest/kona_registry/struct.ROLLUP_CONFIGS.html
[superchains]: https://docs.rs/kona-genesis/latest/kona_genesis/struct.Superchain.html
