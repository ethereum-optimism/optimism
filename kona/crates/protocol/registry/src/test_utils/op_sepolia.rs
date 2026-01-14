//! OP Sepolia Rollup Config.

use alloy_chains::Chain;
use alloy_eips::BlockNumHash;
use alloy_op_hardforks::{
    OP_SEPOLIA_CANYON_TIMESTAMP, OP_SEPOLIA_ECOTONE_TIMESTAMP, OP_SEPOLIA_FJORD_TIMESTAMP,
    OP_SEPOLIA_GRANITE_TIMESTAMP, OP_SEPOLIA_HOLOCENE_TIMESTAMP, OP_SEPOLIA_ISTHMUS_TIMESTAMP,
    OP_SEPOLIA_JOVIAN_TIMESTAMP,
};
use alloy_primitives::{address, b256, uint};
use kona_genesis::{
    ChainGenesis, DEFAULT_INTEROP_MESSAGE_EXPIRY_WINDOW, HardForkConfig,
    OP_SEPOLIA_BASE_FEE_CONFIG, RollupConfig, SystemConfig,
};

/// The [RollupConfig] for OP Sepolia.
pub const OP_SEPOLIA_CONFIG: RollupConfig = RollupConfig {
    genesis: ChainGenesis {
        l1: BlockNumHash {
            hash: b256!("48f520cf4ddaf34c8336e6e490632ea3cf1e5e93b0b2bc6e917557e31845371b"),
            number: 4071408,
        },
        l2: BlockNumHash {
            hash: b256!("102de6ffb001480cc9b8b548fd05c34cd4f46ae4aa91759393db90ea0409887d"),
            number: 0,
        },
        l2_time: 1691802540,
        system_config: Some(SystemConfig {
            batcher_address: address!("8f23bb38f531600e5d8fddaaec41f13fab46e98c"),
            overhead: uint!(0xbc_U256),
            scalar: uint!(0xa6fe0_U256),
            gas_limit: 30_000_000,
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
    block_time: 2,
    max_sequencer_drift: 600,
    seq_window_size: 3600,
    channel_timeout: 300,
    granite_channel_timeout: 50,
    l1_chain_id: 11155111,
    l2_chain_id: Chain::optimism_sepolia(),
    chain_op_config: OP_SEPOLIA_BASE_FEE_CONFIG,
    alt_da_config: None,
    hardforks: HardForkConfig {
        regolith_time: None,
        canyon_time: Some(OP_SEPOLIA_CANYON_TIMESTAMP),
        delta_time: Some(1703203200),
        ecotone_time: Some(OP_SEPOLIA_ECOTONE_TIMESTAMP),
        fjord_time: Some(OP_SEPOLIA_FJORD_TIMESTAMP),
        granite_time: Some(OP_SEPOLIA_GRANITE_TIMESTAMP),
        holocene_time: Some(OP_SEPOLIA_HOLOCENE_TIMESTAMP),
        pectra_blob_schedule_time: Some(1742486400),
        isthmus_time: Some(OP_SEPOLIA_ISTHMUS_TIMESTAMP),
        jovian_time: Some(OP_SEPOLIA_JOVIAN_TIMESTAMP),
        interop_time: None,
    },
    batch_inbox_address: address!("ff00000000000000000000000000000011155420"),
    deposit_contract_address: address!("16fc5058f25648194471939df75cf27a2fdc48bc"),
    l1_system_config_address: address!("034edd2a225f7f429a63e0f1d2084b9e0a93b538"),
    protocol_versions_address: address!("79add5713b383daa0a138d3c4780c7a1804a8090"),
    superchain_config_address: Some(address!("C2Be75506d5724086DEB7245bd260Cc9753911Be")),
    da_challenge_address: None,
    blobs_enabled_l1_timestamp: None,
    interop_message_expiry_window: DEFAULT_INTEROP_MESSAGE_EXPIRY_WINDOW,
};
