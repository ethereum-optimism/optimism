//! OP Mainnet Rollup Config.

use alloy_chains::Chain;
use alloy_eips::BlockNumHash;
use alloy_op_hardforks::{
    OP_MAINNET_CANYON_TIMESTAMP, OP_MAINNET_ECOTONE_TIMESTAMP, OP_MAINNET_FJORD_TIMESTAMP,
    OP_MAINNET_GRANITE_TIMESTAMP, OP_MAINNET_HOLOCENE_TIMESTAMP, OP_MAINNET_ISTHMUS_TIMESTAMP,
    OP_MAINNET_JOVIAN_TIMESTAMP,
};
use alloy_primitives::{address, b256, uint};
use kona_genesis::{
    ChainGenesis, DEFAULT_INTEROP_MESSAGE_EXPIRY_WINDOW, HardForkConfig,
    OP_MAINNET_BASE_FEE_CONFIG, RollupConfig, SystemConfig,
};

/// The [`RollupConfig`] for OP Mainnet.
pub const OP_MAINNET_CONFIG: RollupConfig = RollupConfig {
    genesis: ChainGenesis {
        l1: BlockNumHash {
            hash: b256!("438335a20d98863a4c0c97999eb2481921ccd28553eac6f913af7c12aec04108"),
            number: 17_422_590_u64,
        },
        l2: BlockNumHash {
            hash: b256!("dbf6a80fef073de06add9b0d14026d6e5a86c85f6d102c36d3d8e9cf89c2afd3"),
            number: 105_235_063_u64,
        },
        l2_time: 1_686_068_903_u64,
        system_config: Some(SystemConfig {
            batcher_address: address!("6887246668a3b87f54deb3b94ba47a6f63f32985"),
            overhead: uint!(0xbc_U256),
            scalar: uint!(0xa6fe0_U256),
            gas_limit: 30_000_000_u64,
            base_fee_scalar: None,
            blob_base_fee_scalar: None,
            eip1559_denominator: None,
            eip1559_elasticity: None,
            operator_fee_scalar: None,
            operator_fee_constant: None,
            min_base_fee: None,
            da_footprint_gas_scalar: None,
        }),
    },
    block_time: 2_u64,
    max_sequencer_drift: 600_u64,
    seq_window_size: 3600_u64,
    channel_timeout: 300_u64,
    granite_channel_timeout: 50,
    l1_chain_id: 1_u64,
    l2_chain_id: Chain::optimism_mainnet(),
    chain_op_config: OP_MAINNET_BASE_FEE_CONFIG,
    alt_da_config: None,
    hardforks: HardForkConfig {
        regolith_time: None,
        canyon_time: Some(OP_MAINNET_CANYON_TIMESTAMP),
        delta_time: Some(1_708_560_000_u64),
        ecotone_time: Some(OP_MAINNET_ECOTONE_TIMESTAMP),
        fjord_time: Some(OP_MAINNET_FJORD_TIMESTAMP),
        granite_time: Some(OP_MAINNET_GRANITE_TIMESTAMP),
        holocene_time: Some(OP_MAINNET_HOLOCENE_TIMESTAMP),
        pectra_blob_schedule_time: None,
        isthmus_time: Some(OP_MAINNET_ISTHMUS_TIMESTAMP),
        jovian_time: Some(OP_MAINNET_JOVIAN_TIMESTAMP),
        interop_time: None,
    },
    batch_inbox_address: address!("ff00000000000000000000000000000000000010"),
    deposit_contract_address: address!("beb5fc579115071764c7423a4f12edde41f106ed"),
    l1_system_config_address: address!("229047fed2591dbec1ef1118d64f7af3db9eb290"),
    protocol_versions_address: address!("8062abc286f5e7d9428a0ccb9abd71e50d93b935"),
    superchain_config_address: Some(address!("95703e0982140D16f8ebA6d158FccEde42f04a4C")),
    da_challenge_address: None,
    blobs_enabled_l1_timestamp: None,
    interop_message_expiry_window: DEFAULT_INTEROP_MESSAGE_EXPIRY_WINDOW,
};
