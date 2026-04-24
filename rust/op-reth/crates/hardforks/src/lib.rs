// Pedantic/nursery lints from workspace-level clippy config are largely stylistic;
// reth upstream maintains its own curated lint set. Allow the cosmetic/architectural-cost
// categories here so real issues stay visible.
#![allow(clippy::cast_lossless)]
#![allow(clippy::cast_possible_truncation)]
#![allow(clippy::cast_possible_wrap)]
#![allow(clippy::cast_precision_loss)]
#![allow(clippy::cast_sign_loss)]
#![allow(clippy::default_trait_access)]
#![allow(clippy::doc_markdown)]
#![allow(clippy::elidable_lifetime_names)]
#![allow(clippy::fallible_impl_from)]
#![allow(clippy::float_cmp)]
#![allow(clippy::future_not_send)]
#![allow(clippy::ignore_without_reason)]
#![allow(clippy::ignored_unit_patterns)]
#![allow(clippy::inconsistent_struct_constructor)]
#![allow(clippy::inline_always)]
#![allow(clippy::items_after_statements)]
#![allow(clippy::large_futures)]
#![allow(clippy::large_stack_arrays)]
#![allow(clippy::large_stack_frames)]
#![allow(clippy::manual_let_else)]
#![allow(clippy::map_unwrap_or)]
#![allow(clippy::match_wildcard_for_single_variants)]
#![allow(clippy::mismatching_type_param_order)]
#![allow(clippy::missing_const_for_fn)]
#![allow(clippy::missing_errors_doc)]
#![allow(clippy::missing_fields_in_debug)]
#![allow(clippy::missing_panics_doc)]
#![allow(clippy::must_use_candidate)]
#![allow(clippy::needless_pass_by_value)]
#![allow(clippy::needless_raw_string_hashes)]
#![allow(clippy::non_std_lazy_statics)]
#![allow(clippy::redundant_closure_for_method_calls)]
#![allow(clippy::redundant_pub_crate)]
#![allow(clippy::ref_option)]
#![allow(clippy::return_self_not_must_use)]
#![allow(clippy::semicolon_if_nothing_returned)]
#![allow(clippy::significant_drop_tightening)]
#![allow(clippy::similar_names)]
#![allow(clippy::single_match_else)]
#![allow(clippy::struct_excessive_bools)]
#![allow(clippy::struct_field_names)]
#![allow(clippy::too_long_first_doc_paragraph)]
#![allow(clippy::too_many_lines)]
#![allow(clippy::unchecked_time_subtraction)]
#![allow(clippy::uninlined_format_args)]
#![allow(clippy::unnecessary_semicolon)]
#![allow(clippy::unnecessary_wraps)]
#![allow(clippy::unreadable_literal)]
#![allow(clippy::unused_async)]
#![allow(clippy::unused_self)]
#![allow(clippy::use_self)]
#![allow(clippy::used_underscore_binding)]
#![allow(clippy::wildcard_imports)]

//! OP-Reth hard forks.
//!
//! This defines the [`ChainHardforks`] for certain op chains.
//! It keeps L2 hardforks that correspond to L1 hardforks in sync by defining both at the same
//! activation timestamp, this includes:
//!  - Canyon : Shanghai
//!  - Ecotone : Cancun
//!  - Isthmus : Prague

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(feature = "std"), no_std)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

extern crate alloc;

use alloy_op_hardforks::{
    BASE_MAINNET_JOVIAN_TIMESTAMP, BASE_SEPOLIA_JOVIAN_TIMESTAMP, OP_MAINNET_JOVIAN_TIMESTAMP,
    OP_SEPOLIA_JOVIAN_TIMESTAMP,
};
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

