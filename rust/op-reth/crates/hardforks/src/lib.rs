//! OP-Reth hard forks.
//!
//! This defines the [`ChainHardforks`] for certain op chains.
//! It keeps L2 hardforks that correspond to L1 hardforks in sync by defining both at the same
//! activation timestamp.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(feature = "std"), no_std)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

extern crate alloc;

// Re-export alloy-op-hardforks types.
pub use alloy_op_hardforks::{OpHardfork, OpHardforks};

use alloc::vec;
use alloy_primitives::U256;
use once_cell::sync::Lazy as LazyLock;
use reth_ethereum_forks::{ChainHardforks, EthereumHardfork, ForkCondition, Hardfork};

/// Dev hardforks
pub static DEV_HARDFORKS: LazyLock<ChainHardforks> = LazyLock::new(|| {
    ChainHardforks::new(vec![
        (EthereumHardfork::Frontier.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Homestead.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Dao.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Tangerine.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::SpuriousDragon.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Byzantium.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Constantinople.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Petersburg.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Istanbul.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Berlin.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::London.boxed(), ForkCondition::Block(0)),
        (
            EthereumHardfork::Paris.boxed(),
            ForkCondition::TTD {
                activation_block_number: 0,
                fork_block: None,
                total_difficulty: U256::ZERO,
            },
        ),
        (OpHardfork::Bedrock.boxed(), ForkCondition::Block(0)),
        (OpHardfork::Regolith.boxed(), ForkCondition::Timestamp(0)),
        (EthereumHardfork::Shanghai.boxed(), ForkCondition::Timestamp(0)),
        (OpHardfork::Canyon.boxed(), ForkCondition::Timestamp(0)),
        (EthereumHardfork::Cancun.boxed(), ForkCondition::Timestamp(0)),
        (OpHardfork::Ecotone.boxed(), ForkCondition::Timestamp(0)),
        (OpHardfork::Fjord.boxed(), ForkCondition::Timestamp(0)),
        (OpHardfork::Granite.boxed(), ForkCondition::Timestamp(0)),
        (OpHardfork::Holocene.boxed(), ForkCondition::Timestamp(0)),
        (EthereumHardfork::Prague.boxed(), ForkCondition::Timestamp(0)),
        (OpHardfork::Isthmus.boxed(), ForkCondition::Timestamp(0)),
        (OpHardfork::Jovian.boxed(), ForkCondition::Timestamp(0)),
        (OpHardfork::Karst.boxed(), ForkCondition::Timestamp(0)),
    ])
});

/// Canonical activation order of OP-stack hardforks.
///
/// Building an `OpChainSpec` from a genesis orders that chain's forks by this sequence; each fork's
/// activation block/timestamp comes from the chain's own config, not from here. Keep this in sync
/// when introducing a new hardfork.
pub static OP_HARDFORK_ORDER: &[&dyn Hardfork] = &[
    &EthereumHardfork::Frontier,
    &EthereumHardfork::Homestead,
    &EthereumHardfork::Tangerine,
    &EthereumHardfork::SpuriousDragon,
    &EthereumHardfork::Byzantium,
    &EthereumHardfork::Constantinople,
    &EthereumHardfork::Petersburg,
    &EthereumHardfork::Istanbul,
    &EthereumHardfork::MuirGlacier,
    &EthereumHardfork::Berlin,
    &EthereumHardfork::London,
    &EthereumHardfork::ArrowGlacier,
    &EthereumHardfork::GrayGlacier,
    &EthereumHardfork::Paris,
    &OpHardfork::Bedrock,
    &OpHardfork::Regolith,
    &EthereumHardfork::Shanghai,
    &OpHardfork::Canyon,
    &EthereumHardfork::Cancun,
    &OpHardfork::Ecotone,
    &OpHardfork::Fjord,
    &OpHardfork::Granite,
    &OpHardfork::Holocene,
    &EthereumHardfork::Prague,
    &OpHardfork::Isthmus,
    &OpHardfork::Jovian,
    &EthereumHardfork::Osaka,
    &OpHardfork::Karst,
    &OpHardfork::Lagoon,
];
