#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(feature = "std"), no_std)]
// Pedantic/nursery lints allowed crate-wide. Each is either a style preference
// that would require hundreds of mechanical rewrites, or a lint inherent to the
// low-level nature of this crate (byte-offset casts, async trait futures, etc.).
#![allow(
    // Byte-offset / length arithmetic routinely crosses u64<->usize.
    clippy::cast_possible_truncation,
    clippy::cast_sign_loss,
    clippy::cast_lossless,
    clippy::cast_precision_loss,
    clippy::cast_possible_wrap,
    // Suggesting `#[must_use]` on every getter / ctor is noise; the public API
    // is stable and documented.
    clippy::must_use_candidate,
    clippy::return_self_not_must_use,
    // `# Errors` / `# Panics` sections would be added to hundreds of public fns.
    clippy::missing_errors_doc,
    clippy::missing_panics_doc,
    // `Default::default()` spread pattern is idiomatic here.
    clippy::default_trait_access,
    // Closures forwarding a single arg are used for readability.
    clippy::redundant_closure_for_method_calls,
    // `map(f).unwrap_or*` forms are used for clarity over `map_or`.
    clippy::option_if_let_else,
    // Async trait methods where `Send`ness depends on generic `C`.
    clippy::future_not_send,
    // Domain-specific field/var naming overlap.
    clippy::similar_names,
    clippy::struct_field_names,
    clippy::module_name_repetitions,
    // Cheap-Copy args consumed by async fns.
    clippy::needless_pass_by_value,
    // Enum wildcard matches in tests and fixtures.
    clippy::wildcard_enum_match_arm,
    // `pub use super::*;` style.
    clippy::wildcard_imports,
    // Long internal fns are kept inline for locality.
    clippy::too_many_lines,
    // Workspace allows inherited: these lints fire under `-W clippy::pedantic/nursery`
    // but are acceptable in this crate.
    clippy::redundant_pub_crate,
    clippy::significant_drop_tightening,
    clippy::too_long_first_doc_paragraph,
    clippy::doc_markdown,
    clippy::unreadable_literal,
    clippy::items_after_statements,
    clippy::missing_const_for_fn,
    clippy::map_unwrap_or,
    clippy::needless_collect,
    clippy::large_stack_arrays,
    clippy::large_stack_frames,
    clippy::large_futures,
    clippy::unused_self,
    clippy::unused_async,
    clippy::bool_to_int_with_if,
    clippy::single_match_else,
    clippy::manual_let_else,
    clippy::redundant_else,
    clippy::ignored_unit_patterns,
    clippy::unnecessary_debug_formatting,
    clippy::unnecessary_semicolon,
    clippy::from_over_into,
    clippy::iter_not_returning_iterator,
    clippy::unnecessary_wraps,
    clippy::redundant_closure,
    clippy::must_use_unit,
    clippy::stable_sort_primitive,
    clippy::match_wildcard_for_single_variants,
    clippy::ref_option,
    clippy::needless_raw_string_hashes,
    clippy::elidable_lifetime_names,
    clippy::inline_always,
    clippy::non_std_lazy_statics,
    clippy::ignore_without_reason,
    clippy::should_panic_without_expect,
    clippy::ptr_as_ptr,
    clippy::range_plus_one,
    clippy::used_underscore_binding,
    clippy::zero_sized_map_values,
    clippy::implicit_clone,
    clippy::ip_constant,
    clippy::needless_continue,
    clippy::or_fun_call,
    clippy::semicolon_if_nothing_returned,
    clippy::let_underscore_future,
    clippy::manual_assert,
)]

extern crate alloc;

pub use alloy_primitives::map::HashMap;
use kona_genesis::L1ChainConfig;
pub use kona_genesis::{Chain, ChainConfig, ChainList, RollupConfig};

pub mod superchain;
pub use superchain::Registry;

/// L1 chain configurations.
pub mod l1;
pub use l1::L1Config;

#[cfg(test)]
pub mod test_utils;