/// Optimism mainnet list of hardforks.
pub static OP_MAINNET_HARDFORKS: LazyLock<ChainHardforks> = LazyLock::new(|| {
    ChainHardforks::new(vec![
        (EthereumHardfork::Frontier.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Homestead.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Tangerine.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::SpuriousDragon.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Byzantium.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Constantinople.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Petersburg.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Istanbul.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::MuirGlacier.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Berlin.boxed(), ForkCondition::Block(3950000)),
        (EthereumHardfork::London.boxed(), ForkCondition::Block(105235063)),
        (EthereumHardfork::ArrowGlacier.boxed(), ForkCondition::Block(105235063)),
        (EthereumHardfork::GrayGlacier.boxed(), ForkCondition::Block(105235063)),
        (
            EthereumHardfork::Paris.boxed(),
            ForkCondition::TTD {
                activation_block_number: 105235063,
                fork_block: Some(105235063),
                total_difficulty: U256::ZERO,
            },
        ),
        (OpHardfork::Bedrock.boxed(), ForkCondition::Block(105235063)),
        (OpHardfork::Regolith.boxed(), ForkCondition::Timestamp(0)),
        (EthereumHardfork::Shanghai.boxed(), ForkCondition::Timestamp(1704992401)),
        (OpHardfork::Canyon.boxed(), ForkCondition::Timestamp(1704992401)),
        (EthereumHardfork::Cancun.boxed(), ForkCondition::Timestamp(1710374401)),
        (OpHardfork::Ecotone.boxed(), ForkCondition::Timestamp(1710374401)),
        (OpHardfork::Fjord.boxed(), ForkCondition::Timestamp(1720627201)),
        (OpHardfork::Granite.boxed(), ForkCondition::Timestamp(1726070401)),
        (OpHardfork::Holocene.boxed(), ForkCondition::Timestamp(1736445601)),
        (EthereumHardfork::Prague.boxed(), ForkCondition::Timestamp(1746806401)),
        (OpHardfork::Isthmus.boxed(), ForkCondition::Timestamp(1746806401)),
        (OpHardfork::Jovian.boxed(), ForkCondition::Timestamp(OP_MAINNET_JOVIAN_TIMESTAMP)),
    ])
});
/// Optimism Sepolia list of hardforks.
pub static OP_SEPOLIA_HARDFORKS: LazyLock<ChainHardforks> = LazyLock::new(|| {
    ChainHardforks::new(vec![
        (EthereumHardfork::Frontier.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Homestead.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Tangerine.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::SpuriousDragon.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Byzantium.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Constantinople.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Petersburg.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Istanbul.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::MuirGlacier.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Berlin.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::London.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::ArrowGlacier.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::GrayGlacier.boxed(), ForkCondition::Block(0)),
        (
            EthereumHardfork::Paris.boxed(),
            ForkCondition::TTD {
                activation_block_number: 0,
                fork_block: Some(0),
                total_difficulty: U256::ZERO,
            },
        ),
        (OpHardfork::Bedrock.boxed(), ForkCondition::Block(0)),
        (OpHardfork::Regolith.boxed(), ForkCondition::Timestamp(0)),
        (EthereumHardfork::Shanghai.boxed(), ForkCondition::Timestamp(1699981200)),
        (OpHardfork::Canyon.boxed(), ForkCondition::Timestamp(1699981200)),
        (EthereumHardfork::Cancun.boxed(), ForkCondition::Timestamp(1708534800)),
        (OpHardfork::Ecotone.boxed(), ForkCondition::Timestamp(1708534800)),
        (OpHardfork::Fjord.boxed(), ForkCondition::Timestamp(1716998400)),
        (OpHardfork::Granite.boxed(), ForkCondition::Timestamp(1723478400)),
        (OpHardfork::Holocene.boxed(), ForkCondition::Timestamp(1732633200)),
        (EthereumHardfork::Prague.boxed(), ForkCondition::Timestamp(1744905600)),
        (OpHardfork::Isthmus.boxed(), ForkCondition::Timestamp(1744905600)),
        (OpHardfork::Jovian.boxed(), ForkCondition::Timestamp(OP_SEPOLIA_JOVIAN_TIMESTAMP)),
    ])
});

