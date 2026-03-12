//! Genesis generation and parsing utilities.
//!
//! Provides helpers to construct a minimal L2 genesis and rollup config for testing,
//! without requiring the full op-deployer Go binary.

use alloy_eips::BlockNumHash;
use alloy_primitives::{Address, B256, U256, address, b256};
use kona_genesis::{HardForkConfig, RollupConfig, SystemConfig};
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::sync::Arc;

/// Default chain ID for test L2 chains.
pub const TEST_L2_CHAIN_ID: u64 = 901;

/// Default block time (2 seconds).
pub const TEST_BLOCK_TIME: u64 = 2;

/// Default max sequencer drift.
pub const TEST_MAX_SEQUENCER_DRIFT: u64 = 600;

/// Default sequencer window size.
pub const TEST_SEQ_WINDOW_SIZE: u64 = 200;

/// Default channel timeout.
pub const TEST_CHANNEL_TIMEOUT: u64 = 120;

/// Default batch inbox address.
pub const TEST_BATCH_INBOX: Address = address!("ff00000000000000000000000000000000000901");

/// Default batcher address.
pub const TEST_BATCHER_ADDR: Address = address!("6887246668a3b87f54deb3b94ba47a6f63f32985");

/// Default L1 genesis block info.
pub fn default_l1_genesis() -> BlockInfo {
    BlockInfo {
        number: 0,
        hash: b256!("0000000000000000000000000000000000000000000000000000000000000001"),
        parent_hash: B256::ZERO,
        timestamp: 1_000_000,
    }
}

/// Default L1 genesis as [`BlockNumHash`].
pub fn default_l1_genesis_num_hash() -> BlockNumHash {
    let info = default_l1_genesis();
    BlockNumHash { number: info.number, hash: info.hash }
}

/// Default L2 genesis block info.
pub fn default_l2_genesis() -> L2BlockInfo {
    let l1_origin = default_l1_genesis();
    L2BlockInfo {
        block_info: BlockInfo {
            number: 0,
            hash: b256!("0000000000000000000000000000000000000000000000000000000000000002"),
            parent_hash: B256::ZERO,
            timestamp: l1_origin.timestamp,
        },
        l1_origin: BlockNumHash {
            number: l1_origin.number,
            hash: l1_origin.hash,
        },
        seq_num: 0,
    }
}

/// Default L2 genesis as [`BlockNumHash`].
pub fn default_l2_genesis_num_hash() -> BlockNumHash {
    let info = default_l2_genesis();
    BlockNumHash { number: info.block_info.number, hash: info.block_info.hash }
}

/// Creates a minimal test [`RollupConfig`] with all hardforks activated at genesis.
pub fn test_rollup_config() -> RollupConfig {
    let l1_genesis = default_l1_genesis_num_hash();
    let l2_genesis = default_l2_genesis();

    RollupConfig {
        genesis: kona_genesis::ChainGenesis {
            l1: l1_genesis,
            l2: BlockNumHash {
                number: l2_genesis.block_info.number,
                hash: l2_genesis.block_info.hash,
            },
            l2_time: l2_genesis.block_info.timestamp,
            system_config: Some(SystemConfig {
                batcher_address: TEST_BATCHER_ADDR,
                overhead: U256::ZERO,
                scalar: U256::from(0x01_0000_0000_0000_0000u128),
                gas_limit: 30_000_000,
                ..Default::default()
            }),
        },
        block_time: TEST_BLOCK_TIME,
        max_sequencer_drift: TEST_MAX_SEQUENCER_DRIFT,
        seq_window_size: TEST_SEQ_WINDOW_SIZE,
        channel_timeout: TEST_CHANNEL_TIMEOUT,
        batch_inbox_address: TEST_BATCH_INBOX,
        l2_chain_id: alloy_chains::Chain::from_id(TEST_L2_CHAIN_ID),
        hardforks: HardForkConfig {
            regolith_time: Some(0),
            canyon_time: Some(0),
            delta_time: Some(0),
            ecotone_time: Some(0),
            fjord_time: Some(0),
            granite_time: Some(0),
            holocene_time: Some(0),
            isthmus_time: Some(0),
            ..Default::default()
        },
        ..Default::default()
    }
}

/// Creates a test rollup config wrapped in an Arc.
pub fn test_rollup_config_arc() -> Arc<RollupConfig> {
    Arc::new(test_rollup_config())
}
