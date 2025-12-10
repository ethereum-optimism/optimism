# `kona-providers-local`

<a href="https://github.com/op-rs/kona/actions/workflows/rust_ci.yaml"><img src="https://github.com/op-rs/kona/actions/workflows/rust_ci.yaml/badge.svg?label=ci" alt="CI"></a>
<a href="https://crates.io/crates/kona-providers-local"><img src="https://img.shields.io/crates/v/kona-providers-alloy.svg?label=kona-providers-local&labelColor=2a2f35" alt="kona-provides-local"></a>
<a href="https://github.com/op-rs/kona/blob/main/LICENSE.md"><img src="https://img.shields.io/badge/License-MIT-d1d1f6.svg?label=license&labelColor=2a2f35" alt="License"></a>
<a href="https://img.shields.io/codecov/c/github/op-rs/kona"><img src="https://img.shields.io/codecov/c/github/op-rs/kona" alt="Codecov"></a>


This crate provides a pure in-memory L2 provider implementation for the Kona OP Stack. It operates without any external RPC dependencies, serving all data from its internal cache.

## Features

- **BufferedL2Provider**: A pure in-memory L2 provider that serves data from cached blocks
- **ChainStateBuffer**: LRU cache for managing chain state with reorganization support
- **Chain Event Handling**: Support for processing execution extension notifications for chain events (commits, reorgs, reverts)
- **No External Dependencies**: Operates entirely from in-memory state without RPC calls

## Architecture

The buffered provider operates as a standalone in-memory data store:

1. **In-Memory Storage**: Complete blocks with L2 block info are stored in memory
2. **Dual Indexing**: Blocks are indexed by both hash and number for efficient queries
3. **Reorg Handling**: Intelligent cache invalidation during chain reorganizations up to a configurable depth
4. **Event Processing**: Integration with execution extension notifications to maintain cache consistency
5. **Genesis Support**: Special handling for genesis blocks from the rollup configuration

## Usage

```rust,ignore
use kona_providers_local::{BufferedL2Provider, ChainStateEvent};
use kona_genesis::RollupConfig;
use kona_protocol::{BatchValidationProvider, L2BlockInfo};
use op_alloy_consensus::OpBlock;
use std::sync::Arc;

async fn example() -> Result<(), Box<dyn std::error::Error>> {
    // Create a buffered provider with rollup configuration
    let rollup_config = Arc::new(RollupConfig::default());
    let provider = BufferedL2Provider::new(rollup_config, 1000, 64);

    // Add blocks to the provider
    // In practice, these would come from execution extension or other sources
    let block: OpBlock = unimplemented!();
    let l2_info: L2BlockInfo = unimplemented!();
    provider.add_block(block, l2_info).await?;

    // Handle chain events from execution extension notifications
    let event = ChainStateEvent::ChainCommitted {
        new_head: alloy_primitives::B256::ZERO,
        committed: vec![],
    };
    provider.handle_chain_event(event).await?;

    // Query blocks from the cache
    let mut provider_clone = provider.clone();
    let block = provider_clone.block_by_number(1).await?;
    let l2_info = provider_clone.l2_block_info_by_number(1).await?;

    Ok(())
}
```

## Configuration

- `cache_size`: Number of blocks to cache (affects memory usage)
- `max_reorg_depth`: Maximum reorganization depth to handle before clearing cache

## Provider Traits

The `BufferedL2Provider` implements the following traits from `kona-derive`:

- `ChainProvider`: Basic block and receipt access
- `L2ChainProvider`: L2-specific functionality including system config access
- `BatchValidationProvider`: Batch validation support

## Error Handling

The provider returns specific errors for different failure scenarios:
- `BlockNotFound`: When a requested block is not in the cache
- `L2BlockInfoConstruction`: When L2 block info cannot be constructed
- `SystemConfigConversion`: When a block cannot be converted to system config
- `Buffer` errors: For cache-related issues including deep reorgs