/// Base Sepolia list of hardforks.
pub static BASE_SEPOLIA_HARDFORKS: LazyLock<ChainHardforks> = LazyLock::new(|| {
    ChainHardforks::new(vec![
        (EthereumHardfork::Frontier.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Homestead.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Tangerine.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::SpuriousDragon.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Byzantium.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Constantinople.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Petersburg.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Istanbul.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::MuirGlacier.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Berlin.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::London.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::ArrowGlacier.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::GrayGlacier.boxed(), ForkCondition::Block(0)),
        (
            EthereumHardfork::Paris.boxed(),
            ForkCondition::TTD {
                activation_block_number: 0,
                fork_block: Some(0),
                total_difficulty: U256::ZERO,
            },
        ),
        (OpHardfork::Bedrock.boxed(), ForkCondition::Block(0)),
        (OpHardfork::Regolith.boxed(), ForkCondition::Timestamp(0)),
        (EthereumHardfork::Shanghai.boxed(), ForkCondition::Timestamp(1699981200)),
        (OpHardfork::Canyon.boxed(), ForkCondition::Timestamp(1699981200)),
        (EthereumHardfork::Cancun.boxed(), ForkCondition::Timestamp(1708534800)),
        (OpHardfork::Ecotone.boxed(), ForkCondition::Timestamp(1708534800)),
        (OpHardfork::Fjord.boxed(), ForkCondition::Timestamp(1716998400)),
        (OpHardfork::Granite.boxed(), ForkCondition::Timestamp(1723478400)),
        (OpHardfork::Holocene.boxed(), ForkCondition::Timestamp(1732633200)),
        (EthereumHardfork::Prague.boxed(), ForkCondition::Timestamp(1744905600)),
        (OpHardfork::Isthmus.boxed(), ForkCondition::Timestamp(1744905600)),
        (OpHardfork::Jovian.boxed(), ForkCondition::Timestamp(BASE_SEPOLIA_JOVIAN_TIMESTAMP)),
    ])
});

/// Base mainnet list of hardforks.
pub static BASE_MAINNET_HARDFORKS: LazyLock<ChainHardforks> = LazyLock::new(|| {
    ChainHardforks::new(vec![
        (EthereumHardfork::Frontier.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Homestead.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Tangerine.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::SpuriousDragon.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Byzantium.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Constantinople.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Petersburg.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Istanbul.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::MuirGlacier.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::Berlin.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::London.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::ArrowGlacier.boxed(), ForkCondition::Block(0)),
        (EthereumHardfork::GrayGlacier.boxed(), ForkCondition::Block(0)),
        (
            EthereumHardfork::Paris.boxed(),
            ForkCondition::TTD {
                activation_block_number: 0,
                fork_block: Some(0),
                total_difficulty: U256::ZERO,
            },
        ),
        (OpHardfork::Bedrock.boxed(), ForkCondition::Block(0)),
        (OpHardfork::Regolith.boxed(), ForkCondition::Timestamp(0)),
        (EthereumHardfork::Shanghai.boxed(), ForkCondition::Timestamp(1704992401)),
        (OpHardfork::Canyon.boxed(), ForkCondition::Timestamp(1704992401)),
        (EthereumHardfork::Cancun.boxed(), ForkCondition::Timestamp(1710374401)),
        (OpHardfork::Ecotone.boxed(), ForkCondition::Timestamp(1710374401)),
        (OpHardfork::Fjord.boxed(), ForkCondition::Timestamp(1720627201)),
        (OpHardfork::Granite.boxed(), ForkCondition::Timestamp(1726070401)),
        (OpHardfork::Holocene.boxed(), ForkCondition::Timestamp(1736445601)),
        (EthereumHardfork::Prague.boxed(), ForkCondition::Timestamp(1746806401)),
        (OpHardfork::Isthmus.boxed(), ForkCondition::Timestamp(1746806401)),
        (OpHardfork::Jovian.boxed(), ForkCondition::Timestamp(BASE_MAINNET_JOVIAN_TIMESTAMP)),
    ])
});