lazy_static::lazy_static! {
    /// Private initializer that loads the superchain configurations.
    static ref _INIT: Registry = Registry::from_chain_list();

    /// Chain configurations exported from the registry
    pub static ref CHAINS: ChainList = _INIT.chain_list.clone();

    /// OP Chain configurations exported from the registry
    pub static ref OPCHAINS: HashMap<u64, ChainConfig> = _INIT.op_chains.clone();

    /// Rollup configurations exported from the registry
    pub static ref ROLLUP_CONFIGS: HashMap<u64, RollupConfig> = _INIT.rollup_configs.clone();

    /// L1 chain configurations exported from the registry
    /// Note: the l1 chain configurations are not exported from the superchain registry but rather from a genesis dump file.
    pub static ref L1_CONFIGS: HashMap<u64, L1ChainConfig> = _INIT.l1_configs.clone();
}

/// Returns a [`RollupConfig`] by its identifier.
#[must_use]
pub fn scr_rollup_config_by_ident(ident: &str) -> Option<&RollupConfig> {
    let chain_id = CHAINS.get_chain_by_ident(ident)?.chain_id;
    ROLLUP_CONFIGS.get(&chain_id)
}

/// Returns a [`RollupConfig`] by its identifier.
#[must_use]
pub fn scr_rollup_config_by_alloy_ident(chain: &alloy_chains::Chain) -> Option<&RollupConfig> {
    ROLLUP_CONFIGS.get(&chain.id())
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_chains::Chain as AlloyChain;
    use alloy_hardforks::{
        holesky::{HOLESKY_BPO1_TIMESTAMP, HOLESKY_BPO2_TIMESTAMP},
        sepolia::{SEPOLIA_BPO1_TIMESTAMP, SEPOLIA_BPO2_TIMESTAMP},
    };
    use alloy_op_hardforks::{
        BASE_MAINNET_JOVIAN_TIMESTAMP, BASE_SEPOLIA_JOVIAN_TIMESTAMP, OP_MAINNET_JOVIAN_TIMESTAMP,
        OP_SEPOLIA_JOVIAN_TIMESTAMP,
    };

    #[test]
    fn test_hardcoded_rollup_configs() {
        let test_cases = [
            (10, test_utils::OP_MAINNET_CONFIG),
            (8453, test_utils::BASE_MAINNET_CONFIG),
            (11_155_420, test_utils::OP_SEPOLIA_CONFIG),
            (84532, test_utils::BASE_SEPOLIA_CONFIG),
        ]
        .to_vec();

        for (chain_id, expected) in test_cases {
            let derived = super::ROLLUP_CONFIGS.get(&chain_id).unwrap();
            assert_eq!(expected, *derived);
        }
    }

    #[test]
    fn test_chain_by_ident() {
        const ALLOY_BASE: AlloyChain = AlloyChain::base_mainnet();

        let chain_by_ident = CHAINS.get_chain_by_ident("mainnet/base").unwrap();
        let chain_by_alloy_ident = CHAINS.get_chain_by_alloy_ident(&ALLOY_BASE).unwrap();
        let chain_by_id = CHAINS.get_chain_by_id(8453).unwrap();

        assert_eq!(chain_by_ident, chain_by_id);
        assert_eq!(chain_by_alloy_ident, chain_by_id);
    }

    #[test]
    fn test_rollup_config_by_ident() {
        const ALLOY_BASE: AlloyChain = AlloyChain::base_mainnet();

        let rollup_config_by_ident = scr_rollup_config_by_ident("mainnet/base").unwrap();
        let rollup_config_by_alloy_ident = scr_rollup_config_by_alloy_ident(&ALLOY_BASE).unwrap();
        let rollup_config_by_id = ROLLUP_CONFIGS.get(&8453).unwrap();

        assert_eq!(rollup_config_by_ident, rollup_config_by_id);
        assert_eq!(rollup_config_by_alloy_ident, rollup_config_by_id);
    }

    #[test]
    fn test_jovian_timestamps() {
        let base_mainnet_config_by_ident = scr_rollup_config_by_ident("mainnet/base").unwrap();
        assert_eq!(
            base_mainnet_config_by_ident.hardforks.jovian_time,
            Some(BASE_MAINNET_JOVIAN_TIMESTAMP)
        );

        let base_sepolia_config_by_ident = scr_rollup_config_by_ident("sepolia/base").unwrap();
        assert_eq!(
            base_sepolia_config_by_ident.hardforks.jovian_time,
            Some(BASE_SEPOLIA_JOVIAN_TIMESTAMP)
        );

        let op_mainnet_config_by_ident = scr_rollup_config_by_ident("mainnet/op").unwrap();
        assert_eq!(
            op_mainnet_config_by_ident.hardforks.jovian_time,
            Some(OP_MAINNET_JOVIAN_TIMESTAMP)
        );

        let op_sepolia_config_by_ident = scr_rollup_config_by_ident("sepolia/op").unwrap();
        assert_eq!(
            op_sepolia_config_by_ident.hardforks.jovian_time,
            Some(OP_SEPOLIA_JOVIAN_TIMESTAMP)
        );
    }

    #[test]
    fn test_bpo_timestamps() {
        let sepolia_config = L1_CONFIGS.get(&11_155_111).unwrap();
        assert_eq!(sepolia_config.bpo1_time, Some(SEPOLIA_BPO1_TIMESTAMP));
        assert_eq!(sepolia_config.bpo2_time, Some(SEPOLIA_BPO2_TIMESTAMP));

        let holesky_config = L1_CONFIGS.get(&17000).unwrap();
        assert_eq!(holesky_config.bpo1_time, Some(HOLESKY_BPO1_TIMESTAMP));
        assert_eq!(holesky_config.bpo2_time, Some(HOLESKY_BPO2_TIMESTAMP));
    }

    const CUSTOM_CONFIGS_TEST_ENABLED: Option<&str> = option_env!("KONA_CUSTOM_CONFIGS_TEST");
    const CUSTOM_CONFIGS: Option<&str> = option_env!("KONA_CUSTOM_CONFIGS");
    const CUSTOM_CONFIGS_DIR: Option<&str> = option_env!("KONA_CUSTOM_CONFIGS_DIR");

    #[test]
    fn custom_chain_is_loaded_when_enabled() {
        if CUSTOM_CONFIGS_TEST_ENABLED != Some("true") {
            return;
        };
        assert!(
            CUSTOM_CONFIGS == Some("true"),
            "KONA_CUSTOM_CONFIGS is required when KONA_CUSTOM_CONFIGS_TEST is set"
        );
        assert!(
            CUSTOM_CONFIGS_DIR.is_some(),
            "KONA_CUSTOM_CONFIGS_DIR is required when KONA_CUSTOM_CONFIGS_TEST is set"
        );

        let test1_chain_id = 123_999_119;
        let test2_chain_id = 223_999_119;
        let test1_ident = "test1/testnet";
        let test2_ident = "test2/testnet";

        let chain1 = CHAINS
            .get_chain_by_ident(test1_ident)
            .unwrap_or_else(|| panic!("custom chain `{test1_ident}` missing"));
        assert_eq!(chain1.chain_id, test1_chain_id);
        let chain2 = CHAINS
            .get_chain_by_ident(test2_ident)
            .unwrap_or_else(|| panic!("custom chain `{test2_ident}` missing"));
        assert_eq!(chain2.chain_id, test2_chain_id);

        assert!(
            OPCHAINS.contains_key(&test1_chain_id),
            "chain config missing for {test1_chain_id}"
        );
        assert!(
            ROLLUP_CONFIGS.contains_key(&test1_chain_id),
            "rollup config missing for {test1_chain_id}"
        );
        assert!(
            OPCHAINS.contains_key(&test2_chain_id),
            "chain config missing for {test2_chain_id}"
        );
        assert!(
            ROLLUP_CONFIGS.contains_key(&test2_chain_id),
            "rollup config missing for {test2_chain_id}"
        );
    }
}
