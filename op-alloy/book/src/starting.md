# Installation

[op-alloy][op-alloy] consists of a number of crates that provide a range of functionality
essential for interfacing with any OP Stack chain.

The most succinct way to work with `op-alloy` is to add the [`op-alloy`][op-alloy-crate] crate
with the `full` feature flag from the command-line using Cargo.

```txt
cargo add op-alloy --features full
```

Alternatively, you can add the following to your `Cargo.toml` file.

```txt
op-alloy = { version = "0.5", features = ["full"] }
```

For more fine-grained control over the features you wish to include, you can add the individual
crates to your `Cargo.toml` file, or use the `op-alloy` crate with the features you need.

After `op-alloy` is added as a dependency, crates re-exported by `op-alloy` are now available.

```rust
use op_alloy::{
   genesis::{RollupConfig, SystemConfig},
   consensus::OpBlock,
   protocol::BlockInfo,
   network::Optimism,
   provider::ext::engine::OpEngineApi,
   rpc_types::OpTransactionReceipt,
   rpc_jsonrpsee::traits::RollupNode,
   rpc_types_engine::OpAttributesWithParent,
};
```

## Features

The [`op-alloy`][op-alloy-crate] defines many [feature flags][op-alloy-ff] including the following.

Default
- `std`
- `k256`
- `serde`

Full enables the most commonly used crates.
- `full`

The `k256` feature flag enables the `k256` feature on the `op-alloy-consensus` crate.
- `k256`

Arbitrary enables arbitrary features on crates, deriving the `Arbitrary` trait on types.
- `arbitrary`

Serde derives serde's Serialize and Deserialize traits on types.
- `serde`

Additionally, individual crates can be enabled using their shorthand names.
For example, the `consensus` feature flag provides the `op-alloy-consensus` re-export
so `op-alloy-consensus` types can be used from `op-alloy` through `op_alloy::consensus::InsertTypeHere`.

## Crates

- [`op-alloy-network`][op-alloy-network]
- [`op-alloy-provider`][op-alloy-provider]
- [`op-alloy-consensus`][op-alloy-consensus] (supports `no_std`)
- [`op-alloy-rpc-jsonrpsee`][op-alloy-rpc-jsonrpsee]
- [`op-alloy-rpc-types`][op-alloy-rpc-types] (supports `no_std`)
- [`op-alloy-rpc-types-engine`][op-alloy-rpc-types-engine] (supports `no_std`)

## `no_std`

As noted above, the following crates are `no_std` compatible.

- [`op-alloy-consensus`][op-alloy-consensus]
- [`op-alloy-rpc-types-engine`][op-alloy-rpc-types-engine]
- [`op-alloy-rpc-types`][op-alloy-rpc-types]

To add `no_std` support to a crate, ensure the [check_no_std][check-no-std]
script is updated to include this crate once `no_std` compatible.


{{#include ./links.md}}
