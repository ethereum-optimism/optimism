# `kona-derive`

<a href="https://github.com/op-rs/kona/actions/workflows/rust_ci.yaml"><img src="https://github.com/op-rs/kona/actions/workflows/rust_ci.yaml/badge.svg?label=ci" alt="CI"></a>
<a href="https://crates.io/crates/kona-derive"><img src="https://img.shields.io/crates/v/kona-derive.svg?label=kona-derive&labelColor=2a2f35" alt="Kona Derive"></a>
<a href="https://github.com/op-rs/kona/blob/main/LICENSE.md"><img src="https://img.shields.io/badge/License-MIT-d1d1f6.svg?label=license&labelColor=2a2f35" alt="License"></a>
<a href="https://app.codecov.io/github/op-rs/kona"><img src="https://img.shields.io/codecov/c/github/op-rs/kona" alt="Codecov"></a>

A `no_std` compatible implementation of the OP Stack's [derivation pipeline][derive].

[derive]: (https://specs.optimism.io/protocol/derivation.html#l2-chain-derivation-specification).

## Usage

The intended way of working with `kona-derive` is to use the [`DerivationPipeline`][dp] which implements the [`Pipeline`][p] trait. To create an instance of the [`DerivationPipeline`][dp], it's recommended to use the [`PipelineBuilder`][pb] as follows.

```rust,ignore
use std::sync::Arc;
use kona_genesis::RollupConfig;
use kona_derive::EthereumDataSource;
use kona_derive::PipelineBuilder;
use kona_derive::StatefulAttributesBuilder;

let chain_provider = todo!();
let l2_chain_provider = todo!();
let blob_provider = todo!();
let l1_origin = todo!();

let cfg = Arc::new(RollupConfig::default());
let attributes = StatefulAttributesBuilder::new(
   cfg.clone(),
   l2_chain_provider.clone(),
   chain_provider.clone(),
);
let dap = EthereumDataSource::new(
   chain_provider.clone(),
   blob_provider,
   cfg.as_ref()
);

// Construct a new derivation pipeline.
let pipeline = PipelineBuilder::new()
   .rollup_config(cfg)
   .dap_source(dap)
   .l2_chain_provider(l2_chain_provider)
   .chain_provider(chain_provider)
   .builder(attributes)
   .origin(l1_origin)
   .build();
```

[p]: ./src/traits/pipeline.rs
[pb]: ./src/pipeline/builder.rs
[dp]: ./src/pipeline/core.rs

## Features

The most up-to-date feature list will be available on the [docs.rs `Feature Flags` tab][ff] of the `kona-derive` crate.

Some features include the following.
- `serde`: Serialization and Deserialization support for `kona-derive` types.
- `test-utils`: Test utilities for downstream libraries.

By default, `kona-derive` enables the `serde` feature.

[ap]: https://docs.rs/crate/alloy-providers/latest
[ff]: https://docs.rs/crate/kona-derive/latest/features
